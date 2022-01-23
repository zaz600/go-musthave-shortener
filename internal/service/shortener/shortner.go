package shortener

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/zaz600/go-musthave-shortener/internal/infrastructure/repository"
)

type Service struct {
	*chi.Mux
	baseURL         string
	linksRepository repository.LinksRepository
	linkRemoveCh    chan<- removeUserLinksRequest
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

func (s *Service) ShortURL(linkID string) string {
	parsedURL, _ := url.Parse(s.baseURL)
	parsedURL.Path = linkID
	return parsedURL.String()
}

func (s *Service) RemoveLinks(removeIDs []string, uid string) {
	s.linkRemoveCh <- removeUserLinksRequest{
		linkIDs: removeIDs,
		uid:     uid,
	}
}

func (s *Service) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return s.linksRepository.Close(ctx)
}

// Deprecated: GetRepo нужна на момент рефакторинга
func (s *Service) GetRepo() repository.LinksRepository {
	return s.linksRepository
}

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
