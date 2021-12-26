package shortener

import (
	"context"
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
	s.Post("/api/shorten/batch", s.ShortenBatch())
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
		if linkEntity, err := s.repository.Get(r.Context(), linkID); err == nil {
			http.Redirect(w, r, linkEntity.OriginalURL, http.StatusTemporaryRedirect)
			return
		}
		http.Error(w, "url not found", http.StatusBadRequest)
	}
}

func (s *Service) ShortenURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statusHeader := http.StatusCreated

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
		linkID, err := s.repository.Put(r.Context(), linkEntity)
		if err != nil {
			var linkExistsErr *repository.LinkExistsError
			if errors.As(err, &linkExistsErr) {
				linkID = linkExistsErr.LinkID
				statusHeader = http.StatusConflict
			} else {
				log.Warn().Err(err).Fields(linkEntity).Msg("")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}
		helper.SetUIDCookie(w, uid)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(statusHeader)
		_, _ = fmt.Fprint(w, s.shortURL(linkID))
	}
}

func (s *Service) ShortenJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statusHeader := http.StatusCreated
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
		linkID, err := s.repository.Put(r.Context(), linkEntity)
		if err != nil {
			var linkExistsErr *repository.LinkExistsError
			if errors.As(err, &linkExistsErr) {
				linkID = linkExistsErr.LinkID
				statusHeader = http.StatusConflict
			} else {
				log.Warn().Err(err).Fields(linkEntity).Msg("")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
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
		w.WriteHeader(statusHeader)
		_, _ = fmt.Fprint(w, string(data))
	}
}

func (s *Service) ShortenBatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var request ShortenBatchRequest
		err := decoder.Decode(&request)
		if err != nil {
			http.Error(w, "invalid request params", http.StatusBadRequest)
			return
		}

		uid, err := helper.ExtractUID(r.Cookies())
		if err != nil {
			s.logCookieError(r, err)
			uid = random.UserID()
		}

		// если записей будет 1М, то конвертацию лучше делать в функции вставки,
		// которая будет конвертировать в нужный формат батчами
		linkEntities := make([]repository.LinkEntity, 0, len(request))
		for _, item := range request {
			if !isValidURL(item.URL) {
				http.Error(w, "invalid url "+item.URL, http.StatusBadRequest)
				return
			}

			entity := repository.NewLinkEntity(item.URL, uid)
			entity.CorrelationID = item.CorrelationID
			linkEntities = append(linkEntities, entity)
		}
		// Сейчас возможна ситуация, когда ошибка произошла на последней записи из пачки,
		// мы вернем ошибку, хотя в бд будут вставлены все записи, кроме последней
		// Нужно уточнить ТЗ :)
		err = s.repository.PutBatch(r.Context(), linkEntities)
		if err != nil {
			log.Warn().Err(err).Str("uid", uid).Msg("")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		var resp ShortenBatchResponse
		for _, entity := range linkEntities {
			resp = append(resp, ShortenBatchResponseItem{
				CorrelationID: entity.CorrelationID,
				ShortURL:      s.shortURL(entity.ID),
			})
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
		links, err := s.repository.FindLinksByUID(r.Context(), uid)
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
		err := s.repository.Status(r.Context())
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

func (s *Service) Shutdown(ctx context.Context) error {
	return s.repository.Close(ctx)
}

// isValidURL проверяет адрес на пригодность для сохранения в БД
func isValidURL(value string) bool {
	if value == "" {
		return false
	}
	_, err := url.Parse(value)

	return err == nil
}
