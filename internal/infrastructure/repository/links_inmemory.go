package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/zaz600/go-musthave-shortener/internal/entity"
)

type InMemoryLinksRepository struct {
	mu *sync.RWMutex
	db map[string]entity.LinkEntity
}

func NewInMemoryLinksRepository(_ context.Context, db map[string]entity.LinkEntity) InMemoryLinksRepository {
	if db == nil {
		db = make(map[string]entity.LinkEntity)
	}
	return InMemoryLinksRepository{
		mu: &sync.RWMutex{},
		db: db,
	}
}

// Get достает по linkID из репозитория информацию по сокращенной ссылке entity.LinkEntity
func (m InMemoryLinksRepository) Get(_ context.Context, linkID string) (*entity.LinkEntity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if e, ok := m.db[linkID]; ok {
		return &e, nil
	}
	return nil, fmt.Errorf("link with id '%s' not found", linkID)
}

// PutIfAbsent сохраняет в БД длинную ссылку, если такой там еще нет.
// Если длинная ссылка есть в БД, выбрасывает исключение LinkExistsError с идентификатором ее короткой ссылки.
func (m InMemoryLinksRepository) PutIfAbsent(_ context.Context, linkEntity entity.LinkEntity) (entity.LinkEntity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, e := range m.db {
		if e.OriginalURL == linkEntity.OriginalURL {
			return entity.LinkEntity{}, NewLinkExistsError(e.ID)
		}
	}

	m.db[linkEntity.ID] = linkEntity
	return linkEntity, nil
}

// PutBatch сохраняет в хранилище список сокращенных ссылок. Все ссылки записываются в одной транзакции.
func (m InMemoryLinksRepository) PutBatch(_ context.Context, linkEntities []entity.LinkEntity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range linkEntities {
		m.db[e.ID] = e
	}
	return nil
}

// Count возвращает количество записей в репозитории.
func (m InMemoryLinksRepository) Count(_ context.Context) (int, error) {
	return len(m.db), nil
}

// FindLinksByUID возвращает ссылки по идентификатору пользователя
func (m InMemoryLinksRepository) FindLinksByUID(_ context.Context, uid string) ([]entity.LinkEntity, error) {
	result := make([]entity.LinkEntity, 0, 100)
	for _, e := range m.db {
		if e.IsOwnedByUserAndExists(uid) {
			result = append(result, e)
		}
	}
	return result, nil
}

// DeleteLinksByUID удаляет ссылки пользователя
func (m InMemoryLinksRepository) DeleteLinksByUID(_ context.Context, uid string, linkIDs ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range linkIDs {
		e, ok := m.db[id]
		if !ok {
			// такого айди не в хранилище, пока просто его пропустим
			continue
		}
		if !e.IsOwnedByUser(uid) {
			// тут возможно надо обработать, что пытаются удалить чужой линк, но пока просто его пропустим
			continue
		}
		e.Removed = true
		m.db[id] = e
	}
	return nil
}

// Status статус подключения к хранилищу
func (m InMemoryLinksRepository) Status(_ context.Context) error {
	return nil
}

// Close закрывает, все, что надо закрыть
func (m InMemoryLinksRepository) Close(_ context.Context) error {
	return nil
}
