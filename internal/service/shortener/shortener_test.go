package shortener

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
