package main

import (
	"log"
	"net/http"

	"github.com/zaz600/go-musthave-shortener/internal/app/shortener"
	"github.com/zaz600/go-musthave-shortener/internal/helpers"
)

const (
	defaultBaseURL       = "localhost:8080"
	defaultServerAddress = "localhost:8080"
)

func main() {
	baseURL := helpers.GetEnvOrDefault("BASE_URL", defaultBaseURL)
	serverAddress := helpers.GetEnvOrDefault("SERVER_ADDRESS", defaultServerAddress)
	s := shortener.NewService(baseURL, shortener.WithMemoryRepository(nil))
	log.Fatalln(http.ListenAndServe(serverAddress, s))
}
