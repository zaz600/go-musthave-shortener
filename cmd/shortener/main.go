package main

import (
	"log"
	"net/http"

	"github.com/zaz600/go-musthave-shortener/internal/app/shortener"
)

const listenAddr = "localhost:8080"

func main() {
	s := shortener.NewService(listenAddr, shortener.WithMemoryRepository(nil))
	log.Fatalln(http.ListenAndServe(listenAddr, s))
}
