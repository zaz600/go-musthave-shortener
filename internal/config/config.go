package config

import (
	"flag"
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

type RepoType int

const (
	MemoryRepo RepoType = iota
	FileRepo
	DatabaseRepo
)

func (s ShortenConfig) GetRepositoryType() RepoType {
	// пока так для совместимости с тестами
	// потом добавится отдельный ключ для типа
	if s.FileStoragePath != "" {
		return FileRepo
	}
	// 11 инкремент
	if s.DatabaseDSN != "" {
		return DatabaseRepo
	}
	return MemoryRepo
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
