package shortener

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zaz600/go-musthave-shortener/internal/entity"
	"github.com/zaz600/go-musthave-shortener/internal/infrastructure/repository"
	"github.com/zaz600/go-musthave-shortener/internal/pkg/random"
)

func Test_isValidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "valid url",
			url:  "https://ya.ru",
			want: true,
		},
		{
			name: "valid url without proto",
			url:  "ya.ru",
			want: true,
		},
		{
			name: "invalid url",
			url:  "",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidURL(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Benchmark_isValidURL(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsValidURL("http://ya.ru?1")
	}
}

func BenchmarkShortenURL(b *testing.B) {
	linksService := NewService("http://localhost:8080", WithRepository(repository.NewInMemoryLinksRepository(context.TODO(), nil)))
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		link := entity.NewLinkEntity("http://ya.ru/?"+random.String(10), "100")
		b.StartTimer()
		_, _ = linksService.ShortenURL(context.TODO(), link)
	}
}
