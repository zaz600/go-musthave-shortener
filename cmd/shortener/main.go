package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

const listenAddr = ":8080"

type ShortenerHandler struct {
	mu  sync.RWMutex
	db  map[int64]string
	seq int64
}

func (s *ShortenerHandler) getURL(idStr string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return "", false
	}
	longUrl, ok := s.db[id]
	return longUrl, ok
}

func (s *ShortenerHandler) putURL(longUrl string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := url.Parse(longUrl)
	if err != nil {
		return 0, err
	}
	s.seq++
	s.db[s.seq] = longUrl
	return s.seq, err
}

func (s *ShortenerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id := strings.TrimPrefix(r.URL.Path, "/")
		if longURL, ok := s.getURL(id); ok {
			http.Redirect(w, r, longURL, http.StatusTemporaryRedirect)
		} else {
			http.Error(w, "url not found", http.StatusBadRequest)
		}
	case http.MethodPost:
		if strings.TrimPrefix(r.URL.Path, "/") != "" {
			http.Error(w, "invalid url, use /", http.StatusBadRequest)
			break
		}

		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
			break
		}
		id, err := s.putURL(string(bytes))
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
		}
		_, _ = fmt.Fprintf(w, "%d", id)

	default:
		http.Error(w, "Only GET/POST requests are allowed!", http.StatusMethodNotAllowed)
	}
}

func main() {
	server := http.Server{
		Addr: listenAddr,
		Handler: &ShortenerHandler{
			db:  make(map[int64]string),
			seq: 1,
		},
	}

	log.Fatalln(server.ListenAndServe())
}
