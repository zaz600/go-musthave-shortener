package shortener

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService_isValidURL(t *testing.T) {
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
			got := isValidURL(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestService_Get(t *testing.T) {
	type want struct {
		code        int
		contentType string
		location    string
	}

	tests := []struct {
		name        string
		db          map[int64]string
		queryString string
		want        want
	}{
		{
			name:        "id exists",
			db:          map[int64]string{1: "http://ya.ru/123"},
			queryString: "/1",
			want: want{
				code:        http.StatusTemporaryRedirect,
				location:    "http://ya.ru/123",
				contentType: "text/html; charset=utf-8",
			},
		},
		{
			name:        "id does not exists",
			db:          map[int64]string{1: "http://ya.ru/123"},
			queryString: "/2",
			want: want{
				code:        http.StatusBadRequest,
				location:    "",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Service{
				db:        tt.db,
				appDomain: "localhost:8080",
			}
			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.ServeHTTP)
			request := httptest.NewRequest(http.MethodGet, tt.queryString, nil)
			h.ServeHTTP(w, request)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.location, res.Header.Get("location"))
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestService_Post(t *testing.T) {
	type want struct {
		code        int
		contentType string
		body        string
	}

	tests := []struct {
		name        string
		db          map[int64]string
		queryString string
		body        []byte
		want        want
	}{
		{
			name:        "correct url",
			db:          map[int64]string{100: "http://ya.ru/123"},
			queryString: "/",
			body:        []byte(`https://yandex.ru/search/?lr=2&text=abc`),
			want: want{
				code:        http.StatusCreated,
				contentType: "text/html; charset=utf-8",
				body:        "http://localhost:8080/1",
			},
		},
		{
			name:        "incorrect url",
			db:          map[int64]string{100: "http://ya.ru/123"},
			queryString: "/",
			body:        []byte(``),
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "invalid request params\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Service{
				db:        tt.db,
				appDomain: "localhost:8080",
			}
			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.ServeHTTP)
			request := httptest.NewRequest(http.MethodPost, tt.queryString, bytes.NewReader(tt.body))
			h.ServeHTTP(w, request)
			res := w.Result()
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.body, string(resBody))
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestService_SuccessPath(t *testing.T) {
	longURL := `https://yandex.ru/search/?lr=2&text=abc`

	want := struct {
		code        int
		location    string
		contentType string
	}{
		code:        http.StatusTemporaryRedirect,
		location:    longURL,
		contentType: "text/html; charset=utf-8",
	}

	s := Service{
		db:        map[int64]string{100: "http://ya.ru/123"},
		appDomain: "localhost:8080",
	}

	// в хелпер унести get/post
	h := http.HandlerFunc(s.ServeHTTP)
	wPost := httptest.NewRecorder()

	// сохраняем урл. Должны получить айди /1
	requestPost := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(longURL)))
	h.ServeHTTP(wPost, requestPost)
	resPost := wPost.Result()
	defer resPost.Body.Close()

	// достаем длинный урл
	wGet := httptest.NewRecorder()
	requestGet := httptest.NewRequest(http.MethodGet, "/1", nil)
	h.ServeHTTP(wGet, requestGet)
	resGet := wGet.Result()
	defer resGet.Body.Close()

	assert.Equal(t, want.code, resGet.StatusCode)
	assert.Equal(t, want.location, resGet.Header.Get("location"))
	assert.Equal(t, want.contentType, resGet.Header.Get("Content-Type"))
}

func TestService_PostMultiple(t *testing.T) {
	s := Service{
		db:        map[int64]string{100: "http://ya.ru/123"},
		appDomain: "localhost:8080",
	}

	for i := 0; i < 5; i++ {
		longURL := fmt.Sprintf(`https://yandex.ru/search/?lr=2&text=abc%d`, i)

		h := http.HandlerFunc(s.ServeHTTP)
		wPost := httptest.NewRecorder()
		// сохраняем урл. Должны получить айди /1
		requestPost := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(longURL)))
		h.ServeHTTP(wPost, requestPost)
		resPost := wPost.Result()
		resPost.Body.Close()
	}

	assert.Equal(t, 6, len(s.db)) // 1 + 5
}
