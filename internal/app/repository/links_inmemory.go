package repository

import (
	"context"
	"fmt"
	"sync"
)

type InMemoryLinksRepository struct {
	mu *sync.RWMutex
	db map[string]LinkEntity
}

func NewInMemoryLinksRepository(ctx context.Context, db map[string]LinkEntity) InMemoryLinksRepository {
	if db == nil {
		db = make(map[string]LinkEntity)
	}
	return InMemoryLinksRepository{
		mu: &sync.RWMutex{},
		db: db,
	}
}

// Get достает по linkID из репозитория информацию по сокращенной ссылке LinkEntity
func (m InMemoryLinksRepository) Get(_ context.Context, linkID string) (LinkEntity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entity, ok := m.db[linkID]; ok {
		return entity, nil
	}
	return LinkEntity{}, fmt.Errorf("link with id '%s' not found", linkID)
}

// PutIfAbsent сохраняет в БД длинную ссылку, если такой там еще нет.
// Если длинная ссылка есть в БД, выбрасывает исключение LinkExistsError с идентификатором ее короткой ссылки.
func (m InMemoryLinksRepository) PutIfAbsent(_ context.Context, linkEntity LinkEntity) (LinkEntity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, entity := range m.db {
		if entity.OriginalURL == linkEntity.OriginalURL {
			return LinkEntity{}, NewLinkExistsError(entity.ID)
		}
	}

	m.db[linkEntity.ID] = linkEntity
	return linkEntity, nil
}

// PutBatch сохраняет в хранилище список сокращенных ссылок. Все ссылки записываются в одной транзакции.
func (m InMemoryLinksRepository) PutBatch(_ context.Context, linkEntities []LinkEntity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, linkEntity := range linkEntities {
		m.db[linkEntity.ID] = linkEntity
	}
	return nil
}

// Count возвращает количество записей в репозитории.
func (m InMemoryLinksRepository) Count(_ context.Context) (int, error) {
	return len(m.db), nil
}

// FindLinksByUID возвращает ссылки по идентификатору пользователя
func (m InMemoryLinksRepository) FindLinksByUID(_ context.Context, uid string) ([]LinkEntity, error) {
	result := make([]LinkEntity, 0, 100)
	for _, entity := range m.db {
		if entity.UID == uid && !entity.Removed {
			result = append(result, entity)
		}
	}
	return result, nil
}

// DeleteLinksByUID удаляет ссылки пользователя
func (m InMemoryLinksRepository) DeleteLinksByUID(ctx context.Context, uid string, linkIDs ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range linkIDs {
		entity, ok := m.db[id]
		if !(ok && entity.UID == uid) {
			// тут возможно надо обработать, что пытаются удалить чужой линк, но пока просто его пропустим
			continue
		}
		entity.Removed = true
		m.db[id] = entity
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
