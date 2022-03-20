package httpcontroller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
	"github.com/zaz600/go-musthave-shortener/internal/entity"
	"github.com/zaz600/go-musthave-shortener/internal/infrastructure/repository"
	"github.com/zaz600/go-musthave-shortener/internal/pkg/random"
	"github.com/zaz600/go-musthave-shortener/internal/service/shortener"
)

type ShortenerController struct {
	*chi.Mux
	linksService *shortener.Service
}

func New(linksService *shortener.Service) *ShortenerController {
	c := &ShortenerController{
		Mux:          chi.NewRouter(),
		linksService: linksService,
	}
	c.setupHandlers()
	return c
}

// setupHandlers настройка роутинга и middleware
func (s ShortenerController) setupHandlers() {
	s.Use(middleware.RequestID)
	s.Use(middleware.RealIP)
	s.Use(middleware.Logger)
	s.Use(middleware.Recoverer)
	s.Use(middleware.Timeout(10 * time.Second))
	s.Use(middleware.Compress(5))
	s.Use(GzDecompressor)

	s.Get("/{linkID}", s.GetOriginalURL())
	s.Post("/", s.ShortenURL())
	s.Post("/api/shorten", s.ShortenJSON())
	s.Post("/api/shorten/batch", s.ShortenBatch())
	s.Get("/api/user/urls", s.GetUserLinks())
	s.Delete("/api/user/urls", s.DeleteUserLinks())
	s.Get("/ping", s.Ping())
	s.Mount("/debug", middleware.Profiler())
}

// GetOriginalURL возвращает http.HandlerFunc для обработки запроса на получение длинной ссылки
// по короткому идентификатору
func (s ShortenerController) GetOriginalURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		linkID := chi.URLParam(r, "linkID")

		linkEntity, err := s.linksService.Get(r.Context(), linkID)
		if err != nil {
			http.Error(w, "url not found", http.StatusBadRequest)
			return
		}
		if linkEntity.Removed {
			http.Error(w, "url was removed", http.StatusGone)
			return
		}

		http.Redirect(w, r, linkEntity.OriginalURL, http.StatusTemporaryRedirect)
	}
}

// ShortenURL возвращает http.HandlerFunc для обработки запроса на сокращение ссылки.
// Ссылка передается через http Body запроса в виде строки.
func (s ShortenerController) ShortenURL() http.HandlerFunc {
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
		if !shortener.IsValidURL(originalURL) {
			http.Error(w, "invalid url "+originalURL, http.StatusBadRequest)
			return
		}
		uid, err := ExtractUID(r.Cookies())
		if err != nil {
			s.logCookieError(r, err)
			uid = random.UserID()
		}

		linkEntity := entity.NewLinkEntity(originalURL, uid)
		_, err = s.linksService.ShortenURL(r.Context(), linkEntity)
		if err != nil {
			var linkExistsErr *repository.LinkExistsError
			if !errors.As(err, &linkExistsErr) {
				log.Warn().Err(err).Fields(linkEntity).Msg("")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			linkEntity.ID = linkExistsErr.LinkID
			statusHeader = http.StatusConflict
		}
		SetUIDCookie(w, uid)
		writeAnswer(w, "text/html", statusHeader, s.linksService.ShortURL(linkEntity.ID))
	}
}

// ShortenJSON возвращает http.HandlerFunc для обработки запроса на сокращение ссылки.
// Ссылка передается в http Body в виде JSON в формате ShortenRequest.
// Ответ возвращается в формате JSON в виде ShortenResponse.
func (s ShortenerController) ShortenJSON() http.HandlerFunc {
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
		if !shortener.IsValidURL(originalURL) {
			http.Error(w, "invalid url", http.StatusBadRequest)
			return
		}
		uid, err := ExtractUID(r.Cookies())
		if err != nil {
			s.logCookieError(r, err)
			uid = random.UserID()
		}

		linkEntity := entity.NewLinkEntity(originalURL, uid)
		_, err = s.linksService.ShortenURL(r.Context(), linkEntity)
		if err != nil {
			var linkExistsErr *repository.LinkExistsError
			if !errors.As(err, &linkExistsErr) {
				log.Warn().Err(err).Fields(linkEntity).Msg("")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			linkEntity.ID = linkExistsErr.LinkID
			statusHeader = http.StatusConflict
		}
		resp := ShortenResponse{
			Result: s.linksService.ShortURL(linkEntity.ID),
		}

		data, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		SetUIDCookie(w, uid)
		writeAnswer(w, "application/json", statusHeader, string(data))
	}
}

//nolint:funlen
// ShortenBatch возвращает http.HandlerFunc для обработки запроса на сокращение пачки ссылок.
// Запрос передается в формате JSON в виде ShortenBatchRequest.
// Ответ возвращается в формате JSON в виде ShortenBatchResponse.
func (s ShortenerController) ShortenBatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var request ShortenBatchRequest
		err := decoder.Decode(&request)
		if err != nil {
			log.Warn().Err(err).Msg("")
			http.Error(w, "invalid request params", http.StatusBadRequest)
			return
		}

		uid, err := ExtractUID(r.Cookies())
		if err != nil {
			s.logCookieError(r, err)
			uid = random.UserID()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		batchService := s.linksService.NewBatchService(10)
		linkEntities := make([]entity.LinkEntity, 0, len(request))
		for _, item := range request {
			if !shortener.IsValidURL(item.URL) {
				http.Error(w, "invalid url "+item.URL, http.StatusBadRequest)
				return
			}

			e := entity.NewLinkEntity(item.URL, uid)
			e.CorrelationID = item.CorrelationID
			err = batchService.Add(ctx, e)
			if err != nil {
				log.Warn().Err(err).Str("uid", uid).Msg("")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			linkEntities = append(linkEntities, e)
		}
		err = batchService.Flush(ctx)
		if err != nil {
			log.Warn().Err(err).Str("uid", uid).Msg("")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		var resp ShortenBatchResponse
		for _, e := range linkEntities {
			resp = append(resp, ShortenBatchResponseItem{
				CorrelationID: e.CorrelationID,
				ShortURL:      s.linksService.ShortURL(e.ID),
			})
		}

		data, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		SetUIDCookie(w, uid)
		writeAnswer(w, "application/json", http.StatusCreated, string(data))
	}
}

// GetUserLinks возвращает http.HandlerFunc для обработки запроса на получение ссылок пользователя.
// Пользователь извлекается из cookie.
// Ответ возвращается в формате JSON в виде UserLinksResponse.
func (s ShortenerController) GetUserLinks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := ExtractUID(r.Cookies())
		if err != nil {
			s.logCookieError(r, err)
			http.Error(w, "no links", http.StatusNoContent)
			return
		}

		links, err := s.linksService.GetUserLinks(r.Context(), uid)
		if err != nil {
			log.Warn().Err(err).Str("uid", uid).Msg("")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if len(links) == 0 {
			http.Error(w, "no links", http.StatusNoContent)
			return
		}
		var result UserLinksResponse

		for _, e := range links {
			result = append(result, UserLinksResponseEntry{
				ShortURL:    s.linksService.ShortURL(e.ID),
				OriginalURL: e.OriginalURL,
			})
		}
		data, err := json.Marshal(result)
		if err != nil {
			http.Error(w, "no links", http.StatusNoContent)
			return
		}
		writeAnswer(w, "application/json", http.StatusOK, string(data))
	}
}

// DeleteUserLinks возвращает http.HandlerFunc для обработки запроса на удаление ссылок пользователя
// Удаление происходит асинхронно.
// Список идентификаторов ссылок передается в http Body в виде строк. На каждую ссылку одна строка.
func (s ShortenerController) DeleteUserLinks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := ExtractUID(r.Cookies())
		if err != nil {
			s.logCookieError(r, err)
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}

		decoder := json.NewDecoder(r.Body)
		var removeIDs []string
		err = decoder.Decode(&removeIDs)
		if err != nil {
			log.Warn().Err(err).Msg("")
			http.Error(w, "invalid removeIDs params", http.StatusBadRequest)
			return
		}

		s.linksService.RemoveLinks(removeIDs, uid)

		w.WriteHeader(http.StatusAccepted)
	}
}

// Ping -
func (s ShortenerController) Ping() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := s.linksService.Status(r.Context())
		if err != nil {
			http.Error(w, "pg connection error", http.StatusInternalServerError)
			return
		}
		writeAnswer(w, "application/json", http.StatusOK, "connected")
	}
}

// writeAnswer обертка для упрощения записи ответа на запросы
func writeAnswer(w http.ResponseWriter, contentType string, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
	_, _ = fmt.Fprint(w, data)
}

// logCookieError логирует ошибку ErrInvalidCookieDigest, если она возникла
func (s ShortenerController) logCookieError(r *http.Request, err error) {
	if errors.Is(err, ErrInvalidCookieDigest) {
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
