package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/zaz600/go-musthave-shortener/internal/app/shortener"
	"github.com/zaz600/go-musthave-shortener/internal/helpers"
)

const (
	defaultBaseURL       = "http://localhost:8080"
	defaultServerAddress = "localhost:8080"
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
	baseURL := helpers.GetEnvOrDefault("BASE_URL", defaultBaseURL)
	serverAddress := helpers.GetEnvOrDefault("SERVER_ADDRESS", defaultServerAddress)
	fileStoragePath := helpers.GetEnvOrDefault("FILE_STORAGE_PATH", "")

	var repoOpt shortener.Option
	if fileStoragePath != "" {
		log.Printf("FileRepository %s\n", fileStoragePath)
		repoOpt = shortener.WithFileRepository(fileStoragePath)
	} else {
		log.Println("MemoryRepository")
		repoOpt = shortener.WithMemoryRepository(nil)
	}

	s := shortener.NewService(baseURL, repoOpt)
	return http.ListenAndServe(serverAddress, s)
}
