package shortener

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/zaz600/go-musthave-shortener/internal/entity"
	"github.com/zaz600/go-musthave-shortener/internal/infrastructure/repository"
	"github.com/zaz600/go-musthave-shortener/internal/service/batch"
)

// Service сервис сокращения ссылок
type Service struct {
	*chi.Mux
	// BaseURL базовый адрес для сокращения ссылок - {BaseURL}/{shortID}
	baseURL string
	// linksRepository репозиторий для работы с хранилищем сокращенных ссылок
	linksRepository repository.LinksRepository
	// linkRemoveCh канал для отправки запросов на асинхронное удаление ссылок
	linkRemoveCh chan<- removeUserLinksRequest
}

func NewService(baseURL string, opts ...Option) *Service {
	s := &Service{
		Mux:             chi.NewRouter(),
		baseURL:         baseURL,
		linksRepository: nil,
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			log.Panic().Err(err).Msg("")
		}
	}

	if s.linksRepository == nil {
		s.linksRepository = repository.NewInMemoryLinksRepository(context.Background(), nil)
	}
	s.linkRemoveCh = s.startRemoveLinksWorkers(context.Background(), 10)
	return s
}

// ShortURL формирует полный адрес сокращенной ссылки по ее идентификатору
func (s *Service) ShortURL(linkID string) string {
	parsedURL, _ := url.Parse(s.baseURL)
	parsedURL.Path = linkID
	return parsedURL.String()
}

// RemoveLinks запрос на удаление ссылок.
// Фактически ссылки не удаляются из БД,
// а помечаются как удаленные и перестают быть доступными в других методах.
func (s *Service) RemoveLinks(removeIDs []string, uid string) {
	s.linkRemoveCh <- removeUserLinksRequest{
		linkIDs: removeIDs,
		uid:     uid,
	}
}

// Shutdown должен вызываться при остановке приложения
func (s *Service) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return s.linksRepository.Close(ctx)
}

// ShortenURL сохраняет в хранилище запись о сокращенной ссылке
func (s *Service) ShortenURL(ctx context.Context, linkEntity entity.LinkEntity) (entity.LinkEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return s.linksRepository.PutIfAbsent(ctx, linkEntity)
}

// GetUserLinks извлекает ссылки, сокращенные пользователем по его идентификатору
func (s *Service) GetUserLinks(ctx context.Context, uid string) ([]entity.LinkEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return s.linksRepository.FindLinksByUID(ctx, uid)
}

// Get возвращает информацию о сокращенной ссылке по ее короткому идентификатору
func (s *Service) Get(ctx context.Context, linkID string) (*entity.LinkEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return s.linksRepository.Get(ctx, linkID)
}

// Count возвращает количество ссылок в хранилище
func (s *Service) Count(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return s.linksRepository.Count(ctx)
}

// Status -
func (s *Service) Status(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return s.linksRepository.Status(ctx)
}

func (s *Service) NewBatchService(batchSize int) *batch.Service {
	return batch.NewBatchService(batchSize, s.linksRepository)
}

// startRemoveLinksWorkers запуск воркеров для асинхронного удаления ссылок в хранилище
func (s *Service) startRemoveLinksWorkers(ctx context.Context, count int) chan<- removeUserLinksRequest {
	linkCh := make(chan removeUserLinksRequest, count*2)
	for i := 0; i < count; i++ {
		workerID := fmt.Sprintf("RemoveLinksWorker#%d", i+1)
		go func() {
			log.Info().Str("worker", workerID).Msg("start remove links worker")
			for {
				select {
				case <-ctx.Done():
					log.Info().Str("worker", workerID).Msg("shutdown remove links worker...")
					return
				case req := <-linkCh:
					func() {
						ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
						defer cancel()

						err := s.linksRepository.DeleteLinksByUID(ctx, req.uid, req.linkIDs...)
						if err != nil {
							log.Warn().Str("worker", workerID).Err(err).Strs("ids", req.linkIDs).Str("uid", req.uid).Msg("error delete user links")
							return
						}
						log.Info().Str("worker", workerID).Str("uid", req.uid).Strs("ids", req.linkIDs).Msg("urls deleted")
					}()
				}
			}
		}()
	}
	return linkCh
}

// IsValidURL проверяет адрес на пригодность для сохранения в БД
func IsValidURL(value string) bool {
	if value == "" {
		return false
	}
	_, err := url.Parse(value)

	return err == nil
}
