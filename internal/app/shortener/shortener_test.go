package shortener

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zaz600/go-musthave-shortener/internal/app/repository"
	"github.com/zaz600/go-musthave-shortener/internal/hellper"
)

const baseURL = "http://localhost:8080"

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
		db          map[string]repository.LinkEntity
		queryString string
		want        want
	}{
		{
			name: "id exists",
			db: map[string]repository.LinkEntity{
				"1": {
					ID:      "1",
					LongURL: "http://ya.ru/123",
				},
			},
			queryString: "/1",
			want: want{
				code:        http.StatusTemporaryRedirect,
				location:    "http://ya.ru/123",
				contentType: "text/html; charset=utf-8",
			},
		},
		{
			name: "id does not exists",
			db: map[string]repository.LinkEntity{
				"1": {
					ID:      "1",
					LongURL: "http://ya.ru/123",
				},
			},
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
			s := NewService(baseURL, WithRepository(repository.NewInMemoryLinksRepository(tt.db)))
			ts := httptest.NewServer(s.Mux)
			defer ts.Close()

			res, _ := testRequest(t, ts, "GET", tt.queryString, nil, "") //nolint:bodyclose
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
		db          map[string]repository.LinkEntity
		queryString string
		body        []byte
		want        want
		correctURL  bool
	}{
		{
			name: "correct url",
			db: map[string]repository.LinkEntity{
				"100": {
					ID:      "100",
					LongURL: "http://ya.ru/123",
				},
			},
			queryString: "/",
			body:        []byte(`https://yandex.ru/search/?lr=2&text=abc`),
			want: want{
				code:        http.StatusCreated,
				contentType: "text/html",
				body:        "http://localhost:8080/",
			},
			correctURL: true,
		},
		{
			name: "incorrect url",
			db: map[string]repository.LinkEntity{
				"100": {
					ID:      "100",
					LongURL: "http://ya.ru/123",
				},
			},
			queryString: "/",
			body:        []byte(``),
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "invalid url",
			},
			correctURL: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewService(baseURL, WithRepository(repository.NewInMemoryLinksRepository(tt.db)))

			ts := httptest.NewServer(s.Mux)
			defer ts.Close()

			res, respBody := testRequest(t, ts, "POST", tt.queryString, bytes.NewReader(tt.body), "") //nolint:bodyclose
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.True(t, strings.HasPrefix(respBody, tt.want.body))
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))

			if tt.correctURL {
				parsedURL, err := url.Parse(respBody)
				assert.NoError(t, err)
				assert.Len(t, parsedURL.Path, 9)
			}
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

	db := map[string]repository.LinkEntity{
		"100": {
			ID:      "100",
			LongURL: "http://ya.ru/123",
		},
	}
	s := NewService(baseURL, WithRepository(repository.NewInMemoryLinksRepository(db)))

	ts := httptest.NewServer(s.Mux)
	defer ts.Close()

	// сохраняем длинный урл
	resGet, shortLink := testRequest(t, ts, "POST", "/", bytes.NewReader([]byte(longURL)), "") //nolint:bodyclose
	defer resGet.Body.Close()
	assert.NotEmpty(t, shortLink)
	parsedURL, err := url.Parse(shortLink)
	assert.NoError(t, err)

	// достаем длинный урл
	res, _ := testRequest(t, ts, "GET", parsedURL.Path, nil, "") //nolint:bodyclose
	defer res.Body.Close()

	assert.Equal(t, want.code, res.StatusCode)
	assert.Equal(t, want.location, res.Header.Get("location"))
	assert.Equal(t, want.contentType, res.Header.Get("Content-Type"))
}

func TestService_PostMultiple(t *testing.T) {
	db := map[string]repository.LinkEntity{
		"100": {
			ID:      "100",
			LongURL: "http://ya.ru/123",
		},
	}
	s := NewService(baseURL, WithRepository(repository.NewInMemoryLinksRepository(db)))
	ts := httptest.NewServer(s.Mux)
	defer ts.Close()

	for i := 0; i < 5; i++ {
		longURL := fmt.Sprintf(`https://yandex.ru/search/?lr=2&text=abc%d`, i)
		res, _ := testRequest(t, ts, "POST", "/", bytes.NewReader([]byte(longURL)), "") //nolint:bodyclose
		res.Body.Close()
	}

	assert.Equal(t, 6, s.repository.Count()) // 1 + 5
}

func TestService_GetUserLinks(t *testing.T) {
	longURL := `https://yandex.ru/search/?lr=2&text=abc`
	db := map[string]repository.LinkEntity{
		"100": {
			ID:      "100",
			LongURL: "http://ya.ru/123",
			UID:     "100500",
		},
	}
	s := NewService(baseURL, WithRepository(repository.NewInMemoryLinksRepository(db)))
	ts := httptest.NewServer(s.Mux)
	defer ts.Close()

	// сохраняем длинный урл
	resGet, shortLink := testRequest(t, ts, "POST", "/", bytes.NewReader([]byte(longURL)), "") //nolint:bodyclose
	defer resGet.Body.Close()
	assert.NotEmpty(t, shortLink)

	uid := hellper.ExtractUID(resGet.Cookies())

	res, respBody := testRequest(t, ts, "GET", "/user/urls", nil, uid)
	res.Body.Close()

	var actual []UserLinksResponseEntry
	err := json.Unmarshal([]byte(respBody), &actual)
	assert.NoError(t, err)

	assert.Len(t, actual, 1)
	assert.Equal(t, longURL, actual[0].OriginalURL)
}

//nolint:funlen
func TestService_Post_JSON(t *testing.T) {
	type want struct {
		code        int
		contentType string
		body        string
	}

	tests := []struct {
		name        string
		db          map[string]repository.LinkEntity
		queryString string
		body        []byte
		contentType string
		want        want
		correctURL  bool
	}{
		{
			name: "correct url",
			db: map[string]repository.LinkEntity{
				"100": {
					ID:      "100",
					LongURL: "http://ya.ru/123",
				},
			},
			queryString: "/api/shorten",
			body:        []byte(`{"url": "https://ya.ru"}`),
			want: want{
				code:        http.StatusCreated,
				contentType: "application/json",
				body:        "http://localhost:8080/",
			},
			correctURL: true,
		},
		{
			name: "incorrect url",
			db: map[string]repository.LinkEntity{
				"100": {
					ID:      "100",
					LongURL: "http://ya.ru/123",
				},
			},
			queryString: "/api/shorten",
			body:        []byte(``),
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "invalid url",
			},
			correctURL: false,
		},
		{
			name: "invalid json",
			db: map[string]repository.LinkEntity{
				"100": {
					ID:      "100",
					LongURL: "http://ya.ru/123",
				},
			},
			queryString: "/api/shorten",
			body:        []byte(`{"url":"http://ya.ru"`),
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "invalid url",
			},
			correctURL: false,
		},
		{
			name: "missing field",
			db: map[string]repository.LinkEntity{
				"100": {
					ID:      "100",
					LongURL: "http://ya.ru/123",
				},
			},
			queryString: "/api/shorten",
			body:        []byte(`{"foo":"http://ya.ru"}`),
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "invalid url",
			},
			correctURL: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewService(baseURL, WithRepository(repository.NewInMemoryLinksRepository(tt.db)))

			ts := httptest.NewServer(s.Mux)
			defer ts.Close()

			res, respBody := testRequest(t, ts, "POST", tt.queryString, bytes.NewReader(tt.body), "") //nolint:bodyclose
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))

			if tt.correctURL {
				var actual ShortenResponse
				err := json.Unmarshal([]byte(respBody), &actual)
				assert.NoError(t, err)
				assert.True(t, strings.HasPrefix(actual.Result, tt.want.body))

				parsedURL, err := url.Parse(actual.Result)
				assert.NoError(t, err)
				assert.Len(t, parsedURL.Path, 9)
			}
		})
	}
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader, uid string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	require.NoError(t, err)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: jar,
	}

	if uid != "" {
		uuidSigned := fmt.Sprintf("%s:%s", uid, hellper.CalcHash(uid))

		cookie := &http.Cookie{
			Name:   "SHORTENER_UID",
			Value:  uuidSigned,
			MaxAge: 3000000,
		}

		urlObj, _ := url.Parse(ts.URL)
		client.Jar.SetCookies(urlObj, []*http.Cookie{cookie})
	}

	resp, err := client.Do(req)
	require.NoError(t, err)

	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	defer resp.Body.Close()

	return resp, string(respBody)
}
