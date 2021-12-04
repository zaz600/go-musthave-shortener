package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/zaz600/go-musthave-shortener/internal/app/shortener"
	"github.com/zaz600/go-musthave-shortener/internal/config"
)

func main() {
	os.Exit(CLI(os.Args))
}

func CLI(args []string) int {
	if err := runApp(args); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		return 1
	}
	return 0
}

func runApp(args []string) error {
	cfg := config.GetConfig(args)
	log.Printf("app cfg: %+v\n", cfg)

	var repoOpt shortener.Option
	if cfg.FileStoragePath != "" {
		log.Printf("FileRepository %s\n", cfg.FileStoragePath)
		repoOpt = shortener.WithFileRepository(cfg.FileStoragePath)
	} else {
		log.Println("MemoryRepository")
		repoOpt = shortener.WithMemoryRepository(nil)
	}

	s := shortener.NewService(cfg.BaseURL, repoOpt)
	return http.ListenAndServe(cfg.ServerAddress, s)
}
