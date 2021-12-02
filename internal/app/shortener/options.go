package shortener

import (
	"github.com/zaz600/go-musthave-shortener/internal/app/repository"
)

type Option func(*Service) error

func WithMemoryRepository(db map[string]string) Option {
	return func(s *Service) error {
		s.repository = repository.New(db)
		return nil
	}
}

func WithFileRepository(storagePath string) Option {
	return func(s *Service) error {
		r, err := repository.NewFileLinksRepository(storagePath)
		if err != nil {
			return err
		}
		s.repository = r
		return nil
	}
}
