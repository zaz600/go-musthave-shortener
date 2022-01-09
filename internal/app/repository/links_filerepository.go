package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
)

type FileLinksRepository struct {
	fileStoragePath string
	file            *os.File
	encoder         *json.Encoder
	mu              *sync.RWMutex
	cache           map[string]LinkEntity
}

func NewFileLinksRepository(ctx context.Context, path string) (*FileLinksRepository, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	repo := &FileLinksRepository{
		fileStoragePath: path,
		file:            file,
		encoder:         json.NewEncoder(file),

		mu:    &sync.RWMutex{},
		cache: make(map[string]LinkEntity),
	}

	if err = repo.loadCache(ctx); err != nil {
		return nil, err
	}
	return repo, nil
}

// Get достает по linkID из репозитория информацию по сокращенной ссылке LinkEntity
func (f *FileLinksRepository) Get(_ context.Context, linkID string) (LinkEntity, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if entity, ok := f.cache[linkID]; ok {
		return entity, nil
	}
	return LinkEntity{}, fmt.Errorf("link with id '%s' not found", linkID)
}

// PutIfAbsent сохраняет в БД длинную ссылку, если такой там еще нет.
// Если длинная ссылка есть в БД, выбрасывает исключение LinkExistsError с идентификатором ее короткой ссылки.
func (f *FileLinksRepository) PutIfAbsent(_ context.Context, linkEntity LinkEntity) (LinkEntity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, entity := range f.cache {
		if entity.OriginalURL == linkEntity.OriginalURL {
			return LinkEntity{}, NewLinkExistsError(entity.ID)
		}
	}

	f.cache[linkEntity.ID] = linkEntity
	if err := f.dump(linkEntity); err != nil {
		return LinkEntity{}, err
	}
	return linkEntity, nil
}

// PutBatch сохраняет в хранилище список сокращенных ссылок. Все ссылки записываются в одной транзакции.
func (f *FileLinksRepository) PutBatch(_ context.Context, linkEntities []LinkEntity) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, linkEntity := range linkEntities {
		f.cache[linkEntity.ID] = linkEntity
		if err := f.dump(linkEntity); err != nil {
			return err
		}
	}
	return nil
}

// Count возвращает количество записей в репозитории.
func (f *FileLinksRepository) Count(_ context.Context) (int, error) {
	return len(f.cache), nil
}

// FindLinksByUID возвращает ссылки по идентификатору пользователя
func (f *FileLinksRepository) FindLinksByUID(_ context.Context, uid string) ([]LinkEntity, error) {
	result := make([]LinkEntity, 0, 100)
	for _, entity := range f.cache {
		if entity.UID == uid {
			result = append(result, entity)
		}
	}
	return result, nil
}

// DeleteLinksByUID удаляет ссылки пользователя
func (f *FileLinksRepository) DeleteLinksByUID(ctx context.Context, uid string, ids []string) error {
	panic("TODO")
}

// dump сохраняет длинную ссылку и ее идентификатор в файл
func (f *FileLinksRepository) dump(item LinkEntity) error {
	defer func(file *os.File) {
		_ = file.Sync()
	}(f.file)

	if err := f.encoder.Encode(item); err != nil {
		return err
	}
	return nil
}

// loadCache загружает кеш из файла
func (f *FileLinksRepository) loadCache(ctx context.Context) error {
	decoder := json.NewDecoder(f.file)
	for {
		entity := LinkEntity{}
		if err := decoder.Decode(&entity); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		f.cache[entity.ID] = entity
	}
	count, _ := f.Count(ctx)
	log.Info().Msgf("load %d records from storage", count)
	return nil
}

// Status статус подключения к хранилищу
func (f *FileLinksRepository) Status(_ context.Context) error {
	return nil
}

// Close закрывает, все, что надо закрыть
func (f *FileLinksRepository) Close(_ context.Context) error {
	return f.file.Close()
}
