package shortener

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
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

func (s *Service) setupHandlers() {
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
	s.Delete("/api/user/urls", s.DeleteUserLinks())
	s.Get("/ping", s.Ping())
}

func (s *Service) GetOriginalURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		linkID := chi.URLParam(r, "linkID")
		linkEntity, err := s.repository.Get(r.Context(), linkID)
		if err != nil {
			if errors.Is(err, repository.ErrLinkRemoved) {
				http.Error(w, "url was removed", http.StatusGone)
				return
			}

			http.Error(w, "url not found", http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, linkEntity.OriginalURL, http.StatusTemporaryRedirect)
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
		_, err = s.repository.PutIfAbsent(r.Context(), linkEntity)
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
		helper.SetUIDCookie(w, uid)
		writeAnswer(w, "text/html", statusHeader, s.shortURL(linkEntity.ID))
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
		_, err = s.repository.PutIfAbsent(r.Context(), linkEntity)
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
			Result: s.shortURL(linkEntity.ID),
		}

		data, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		helper.SetUIDCookie(w, uid)
		writeAnswer(w, "application/json", statusHeader, string(data))
	}
}

func (s *Service) ShortenBatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var request ShortenBatchRequest
		err := decoder.Decode(&request)
		if err != nil {
			log.Warn().Err(err).Msg("")
			http.Error(w, "invalid request params", http.StatusBadRequest)
			return
		}

		uid, err := helper.ExtractUID(r.Cookies())
		if err != nil {
			s.logCookieError(r, err)
			uid = random.UserID()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		batch := repository.NewBatchService(100, s.repository)
		linkEntities := make([]repository.LinkEntity, 0, len(request))
		for _, item := range request {
			if !isValidURL(item.URL) {
				http.Error(w, "invalid url "+item.URL, http.StatusBadRequest)
				return
			}

			entity := repository.NewLinkEntity(item.URL, uid)
			entity.CorrelationID = item.CorrelationID
			err = batch.Add(ctx, entity)
			if err != nil {
				log.Warn().Err(err).Str("uid", uid).Msg("")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			linkEntities = append(linkEntities, entity)
		}
		err = batch.Flush(ctx)
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
		writeAnswer(w, "application/json", http.StatusCreated, string(data))
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
		data, err := json.Marshal(result)
		if err != nil {
			http.Error(w, "no links", http.StatusNoContent)
			return
		}
		writeAnswer(w, "application/json", http.StatusOK, string(data))
	}
}

func (s *Service) DeleteUserLinks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := helper.ExtractUID(r.Cookies())
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

		go func() {
			// если использовать r.Context() то он отменяется при завершении зарпоса
			err = s.repository.DeleteLinksByUID(context.Background(), uid, removeIDs)
			if err != nil {
				log.Warn().Err(err).Str("uid", uid).Msg("")
				return
			}
			log.Info().Str("uid", uid).Strs("ids", removeIDs).Msg("urls deleted")
		}()

		w.WriteHeader(http.StatusAccepted)
	}
}

func (s *Service) Ping() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := s.repository.Status(r.Context())
		if err != nil {
			http.Error(w, "pg connection error", http.StatusInternalServerError)
			return
		}
		writeAnswer(w, "application/json", http.StatusOK, "connected")
	}
}
