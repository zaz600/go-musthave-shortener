package shortener

import (
	"github.com/zaz600/go-musthave-shortener/internal/app/repository/inmemoryrepository"
)

type Option func(*Service)

func WithMemoryRepository(db map[string]string) Option {
	return func(s *Service) {
		s.repository = inmemoryrepository.New(db)
	}
}
