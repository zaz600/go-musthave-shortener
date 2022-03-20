package shortener

import (
	"github.com/zaz600/go-musthave-shortener/internal/infrastructure/repository"
)

type Option func(*Service) error

// WithRepository указание типа репозитория для сервиса
func WithRepository(linksRepository repository.LinksRepository) Option {
	return func(s *Service) error {
		s.linksRepository = linksRepository
		return nil
	}
}
