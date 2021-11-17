package shortener

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		db          map[string]string
		queryString string
		want        want
	}{
		{
			name:        "id exists",
			db:          map[string]string{"1": "http://ya.ru/123"},
			queryString: "/1",
			want: want{
				code:        http.StatusTemporaryRedirect,
				location:    "http://ya.ru/123",
				contentType: "text/html; charset=utf-8",
			},
		},
		{
			name:        "id does not exists",
			db:          map[string]string{"1": "http://ya.ru/123"},
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
			s := NewService("localhost:8080", WithMemoryRepository(tt.db))
			ts := httptest.NewServer(s.Mux)
			defer ts.Close()

			res, _ := testRequest(t, ts, "GET", tt.queryString, nil) //nolint:bodyclose
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
		db          map[string]string
		queryString string
		body        []byte
		want        want
	}{
		{
			name:        "correct url",
			db:          map[string]string{"100": "http://ya.ru/123"},
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
			db:          map[string]string{"100": "http://ya.ru/123"},
			queryString: "/",
			body:        []byte(``),
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "invalid url\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewService("localhost:8080", WithMemoryRepository(tt.db))

			ts := httptest.NewServer(s.Mux)
			defer ts.Close()

			res, respBody := testRequest(t, ts, "POST", tt.queryString, bytes.NewReader(tt.body)) //nolint:bodyclose
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.body, respBody)
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

	db := map[string]string{"100": "http://ya.ru/123"}
	s := NewService("localhost:8080", WithMemoryRepository(db))

	ts := httptest.NewServer(s.Mux)
	defer ts.Close()

	resGet, _ := testRequest(t, ts, "POST", "/", bytes.NewReader([]byte(longURL))) //nolint:bodyclose
	defer resGet.Body.Close()

	// достаем длинный урл
	res, _ := testRequest(t, ts, "GET", "/1", nil) //nolint:bodyclose
	defer res.Body.Close()

	assert.Equal(t, want.code, res.StatusCode)
	assert.Equal(t, want.location, res.Header.Get("location"))
	assert.Equal(t, want.contentType, res.Header.Get("Content-Type"))
}

func TestService_PostMultiple(t *testing.T) {
	db := map[string]string{"100": "http://ya.ru/123"}
	s := NewService("localhost:8080", WithMemoryRepository(db))
	ts := httptest.NewServer(s.Mux)
	defer ts.Close()

	for i := 0; i < 5; i++ {
		longURL := fmt.Sprintf(`https://yandex.ru/search/?lr=2&text=abc%d`, i)
		res, _ := testRequest(t, ts, "POST", "/", bytes.NewReader([]byte(longURL))) //nolint:bodyclose
		res.Body.Close()
	}

	assert.Equal(t, 6, s.repository.Len()) // 1 + 5
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	require.NoError(t, err)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	require.NoError(t, err)

	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	defer resp.Body.Close()

	return resp, string(respBody)
}
