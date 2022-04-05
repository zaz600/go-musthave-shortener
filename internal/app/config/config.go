package config

import (
	"flag"
	"strconv"
)

const (
	defaultBaseURL       = "http://localhost:8080"
	defaultServerAddress = "localhost:8080"
)

// ShortenConfig настройки приложения
type ShortenConfig struct {
	// BaseURL базовый адрес для сокращения ссылок - {BaseURL}/{shortID}
	BaseURL string
	// ServerAddress адрес для прослушивания входящих запросов
	ServerAddress string
	// FileStoragePath путь к файлу для хранения БД сокращенных ссылок. Опциональный параметр.
	FileStoragePath string
	// DatabaseDSN строка подключения к БД. Поддерживается PG. Параметр опциональный
	DatabaseDSN string
	// EnableTLS включать или нет ssl на прослушиваемом порту
	EnableTLS bool
}

// RepoType тип хранилища для хранения БД сокращенных ссылок
type RepoType int

const (
	// MemoryRepo хранить ссылки в памяти. БД теряется при рестарте приложения.
	MemoryRepo RepoType = iota
	// FileRepo хранить БД сокращенных ссылок в файле
	FileRepo
	// DatabaseRepo хранить БД сокращенных ссылок в БД
	DatabaseRepo
)

// GetRepositoryType возвращает тип репозитория RepoType,
// который вычисляется по переданным через флаги/env параметрам.
// Если путь к файлу и строка подключения к БД не заданы, то вернется MemoryRepo
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
	enableTLS, _ := strconv.ParseBool(getEnvOrDefault("ENABLE_HTTPS", "false"))
	flag.BoolVar(&cfg.EnableTLS, "s", enableTLS, "enable ssl. env: ENABLE_HTTPS")
	flag.Parse()
	return cfg
}
