package shortener

import (
	"github.com/zaz600/go-musthave-shortener/internal/app/repository"
)

type Option func(*Service) error

func WithRepository(linksRepository repository.LinksRepository) Option {
	return func(s *Service) error {
		s.repository = linksRepository
		return nil
	}
}
