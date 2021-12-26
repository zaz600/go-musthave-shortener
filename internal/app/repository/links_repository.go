package repository

import (
	"context"

	"github.com/zaz600/go-musthave-shortener/internal/random"
)

type RepoType int

const (
	MemoryRepo RepoType = iota
	FileRepo
	DatabaseRepo
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
	Get(ctx context.Context, linkID string) (LinkEntity, error)
	Put(ctx context.Context, linkEntity LinkEntity) (string, error)
	PutBatch(ctx context.Context, linkEntities []LinkEntity) error
	Count(ctx context.Context) (int, error)
	FindLinksByUID(ctx context.Context, uid string) ([]LinkEntity, error)
	Status(ctx context.Context) error
	Close(ctx context.Context) error
}
