package repository

import "github.com/zaz600/go-musthave-shortener/internal/random"

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
	Get(linkID string) (LinkEntity, error)
	Put(linkEntity LinkEntity) (string, error)
	PutBatch(linkEntities []LinkEntity) error
	Count() (int, error)
	FindLinksByUID(uid string) ([]LinkEntity, error)
	Status() error
	Close() error
}
