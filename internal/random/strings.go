package random

import (
	"math/rand"
	"time"
)

const charSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

func String(length int) string {
	if length < 0 {
		return ""
	}
	b := make([]byte, length)
	for i := range b {
		b[i] = charSet[seededRand.Intn(len(charSet))]
	}
	return string(b)
}

// UserID генерирует uid пользователя.
// В будущем лучше заменить на https://pkg.go.dev/github.com/google/uuid
func UserID() string {
	return String(24)
}
