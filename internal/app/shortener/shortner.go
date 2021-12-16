package shortener

import (
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
	"github.com/zaz600/go-musthave-shortener/internal/compress"
	"github.com/zaz600/go-musthave-shortener/internal/hellper"
)

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
	s.Use(compress.GzDecompressor)

	s.Get("/{linkID}", s.GetLongURL())
	s.Post("/", s.SaveLongURL())
	s.Post("/api/shorten", s.ShortenJSON())
	s.Get("/user/urls", s.GetUserLinks())
	return s
}

func (s *Service) shortURL(linkID string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, linkID)
}

func (s *Service) GetLongURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		linkID := chi.URLParam(r, "linkID")
		if linkEntity, err := s.repository.Get(linkID); err == nil {
			http.Redirect(w, r, linkEntity.LongURL, http.StatusTemporaryRedirect)
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
			http.Error(w, "invalid url "+longURL, http.StatusBadRequest)
			return
		}
		uid := hellper.ExtractUID(r.Cookies())
		linkEntity := repository.NewLinkEntity(longURL, uid)
		linkID, err := s.repository.Put(linkEntity)
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
			return
		}
		hellper.SetUIDCookie(w, uid)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, s.shortURL(linkID))
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
		uid := hellper.ExtractUID(r.Cookies())
		linkEntity := repository.NewLinkEntity(longURL, uid)
		linkID, err := s.repository.Put(linkEntity)
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
		hellper.SetUIDCookie(w, uid)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, string(data))
	}
}

func (s *Service) GetUserLinks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := hellper.ExtractUID(r.Cookies())
		links := s.repository.FindLinksByUID(uid)
		if len(links) == 0 {
			http.Error(w, "no links", http.StatusNoContent)
			return
		}
		var result []UserLinksResponseEntry

		for _, entity := range links {
			result = append(result, UserLinksResponseEntry{
				ShortURL:    s.shortURL(entity.ID),
				OriginalURL: entity.LongURL,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		data, err := json.Marshal(result)
		if err != nil {
			http.Error(w, "no links", http.StatusNoContent)
			return
		}
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
