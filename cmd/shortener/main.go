package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zaz600/go-musthave-shortener/internal/app/repository"
	"github.com/zaz600/go-musthave-shortener/internal/app/shortener"
	"github.com/zaz600/go-musthave-shortener/internal/config"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
	os.Exit(CLI(os.Args))
}

func CLI(args []string) int {
	if err := runApp(args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		return 1
	}
	return 0
}

func runApp(args []string) (err error) {
	ctx := context.Background()
	cfg := config.GetConfig(args)
	log.Info().Msgf("app cfg: %+v", cfg)

	var repo repository.LinksRepository
	switch cfg.GetRepositoryType() {
	case repository.FileRepo:
		log.Info().Msgf("FileRepository %s", cfg.FileStoragePath)
		repo, err = repository.NewFileLinksRepository(ctx, cfg.FileStoragePath)
		if err != nil {
			return err
		}
	case repository.DatabaseRepo:
		log.Info().Msg("DatabaseRepo")
		repo, err = repository.NewPgLinksRepository(ctx, cfg.DatabaseDSN)
		if err != nil {
			return err
		}
	default:
		log.Info().Msg("MemoryRepository")
		repo = repository.NewInMemoryLinksRepository(nil)
	}

	s := shortener.NewService(cfg.BaseURL, shortener.WithRepository(repo))
	defer func(s *shortener.Service, ctx context.Context) {
		_ = s.Shutdown(ctx)
	}(s, context.Background())
	return http.ListenAndServe(cfg.ServerAddress, s)
}
