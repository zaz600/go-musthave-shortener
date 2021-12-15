package shortener

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/zaz600/go-musthave-shortener/internal/app/repository"
)

type gzReadCloser struct {
	*gzip.Reader
	io.Closer
}

func (gz gzReadCloser) Close() error {
	return gz.Closer.Close()
}

type Service struct {
	*chi.Mux
	baseURL    string
	repository repository.LinksRepository
}

func NewService(baseURL string, opts ...Option) *Service {
	s := &Service{
		Mux:        chi.NewRouter(),
		baseURL:    baseURL,
		repository: nil,
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			log.Panicln(err)
		}
	}

	if s.repository == nil {
		s.repository = repository.NewInMemoryLinksRepository(nil)
	}

	s.Use(middleware.RequestID)
	s.Use(middleware.RealIP)
	s.Use(middleware.Logger)
	s.Use(middleware.Recoverer)
	s.Use(middleware.Timeout(10 * time.Second))
	s.Use(middleware.Compress(5))

	s.Get("/{linkID}", s.GetLongURL())
	s.Post("/", s.SaveLongURL())
	s.Post("/api/shorten", s.ShortenJSON())
	return s
}

func (s *Service) GetLongURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		linkID := chi.URLParam(r, "linkID")
		if longURL, err := s.repository.Get(linkID); err == nil {
			http.Redirect(w, r, longURL, http.StatusTemporaryRedirect)
			return
		}
		http.Error(w, "url not found", http.StatusBadRequest)
	}
}

func (s *Service) SaveLongURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimPrefix(r.URL.Path, "/") != "" {
			http.Error(w, "invalid url, use /", http.StatusBadRequest)
			return
		}

		if r.Header.Get("Content-Encoding") == "x-gzip" {
			r.Header.Del("Content-Length")
			zr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "invalid request params", http.StatusBadRequest)
				return
			}
			r.Body = gzReadCloser{zr, r.Body}
		}

		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
			return
		}
		longURL := string(bytes)
		if !isValidURL(longURL) {
			http.Error(w, "invalid url "+longURL, http.StatusBadRequest)
			return
		}
		linkID, err := s.repository.Put(longURL)
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, "%s/%s", s.baseURL, linkID)
	}
}

func (s *Service) ShortenJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "x-gzip" {
			r.Header.Del("Content-Length")
			zr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "invalid request params", http.StatusBadRequest)
				return
			}
			r.Body = gzReadCloser{zr, r.Body}
		}

		decoder := json.NewDecoder(r.Body)
		var request ShortenRequest
		err := decoder.Decode(&request)
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
			return
		}

		longURL := request.URL
		if !isValidURL(longURL) {
			http.Error(w, "invalid url", http.StatusBadRequest)
			return
		}
		linkID, err := s.repository.Put(longURL)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		resp := ShortenResponse{
			Result: fmt.Sprintf("%s/%s", s.baseURL, linkID),
		}

		data, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, string(data))
	}
}

// isValidURL проверяет адрес на пригодность для сохранения в БД
func isValidURL(longURL string) bool {
	if longURL == "" {
		return false
	}
	_, err := url.Parse(longURL)

	return err == nil
}
