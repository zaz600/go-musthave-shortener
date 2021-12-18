package repository

import "github.com/zaz600/go-musthave-shortener/internal/random"

type RepoType int

const (
	MemoryRepo RepoType = iota
	FileRepo
	DatabaseRepo
)

type LinkEntity struct {
	ID      string `json:"id"`
	LongURL string `json:"long_url"`
	UID     string `json:"uid,omitempty"`
}

func NewLinkEntity(longURL string, uid string) LinkEntity {
	return LinkEntity{
		ID:      random.String(8),
		LongURL: longURL,
		UID:     uid,
	}
}

type LinksRepository interface {
	Get(linkID string) (LinkEntity, error)
	Put(linkEntity LinkEntity) (string, error)
	Count() int
	FindLinksByUID(uuid string) []LinkEntity
	Status() error
	Close() error
}
