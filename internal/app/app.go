package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zaz600/go-musthave-shortener/internal/app/config"
	"github.com/zaz600/go-musthave-shortener/internal/infrastructure/repository"
	"github.com/zaz600/go-musthave-shortener/internal/service/shortener"
)

func Run(args []string) (err error) {
	ctxBg := context.Background()
	ctx, cancel := signal.NotifyContext(ctxBg, os.Interrupt, syscall.SIGINT)
	defer cancel()

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

	server := &http.Server{Addr: cfg.ServerAddress, Handler: s}

	go func() {
		<-ctx.Done()
		log.Info().Msg("Shutdown...")
		ctx, cancel := context.WithTimeout(ctxBg, 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Err(err).Msg("error during shutdown server")
		}
	}()

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
