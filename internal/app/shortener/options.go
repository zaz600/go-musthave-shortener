package shortener

import (
	"github.com/zaz600/go-musthave-shortener/internal/app/repository/memoryrepository"
)

type Option func(*Service)

func WithMemoryRepository(db map[int64]string) Option {
	return func(s *Service) {
		s.repository = memoryrepository.NewMemoryLinksRepository(db)
	}
}
