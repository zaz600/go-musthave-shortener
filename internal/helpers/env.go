package helpers

import "os"

func GetEnvOrDefault(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return defaultValue
}
