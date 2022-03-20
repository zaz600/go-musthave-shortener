package entity

import (
	"github.com/zaz600/go-musthave-shortener/internal/pkg/random"
)

// LinkEntity ссылка, которую сократили
type LinkEntity struct {
	// ID идентификатор в БД
	ID string `json:"id"`
	// OriginalURL оригинальная сокращаемая ссылка
	OriginalURL string `json:"original_url"`
	// UID пользователь, который сократил ссылку
	UID string `json:"uid,omitempty"`
	// CorrelationID внешний идентификатор ссылки, передаваемый через API
	CorrelationID string `json:"correlation_id,omitempty"`
	// Removed признак удаления ссылки. Нет ручек, которым нужен был бы этот признак
	Removed bool `json:"-"`
}

// NewLinkEntity -
func NewLinkEntity(originalURL string, uid string) LinkEntity {
	return LinkEntity{
		ID:          random.String(8),
		OriginalURL: originalURL,
		UID:         uid,
	}
}

// IsOwnedByUserAndExists возвращает true,
// если ссылка принадлежит указанному пользователю и она не удалена
func (e LinkEntity) IsOwnedByUserAndExists(uid string) bool {
	return e.IsOwnedByUser(uid) && !e.Removed
}

// IsOwnedByUser возвращает true, если ссылка принадлежит указанному пользователю.
// Признак удаления ссылки игнорируется
func (e LinkEntity) IsOwnedByUser(uid string) bool {
	return e.UID == uid
}
