package memoryrepository

import (
	"strconv"
	"sync"
)

type MemoryLinksRepository struct {
	mu  sync.RWMutex
	db  map[int64]string
	seq int64
}

func NewMemoryLinksRepository(db map[int64]string) *MemoryLinksRepository {
	if db == nil {
		db = make(map[int64]string)
	}
	return &MemoryLinksRepository{
		mu:  sync.RWMutex{},
		db:  db,
		seq: 0,
	}
}

// GetURL извлекает из хранилища длинный url по идентификатору
func (m *MemoryLinksRepository) GetURL(idStr string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return "", false
	}
	longURL, ok := m.db[id]
	return longURL, ok
}

// PutURL сохраняет длинный url в хранилище и возвращает идентификатор,
// с которым длинный url можно получить обратно
func (m *MemoryLinksRepository) PutURL(longURL string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	m.db[m.seq] = longURL
	return m.seq, nil
}

func (m *MemoryLinksRepository) Len() int {
	return len(m.db)
}
