package shortener

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/zaz600/go-musthave-shortener/internal/app/repository"
	"github.com/zaz600/go-musthave-shortener/internal/app/repository/memoryrepository"
)

type Service struct {
	*chi.Mux
	appDomain  string
	repository repository.LinksRepository
}

func NewService(appDomain string, opts ...Option) *Service {
	s := &Service{
		Mux:        chi.NewRouter(),
		appDomain:  appDomain,
		repository: nil,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.repository == nil {
		s.repository = memoryrepository.NewMemoryLinksRepository(nil)
	}

	s.Use(middleware.RequestID)
	s.Use(middleware.RealIP)
	s.Use(middleware.Logger)
	s.Use(middleware.Recoverer)
	s.Use(middleware.Timeout(10 * time.Second))

	s.Get("/{id}", s.GetLongURL())
	s.Post("/", s.SaveLongURL())
	return s
}

func (s *Service) GetLongURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if longURL, ok := s.repository.GetURL(id); ok {
			http.Redirect(w, r, longURL, http.StatusTemporaryRedirect)
		} else {
			http.Error(w, "url not found", http.StatusBadRequest)
		}
	}
}

func (s *Service) SaveLongURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimPrefix(r.URL.Path, "/") != "" {
			http.Error(w, "invalid url, use /", http.StatusBadRequest)
			return
		}

		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
			return
		}
		longURL := string(bytes)
		if !isValidURL(longURL) {
			http.Error(w, "invalid url", http.StatusBadRequest)
			return
		}
		id, err := s.repository.PutURL(longURL)
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, "http://%s/%d", s.appDomain, id)
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
