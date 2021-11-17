package memoryrepository

import (
	"strconv"
	"sync"
)

type MemoryLinksRepository struct {
	mu  sync.RWMutex
	db  map[string]string
	seq int64
}

func NewMemoryLinksRepository(db map[string]string) *MemoryLinksRepository {
	if db == nil {
		db = make(map[string]string)
	}
	return &MemoryLinksRepository{
		mu:  sync.RWMutex{},
		db:  db,
		seq: 0,
	}
}

// Get извлекает из хранилища длинный url по идентификатору
func (m *MemoryLinksRepository) Get(linkID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	longURL, ok := m.db[linkID]
	return longURL, ok
}

// Put сохраняет длинный url в хранилище и возвращает идентификатор,
// с которым длинный url можно получить обратно
func (m *MemoryLinksRepository) Put(link string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	m.db[strconv.FormatInt(m.seq, 10)] = link
	return m.seq, nil
}

func (m *MemoryLinksRepository) Len() int {
	return len(m.db)
}
