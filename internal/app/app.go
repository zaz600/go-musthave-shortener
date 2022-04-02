package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zaz600/go-musthave-shortener/internal/app/config"
	"github.com/zaz600/go-musthave-shortener/internal/controller/httpcontroller"
	"github.com/zaz600/go-musthave-shortener/internal/infrastructure/repository"
	"github.com/zaz600/go-musthave-shortener/internal/service/shortener"
)

var (
	BuildVersion = "n/a"
	BuildTime    = "n/a"
	BuildCommit  = "n/a"
)

// Run инициализация и запуск приложения
func Run(args []string) (err error) {
	printBuildInfo()

	ctxBg := context.Background()
	ctx, cancel := signal.NotifyContext(ctxBg, os.Interrupt, syscall.SIGINT)
	defer cancel()

	cfg := config.GetConfig(args)
	log.Info().Msgf("app cfg: %+v", cfg)

	repo, err := repository.NewRepository(ctx, cfg)
	if err != nil {
		return err
	}

	linksService := shortener.NewService(cfg.BaseURL, shortener.WithRepository(repo))
	defer func(ctx context.Context, s *shortener.Service) {
		_ = s.Shutdown(ctx)
	}(ctx, linksService)

	server := &http.Server{Addr: cfg.ServerAddress, Handler: httpcontroller.New(linksService)}

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

func printBuildInfo() {
	fmt.Println("Build version:", BuildVersion)
	fmt.Println("Build date:", BuildTime)
	fmt.Println("Build commit:", BuildCommit)
}
