package shortener

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/zaz600/go-musthave-shortener/internal/app/repository"
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
		s.repository = repository.New(nil)
	}

	s.Use(middleware.RequestID)
	s.Use(middleware.RealIP)
	s.Use(middleware.Logger)
	s.Use(middleware.Recoverer)
	s.Use(middleware.Timeout(10 * time.Second))

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
		linkID, err := s.repository.Put(longURL)
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, "http://%s/%s", s.appDomain, linkID)
	}
}

func (s *Service) ShortenJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			Result: fmt.Sprintf("http://%s/%s", s.appDomain, linkID),
		}

		data, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
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
