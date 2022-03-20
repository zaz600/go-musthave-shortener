package random_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zaz600/go-musthave-shortener/internal/pkg/random"
)

func TestString(t *testing.T) {
	assert.Empty(t, random.String(-1))
	assert.Empty(t, random.String(0))
	assert.Len(t, random.String(1), 1)
	assert.Len(t, random.String(10), 10)

	assert.NotEqual(t, random.String(10), random.String(10))
}

func TestUserID(t *testing.T) {
	assert.Len(t, random.UserID(), 24)
	assert.NotEqual(t, random.UserID(), random.UserID())
}

func BenchmarkString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		random.String(10)
	}
}

func BenchmarkString100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		random.String(100)
	}
}

func ExampleString() {
	fmt.Println(random.String(10))
}

func ExampleUserID() {
	fmt.Println(random.UserID())
}
