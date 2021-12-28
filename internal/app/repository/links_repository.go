package repository

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/zaz600/go-musthave-shortener/internal/config"
	"github.com/zaz600/go-musthave-shortener/internal/random"
)

type LinkEntity struct {
	ID            string `json:"id"`
	OriginalURL   string `json:"original_url"`
	UID           string `json:"uid,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func NewLinkEntity(originalURL string, uid string) LinkEntity {
	return LinkEntity{
		ID:          random.String(8),
		OriginalURL: originalURL,
		UID:         uid,
	}
}

type LinksRepository interface {
	// Get достает по linkID из репозитория информацию по сокращенной ссылке LinkEntity
	Get(ctx context.Context, linkID string) (LinkEntity, error)

	// PutIfAbsent сохраняет в БД длинную ссылку, если такой там еще нет.
	// Если длинная ссылка есть в БД, выбрасывает исключение LinkExistsError с идентификатором ее короткой ссылки.
	PutIfAbsent(ctx context.Context, linkEntity LinkEntity) (LinkEntity, error)

	// PutBatch сохраняет в хранилище список сокращенных ссылок. Все ссылки записываются в одной транзакции.
	PutBatch(ctx context.Context, linkEntities []LinkEntity) error

	// Count возвращает количество записей в репозитории.
	Count(ctx context.Context) (int, error)

	// FindLinksByUID возвращает ссылки по идентификатору пользователя
	FindLinksByUID(ctx context.Context, uid string) ([]LinkEntity, error)

	// Status статус подключения к хранилищу
	Status(ctx context.Context) error

	// Close закрывает, все, что надо закрыть
	Close(ctx context.Context) error
}

func NewRepository(ctx context.Context, cfg *config.ShortenConfig) (LinksRepository, error) {
	var repo LinksRepository
	var err error
	switch cfg.GetRepositoryType() {
	case config.FileRepo:
		log.Info().Msgf("FileRepository %s", cfg.FileStoragePath)
		repo, err = NewFileLinksRepository(ctx, cfg.FileStoragePath)
		if err != nil {
			return nil, err
		}
	case config.DatabaseRepo:
		log.Info().Msg("DatabaseRepo")
		repo, err = NewPgLinksRepository(ctx, cfg.DatabaseDSN)
		if err != nil {
			return nil, err
		}
	default:
		log.Info().Msg("MemoryRepository")
		repo = NewInMemoryLinksRepository(nil)
	}

	return repo, nil
}
