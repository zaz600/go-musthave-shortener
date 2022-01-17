package shortener

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/zaz600/go-musthave-shortener/internal/app/repository"
	"github.com/zaz600/go-musthave-shortener/internal/helper"
)

type Service struct {
	*chi.Mux
	baseURL      string
	repository   repository.LinksRepository
	linkRemoveCh chan<- removeUserLinksRequest
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
		s.repository = repository.NewInMemoryLinksRepository(context.Background(), nil)
	}
	s.setupHandlers()
	s.linkRemoveCh = s.startRemoveLinksWorkers(context.Background(), 10)
	return s
}

func (s *Service) shortURL(linkID string) string {
	parsedURL, _ := url.Parse(s.baseURL)
	parsedURL.Path = linkID
	return parsedURL.String()
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

func (s *Service) startRemoveLinksWorkers(ctx context.Context, count int) chan<- removeUserLinksRequest {
	linkCh := make(chan removeUserLinksRequest, count*2)
	for i := 0; i < count; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case req := <-linkCh:
					err := s.repository.DeleteLinksByUID(ctx, req.uid, req.linkIDs...)
					if err != nil {
						log.Warn().Err(err).Strs("ids", req.linkIDs).Str("uid", req.uid).Msg("error delete user links")
					} else {
						log.Info().Str("uid", req.uid).Strs("ids", req.linkIDs).Msg("urls deleted")
					}
				}
			}
		}()
	}
	return linkCh
}

// isValidURL проверяет адрес на пригодность для сохранения в БД
func isValidURL(value string) bool {
	if value == "" {
		return false
	}
	_, err := url.Parse(value)

	return err == nil
}

func writeAnswer(w http.ResponseWriter, contentType string, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
	_, _ = fmt.Fprint(w, data)
}
