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

	repo, err := repository.NewRepository(ctx, cfg)
	if err != nil {
		return err
	}

	s := shortener.NewService(cfg.BaseURL, shortener.WithRepository(repo))
	defer func(ctx context.Context, s *shortener.Service) {
		_ = s.Shutdown(ctx)
	}(ctx, s)
	return http.ListenAndServe(cfg.ServerAddress, s)
}
