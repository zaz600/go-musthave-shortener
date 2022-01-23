package entity

import "github.com/zaz600/go-musthave-shortener/internal/random"

type LinkEntity struct {
	ID            string `json:"id"`
	OriginalURL   string `json:"original_url"`
	UID           string `json:"uid,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	Removed       bool   `json:"-"`
}

func NewLinkEntity(originalURL string, uid string) LinkEntity {
	return LinkEntity{
		ID:          random.String(8),
		OriginalURL: originalURL,
		UID:         uid,
	}
}

func (e LinkEntity) IsOwnedByUserAndExists(uid string) bool {
	return e.IsOwnedByUser(uid) && !e.Removed
}

func (e LinkEntity) IsOwnedByUser(uid string) bool {
	return e.UID == uid
}
