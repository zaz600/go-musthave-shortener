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

func NewInMemoryLinksRepository(db map[string]LinkEntity) InMemoryLinksRepository {
	if db == nil {
		db = make(map[string]LinkEntity)
	}
	return InMemoryLinksRepository{
		mu: &sync.RWMutex{},
		db: db,
	}
}

// Get извлекает из хранилища длинный url по идентификатору
func (m InMemoryLinksRepository) Get(_ context.Context, linkID string) (LinkEntity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entity, ok := m.db[linkID]; ok {
		return entity, nil
	}
	return LinkEntity{}, fmt.Errorf("link with id '%s' not found", linkID)
}

// Put сохраняет длинный url в хранилище и возвращает идентификатор,
// с которым длинный url можно получить обратно
func (m InMemoryLinksRepository) Put(_ context.Context, linkEntity LinkEntity) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, entity := range m.db {
		if entity.OriginalURL == linkEntity.OriginalURL {
			return "", NewLinkExistsError(entity.ID)
		}
	}

	m.db[linkEntity.ID] = linkEntity
	return linkEntity.ID, nil
}

func (m InMemoryLinksRepository) PutBatch(_ context.Context, linkEntities []LinkEntity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, linkEntity := range linkEntities {
		m.db[linkEntity.ID] = linkEntity
	}
	return nil
}

func (m InMemoryLinksRepository) Count(_ context.Context) (int, error) {
	return len(m.db), nil
}

func (m InMemoryLinksRepository) FindLinksByUID(_ context.Context, uid string) ([]LinkEntity, error) {
	result := make([]LinkEntity, 0, 100)
	for _, entity := range m.db {
		if entity.UID == uid {
			result = append(result, entity)
		}
	}
	return result, nil
}

func (m InMemoryLinksRepository) Status(_ context.Context) error {
	return nil
}

func (m InMemoryLinksRepository) Close(_ context.Context) error {
	return nil
}
