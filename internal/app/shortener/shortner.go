package shortener

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
	"github.com/zaz600/go-musthave-shortener/internal/app/repository"
	"github.com/zaz600/go-musthave-shortener/internal/compress"
	"github.com/zaz600/go-musthave-shortener/internal/helper"
	"github.com/zaz600/go-musthave-shortener/internal/random"
)

type Service struct {
	*chi.Mux
	baseURL    string
	repository repository.LinksRepository
	RepoTmp    *repository.PgLinksRepository // удалить на следующем инкременте
}

func NewService(baseURL string, opts ...Option) *Service {
	s := &Service{
		Mux:        chi.NewRouter(),
		baseURL:    baseURL,
		repository: nil,
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			log.Panic().Err(err).Msg("")
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

	s.Get("/{linkID}", s.GetOriginalURL())
	s.Post("/", s.ShortenURL())
	s.Post("/api/shorten", s.ShortenJSON())
	s.Get("/user/urls", s.GetUserLinks())
	s.Get("/ping", s.Ping())
	return s
}

func (s *Service) shortURL(linkID string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, linkID)
}

func (s *Service) GetOriginalURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		linkID := chi.URLParam(r, "linkID")
		if linkEntity, err := s.repository.Get(linkID); err == nil {
			http.Redirect(w, r, linkEntity.OriginalURL, http.StatusTemporaryRedirect)
			return
		}
		http.Error(w, "url not found", http.StatusBadRequest)
	}
}

func (s *Service) ShortenURL() http.HandlerFunc {
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
		originalURL := string(bytes)
		if !isValidURL(originalURL) {
			http.Error(w, "invalid url "+originalURL, http.StatusBadRequest)
			return
		}
		uid, err := helper.ExtractUID(r.Cookies())
		if err != nil {
			s.logCookieError(r, err)
			uid = random.UserID()
		}
		linkEntity := repository.NewLinkEntity(originalURL, uid)
		linkID, err := s.repository.Put(linkEntity)
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
			return
		}
		helper.SetUIDCookie(w, uid)
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

		originalURL := request.URL
		if !isValidURL(originalURL) {
			http.Error(w, "invalid url", http.StatusBadRequest)
			return
		}
		uid, err := helper.ExtractUID(r.Cookies())
		if err != nil {
			s.logCookieError(r, err)
			uid = random.UserID()
		}
		linkEntity := repository.NewLinkEntity(originalURL, uid)
		linkID, err := s.repository.Put(linkEntity)
		if err != nil {
			log.Warn().Err(err).Fields(linkEntity).Msg("")
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
		helper.SetUIDCookie(w, uid)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, string(data))
	}
}

func (s *Service) GetUserLinks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := helper.ExtractUID(r.Cookies())
		if err != nil {
			s.logCookieError(r, err)
			http.Error(w, "no links", http.StatusNoContent)
			return
		}
		links, err := s.repository.FindLinksByUID(uid)
		if err != nil {
			log.Warn().Err(err).Str("uid", uid).Msg("")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if len(links) == 0 {
			http.Error(w, "no links", http.StatusNoContent)
			return
		}
		var result []UserLinksResponseEntry

		for _, entity := range links {
			result = append(result, UserLinksResponseEntry{
				ShortURL:    s.shortURL(entity.ID),
				OriginalURL: entity.OriginalURL,
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

func (s *Service) Ping() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := s.RepoTmp.Status()
		if err != nil {
			http.Error(w, "pg connection error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "connected")
	}
}

func (s *Service) logCookieError(r *http.Request, err error) {
	if errors.Is(err, helper.ErrInvalidCookieDigest) {
		log.Warn().
			Err(err).
			Fields(map[string]interface{}{
				"remote_ip":  r.RemoteAddr,
				"url":        r.URL.Path,
				"proto":      r.Proto,
				"method":     r.Method,
				"user_agent": r.Header.Get("User-Agent"),
				"bytes_in":   r.Header.Get("Content-Length"),
			}).
			Msg("")
	}
}
func (s *Service) Shutdown() error {
	return s.repository.Close()
}

// isValidURL проверяет адрес на пригодность для сохранения в БД
func isValidURL(value string) bool {
	if value == "" {
		return false
	}
	_, err := url.Parse(value)

	return err == nil
}
