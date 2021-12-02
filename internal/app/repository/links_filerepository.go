package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/zaz600/go-musthave-shortener/internal/random"
)

type FileLinksRepository struct {
	fileStoragePath string
	file            *os.File
	encoder         *json.Encoder
	mu              *sync.RWMutex
	cache           map[string]string
}

type ShortenEntity struct {
	ID      string `json:"id"`
	LongURL string `json:"long_url"`
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
		cache: make(map[string]string),
	}

	if err = repo.loadCache(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (f *FileLinksRepository) Get(linkID string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if longURL, ok := f.cache[linkID]; ok {
		return longURL, nil
	}
	return "", fmt.Errorf("link with id '%s' not found", linkID)
}

func (f *FileLinksRepository) Put(link string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	linkID := random.String(8)
	f.cache[linkID] = link
	if err := f.dump(linkID, link); err != nil {
		return "", err
	}
	return linkID, nil
}

func (f *FileLinksRepository) Count() int {
	return len(f.cache)
}

// dump сохраняет длинную ссылку и ее идентификатор в файл
func (f *FileLinksRepository) dump(linkID string, link string) error {
	defer f.file.Sync()

	entity := ShortenEntity{
		ID:      linkID,
		LongURL: link,
	}
	if err := f.encoder.Encode(entity); err != nil {
		return err
	}
	return nil
}

// loadCache загружает кеш из файла
func (f *FileLinksRepository) loadCache() error {
	cache := make(map[string]string)
	decoder := json.NewDecoder(f.file)
	for {
		entity := ShortenEntity{}
		if err := decoder.Decode(&entity); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		cache[entity.ID] = entity.LongURL
	}
	f.cache = cache
	log.Printf("load %d records from storage", f.Count())
	return nil
}
