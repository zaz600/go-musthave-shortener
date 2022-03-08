package config

import "os"

// getEnvOrDefault возвращает значение из переменной среды окружения,
// если такая задана или значение по умолчанию.
func getEnvOrDefault(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return defaultValue
}
