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
)

type Service struct {
	mu        sync.RWMutex
	db        map[int64]string
	seq       int64
	appDomain string
}

func NewService(appDomain string) *Service {
	return &Service{
		mu:        sync.RWMutex{},
		db:        make(map[int64]string),
		seq:       0,
		appDomain: appDomain,
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
	if !s.isValidURL(longURL) {
		return -1, errors.New("invalid url")
	}
	s.seq++
	s.db[s.seq] = longURL
	return s.seq, nil
}

// isValidURL проверяет адрес на пригодность для сохранения в БД
func (s *Service) isValidURL(longURL string) bool {
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
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, "http://%s/%d", s.appDomain, id)

	default:
		http.Error(w, "Only GET/POST requests are allowed!", http.StatusMethodNotAllowed)
	}
}
