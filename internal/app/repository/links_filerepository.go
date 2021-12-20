package repository

import (
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

func NewFileLinksRepository(path string) (*FileLinksRepository, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0777)
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

	if err = repo.loadCache(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (f *FileLinksRepository) Get(linkID string) (LinkEntity, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if entity, ok := f.cache[linkID]; ok {
		return entity, nil
	}
	return LinkEntity{}, fmt.Errorf("link with id '%s' not found", linkID)
}

func (f *FileLinksRepository) Put(linkEntity LinkEntity) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cache[linkEntity.ID] = linkEntity
	if err := f.dump(linkEntity); err != nil {
		return "", err
	}
	return linkEntity.ID, nil
}

func (f *FileLinksRepository) Count() (int, error) {
	return len(f.cache), nil
}

func (f *FileLinksRepository) FindLinksByUID(uid string) ([]LinkEntity, error) {
	result := make([]LinkEntity, 0, 100)
	for _, entity := range f.cache {
		if entity.UID == uid {
			result = append(result, entity)
		}
	}
	return result, nil
}

// dump сохраняет длинную ссылку и ее идентификатор в файл
func (f *FileLinksRepository) dump(item LinkEntity) error {
	defer f.file.Sync()

	if err := f.encoder.Encode(item); err != nil {
		return err
	}
	return nil
}

// loadCache загружает кеш из файла
func (f *FileLinksRepository) loadCache() error {
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
	count, _ := f.Count()
	log.Info().Msgf("load %d records from storage", count)
	return nil
}

func (f *FileLinksRepository) Status() error {
	return nil
}

func (f *FileLinksRepository) Close() error {
	return f.file.Close()
}
