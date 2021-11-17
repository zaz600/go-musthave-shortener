package shortener

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Service struct {
	*chi.Mux
	mu        sync.RWMutex
	db        map[int64]string
	seq       int64
	appDomain string
}

func NewService(appDomain string) *Service {
	s := &Service{
		Mux:       chi.NewRouter(),
		mu:        sync.RWMutex{},
		db:        make(map[int64]string),
		seq:       0,
		appDomain: appDomain,
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
		if longURL, ok := s.GetURL(id); ok {
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
		id, err := s.PutURL(string(bytes))
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, "http://%s/%d", s.appDomain, id)
	}
}

// GetURL извлекает из хранилища длинный url по идентификатору
func (s *Service) GetURL(idStr string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return "", false
	}
	longURL, ok := s.db[id]
	return longURL, ok
}

// PutURL сохраняет длинный url в хранилище и возвращает идентификатор,
// с которым длинный url можно получить обратно
func (s *Service) PutURL(longURL string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !isValidURL(longURL) {
		return -1, errors.New("invalid url")
	}
	s.seq++
	s.db[s.seq] = longURL
	return s.seq, nil
}

// isValidURL проверяет адрес на пригодность для сохранения в БД
func isValidURL(longURL string) bool {
	if longURL == "" {
		return false
	}
	_, err := url.Parse(longURL)

	return err == nil
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id := strings.TrimPrefix(r.URL.Path, "/")
		if longURL, ok := s.GetURL(id); ok {
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
		id, err := s.PutURL(string(bytes))
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
			break
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, "http://%s/%d", s.appDomain, id)

	default:
		http.Error(w, "Only GET/POST requests are allowed!", http.StatusMethodNotAllowed)
	}
}
