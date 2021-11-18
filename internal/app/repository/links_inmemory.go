package repository

import (
	"fmt"
	"sync"

	"github.com/zaz600/go-musthave-shortener/internal/app/random"
)

type InMemoryLinksRepository struct {
	mu *sync.RWMutex
	db map[string]string
}

func New(db map[string]string) *InMemoryLinksRepository {
	if db == nil {
		db = make(map[string]string)
	}
	return &InMemoryLinksRepository{
		mu: &sync.RWMutex{},
		db: db,
	}
}

// Get извлекает из хранилища длинный url по идентификатору
func (m *InMemoryLinksRepository) Get(linkID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if longURL, ok := m.db[linkID]; ok {
		return longURL, nil
	}
	return "", fmt.Errorf("link with id '%s' not found", linkID)
}

// Put сохраняет длинный url в хранилище и возвращает идентификатор,
// с которым длинный url можно получить обратно
func (m *InMemoryLinksRepository) Put(link string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	linkID := random.String(8)
	m.db[linkID] = link
	return linkID, nil
}

func (m *InMemoryLinksRepository) Len() int {
	return len(m.db)
}
