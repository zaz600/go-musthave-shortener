package repository

import (
	"bufio"
	"encoding/gob"
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
	w               *bufio.Writer

	mu    *sync.RWMutex
	cache map[string]string
}

func NewFileLinksRepository(path string) (*FileLinksRepository, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}

	repo := &FileLinksRepository{
		fileStoragePath: path,
		file:            file,
		w:               bufio.NewWriter(file),
		mu:              &sync.RWMutex{},
		cache:           make(map[string]string),
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
	if err := f.dumpCache(); err != nil {
		return "", err
	}
	return linkID, nil
}

func (f *FileLinksRepository) Count() int {
	return len(f.cache)
}

func (f *FileLinksRepository) dumpCache() error {
	defer f.file.Sync()
	f.file.Truncate(0)
	f.file.Seek(0, 0)

	if err := gob.NewEncoder(f.file).Encode(f.cache); err != nil {
		return err
	}
	return nil
}

func (f *FileLinksRepository) loadCache() error {
	var cache map[string]string
	err := gob.NewDecoder(f.file).Decode(&cache)
	if errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return err
	}
	f.cache = cache
	log.Printf("load %d records from storage", f.Count())
	return nil
}
