package config

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"

	"github.com/rs/zerolog/log"
)

const (
	defaultBaseURL       = "http://localhost:8080"
	defaultServerAddress = "localhost:8080"
)

// ShortenConfig настройки приложения
type ShortenConfig struct {
	// BaseURL базовый адрес для сокращения ссылок - {BaseURL}/{shortID}
	BaseURL string `json:"base_url"`
	// ServerAddress адрес для прослушивания входящих запросов
	ServerAddress string `json:"server_address"`
	// FileStoragePath путь к файлу для хранения БД сокращенных ссылок. Опциональный параметр.
	FileStoragePath string `json:"file_storage_path"`
	// DatabaseDSN строка подключения к БД. Поддерживается PG. Параметр опциональный
	DatabaseDSN string `json:"database_dsn"`
	// EnableHTTPS включать или нет ssl на прослушиваемом порту
	EnableHTTPS bool `json:"enable_https"`
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

func getConfigFileName(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// GetConfig возвращает конфигурацию приложения, вычитывая в таком порядке
// аргументы командной строки -> env
// args - пока не используется
func GetConfig(args []string) *ShortenConfig {
	configFile := getConfigFileName(args)
	cfg := mustGetParamsFromFile(configFile)

	_ = flag.String("c", "", "config file. env: CONFIG")
	flag.StringVar(&cfg.ServerAddress, "a", getEnvOrDefault("SERVER_ADDRESS", defaultServerAddress), "listen address. env: SERVER_ADDRESS")
	flag.StringVar(&cfg.BaseURL, "b", getEnvOrDefault("BASE_URL", defaultBaseURL), "base url for short link. env: BASE_URL")
	flag.StringVar(&cfg.FileStoragePath, "f", getEnvOrDefault("FILE_STORAGE_PATH", cfg.FileStoragePath), "file storage path. env: FILE_STORAGE_PATH")
	flag.StringVar(&cfg.DatabaseDSN, "d", getEnvOrDefault("DATABASE_DSN", cfg.DatabaseDSN), "PG dsn. env: DATABASE_DSN")
	enableHTTPS, _ := strconv.ParseBool(getEnvOrDefault("ENABLE_HTTPS", strconv.FormatBool(cfg.EnableHTTPS)))
	flag.BoolVar(&cfg.EnableHTTPS, "s", enableHTTPS, "enable ssl. env: ENABLE_HTTPS")
	flag.Parse()
	return &cfg
}

func mustGetParamsFromFile(configFile string) ShortenConfig {
	if configFile == "" {
		return ShortenConfig{}
	}
	f, err := os.Open(configFile)
	if err != nil {
		log.Panic().Err(err).Msg("error opening config file")
	}
	defer f.Close()
	var configJSON ShortenConfig
	err = json.NewDecoder(f).Decode(&configJSON)
	if err != nil {
		log.Panic().Err(err).Msg("error reading config file")
	}
	return configJSON
}
