package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvOrDefault_Env_Exists(t *testing.T) {
	value := "abcdef"
	key := "MY_ENV_VAR1"
	defValue := "foobarbaz"
	_ = os.Setenv(key, value)
	actual := getEnvOrDefault(key, defValue)
	assert.Equal(t, value, actual)
}

func TestGetEnvOrDefault_Env_not_Exists(t *testing.T) {
	key := "MY_ENV_VAR2"
	defValue := "foobarbaz"

	actual := getEnvOrDefault(key, defValue)
	assert.Equal(t, defValue, actual)
}
