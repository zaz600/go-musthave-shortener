package config

import "os"

func getEnvOrDefault(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return defaultValue
}
