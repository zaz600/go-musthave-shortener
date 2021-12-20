package config

import (
	"flag"

	"github.com/zaz600/go-musthave-shortener/internal/app/repository"
)

const (
	defaultBaseURL       = "http://localhost:8080"
	defaultServerAddress = "localhost:8080"
)

type ShortenConfig struct {
	BaseURL         string
	ServerAddress   string
	FileStoragePath string
	DatabaseDSN     string
}

func (s ShortenConfig) GetRepositoryType() repository.RepoType {
	// пока так для совместимости с тестами
	// потом добавится отдельный ключ для типа
	if s.FileStoragePath != "" {
		return repository.FileRepo
	}
	// 11 инкремент
	if s.DatabaseDSN != "" {
		return repository.DatabaseRepo
	}
	return repository.MemoryRepo
}

// GetConfig возвращает конфигурацию приложения, вычитывая в таком порядке
// аргументы командной строки -> env
// args - пока не используется
func GetConfig(args []string) *ShortenConfig {
	cfg := &ShortenConfig{}
	flag.StringVar(&cfg.ServerAddress, "a", getEnvOrDefault("SERVER_ADDRESS", defaultServerAddress), "listen address. env: SERVER_ADDRESS")
	flag.StringVar(&cfg.BaseURL, "b", getEnvOrDefault("BASE_URL", defaultBaseURL), "base url for short link. env: BASE_URL")
	flag.StringVar(&cfg.FileStoragePath, "f", getEnvOrDefault("FILE_STORAGE_PATH", ""), "file storage path. env: FILE_STORAGE_PATH")
	flag.StringVar(&cfg.DatabaseDSN, "d", getEnvOrDefault("DATABASE_DSN", ""), "PG dsn. env: DATABASE_DSN")
	flag.Parse()
	return cfg
}
