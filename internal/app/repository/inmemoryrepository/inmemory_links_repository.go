package inmemoryrepository

import (
	"sync"

	"github.com/zaz600/go-musthave-shortener/internal/app/random"
)

type InMemoryLinksRepository struct {
	mu  *sync.RWMutex
	db  map[string]string
	seq int64
}

func New(db map[string]string) *InMemoryLinksRepository {
	if db == nil {
		db = make(map[string]string)
	}
	return &InMemoryLinksRepository{
		mu:  &sync.RWMutex{},
		db:  db,
		seq: 0,
	}
}

// Get извлекает из хранилища длинный url по идентификатору
func (m *InMemoryLinksRepository) Get(linkID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	longURL, ok := m.db[linkID]
	return longURL, ok
}

// Put сохраняет длинный url в хранилище и возвращает идентификатор,
// с которым длинный url можно получить обратно
func (m *InMemoryLinksRepository) Put(link string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	linkID := random.RandString(8)
	m.db[linkID] = link
	return linkID, nil
}

func (m *InMemoryLinksRepository) Len() int {
	return len(m.db)
}
