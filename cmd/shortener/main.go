package main

import (
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
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		return 1
	}
	return 0
}

func runApp(args []string) (err error) {
	cfg := config.GetConfig(args)
	log.Info().Msgf("app cfg: %+v", cfg)

	var repo repository.LinksRepository
	if cfg.GetRepositoryType() == repository.FileRepo {
		log.Info().Msgf("FileRepository %s", cfg.FileStoragePath)
		repo, err = repository.NewFileLinksRepository(cfg.FileStoragePath)
		if err != nil {
			return err
		}
	} else {
		log.Info().Msg("MemoryRepository")
		repo = repository.NewInMemoryLinksRepository(nil)
	}

	s := shortener.NewService(cfg.BaseURL, shortener.WithRepository(repo))
	return http.ListenAndServe(cfg.ServerAddress, s)
}
