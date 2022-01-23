package httpcontroller

import (
	"bytes"
	"context"
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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zaz600/go-musthave-shortener/internal/entity"
	"github.com/zaz600/go-musthave-shortener/internal/infrastructure/repository"
	"github.com/zaz600/go-musthave-shortener/internal/service/shortener"
)

const baseURL = "http://localhost:8080"

func TestShortenerController_Ping(t *testing.T) {
	linksService := shortener.NewService(baseURL, shortener.WithRepository(repository.NewInMemoryLinksRepository(context.TODO(), nil)))
	controller := New(linksService)
	ts := httptest.NewServer(controller.Mux)
	defer ts.Close()

	res, body := testRequest(t, ts, "GET", "/ping", nil, nil) //nolint:bodyclose
	defer res.Body.Close()
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "connected", body)
}

func TestShortenerController_GetOriginalURL(t *testing.T) {
	type want struct {
		code        int
		contentType string
		location    string
	}

	tests := []struct {
		name        string
		db          map[string]entity.LinkEntity
		queryString string
		want        want
	}{
		{
			name: "id exists",
			db: map[string]entity.LinkEntity{
				"1": {
					ID:          "1",
					OriginalURL: "http://ya.ru/123",
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
			db: map[string]entity.LinkEntity{
				"1": {
					ID:          "1",
					OriginalURL: "http://ya.ru/123",
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
			linksService := shortener.NewService(baseURL, shortener.WithRepository(repository.NewInMemoryLinksRepository(context.TODO(), tt.db)))
			controller := New(linksService)
			ts := httptest.NewServer(controller.Mux)
			defer ts.Close()

			res, _ := testRequest(t, ts, "GET", tt.queryString, nil, nil) //nolint:bodyclose
			defer res.Body.Close()
			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.location, res.Header.Get("location"))
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

//nolint:funlen
func TestShortenerController_ShortenURL(t *testing.T) {
	type want struct {
		code        int
		contentType string
		body        string
	}

	tests := []struct {
		name        string
		db          map[string]entity.LinkEntity
		queryString string
		body        []byte
		want        want
		correctURL  bool
	}{
		{
			name: "correct url",
			db: map[string]entity.LinkEntity{
				"100": {
					ID:          "100",
					OriginalURL: "http://ya.ru/123",
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
			db: map[string]entity.LinkEntity{
				"100": {
					ID:          "100",
					OriginalURL: "http://ya.ru/123",
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
			linksService := shortener.NewService(baseURL, shortener.WithRepository(repository.NewInMemoryLinksRepository(context.TODO(), tt.db)))
			controller := New(linksService)
			ts := httptest.NewServer(controller.Mux)
			defer ts.Close()

			res, respBody := testRequest(t, ts, "POST", tt.queryString, bytes.NewReader(tt.body), nil) //nolint:bodyclose
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

func TestShortenerController_ShortenURLMultiple(t *testing.T) {
	db := map[string]entity.LinkEntity{
		"100": {
			ID:          "100",
			OriginalURL: "http://ya.ru/123",
		},
	}
	linksService := shortener.NewService(baseURL, shortener.WithRepository(repository.NewInMemoryLinksRepository(context.TODO(), db)))
	controller := New(linksService)
	ts := httptest.NewServer(controller.Mux)
	defer ts.Close()

	for i := 0; i < 5; i++ {
		longURL := fmt.Sprintf(`https://yandex.ru/search/?lr=2&text=abc%d`, i)
		res, _ := testRequest(t, ts, "POST", "/", bytes.NewReader([]byte(longURL)), nil) //nolint:bodyclose
		res.Body.Close()
	}
	count, err := linksService.Count(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, 6, count) // 1 + 5
}

func TestShortenerController_ShortenURLExists(t *testing.T) {
	longURL := "http://ya.ru/123"
	db := map[string]entity.LinkEntity{
		"100": {
			ID:          "100",
			OriginalURL: longURL,
		},
	}
	linksService := shortener.NewService(baseURL, shortener.WithRepository(repository.NewInMemoryLinksRepository(context.TODO(), db)))
	controller := New(linksService)
	ts := httptest.NewServer(controller.Mux)
	defer ts.Close()

	res, body := testRequest(t, ts, "POST", "/", bytes.NewReader([]byte(longURL)), nil) //nolint:bodyclose
	defer res.Body.Close()

	assert.Equal(t, http.StatusConflict, res.StatusCode)
	assert.Equal(t, body, "http://localhost:8080/100")
}

func TestShortenerController_ShortenSuccessPath(t *testing.T) {
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

	db := map[string]entity.LinkEntity{
		"100": {
			ID:          "100",
			OriginalURL: "http://ya.ru/123",
		},
	}
	linksService := shortener.NewService(baseURL, shortener.WithRepository(repository.NewInMemoryLinksRepository(context.TODO(), db)))
	controller := New(linksService)
	ts := httptest.NewServer(controller.Mux)
	defer ts.Close()

	// сохраняем длинный урл
	resGet, shortLink := testRequest(t, ts, "POST", "/", bytes.NewReader([]byte(longURL)), nil) //nolint:bodyclose
	defer resGet.Body.Close()
	assert.NotEmpty(t, shortLink)
	parsedURL, err := url.Parse(shortLink)
	assert.NoError(t, err)

	// достаем длинный урл
	res, _ := testRequest(t, ts, "GET", parsedURL.Path, nil, nil) //nolint:bodyclose
	defer res.Body.Close()

	assert.Equal(t, want.code, res.StatusCode)
	assert.Equal(t, want.location, res.Header.Get("location"))
	assert.Equal(t, want.contentType, res.Header.Get("Content-Type"))
}

//nolint:funlen
func TestShortenerController_ShortenJSON(t *testing.T) {
	type want struct {
		code        int
		contentType string
		body        string
	}

	tests := []struct {
		name        string
		db          map[string]entity.LinkEntity
		queryString string
		body        []byte
		contentType string
		want        want
		correctURL  bool
	}{
		{
			name: "correct url",
			db: map[string]entity.LinkEntity{
				"100": {
					ID:          "100",
					OriginalURL: "http://ya.ru/123",
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
			db: map[string]entity.LinkEntity{
				"100": {
					ID:          "100",
					OriginalURL: "http://ya.ru/123",
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
			db: map[string]entity.LinkEntity{
				"100": {
					ID:          "100",
					OriginalURL: "http://ya.ru/123",
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
			db: map[string]entity.LinkEntity{
				"100": {
					ID:          "100",
					OriginalURL: "http://ya.ru/123",
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
			linksService := shortener.NewService(baseURL, shortener.WithRepository(repository.NewInMemoryLinksRepository(context.TODO(), tt.db)))
			controller := New(linksService)
			ts := httptest.NewServer(controller.Mux)
			defer ts.Close()

			res, respBody := testRequest(t, ts, "POST", tt.queryString, bytes.NewReader(tt.body), nil) //nolint:bodyclose
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

func TestShortenerController_ShortenJSONExists(t *testing.T) {
	longURL := "http://ya.ru/123"

	db := map[string]entity.LinkEntity{
		"100": {
			ID:          "100",
			OriginalURL: longURL,
		},
	}
	linksService := shortener.NewService(baseURL, shortener.WithRepository(repository.NewInMemoryLinksRepository(context.TODO(), db)))
	controller := New(linksService)
	ts := httptest.NewServer(controller.Mux)
	defer ts.Close()

	request := []byte(fmt.Sprintf(`{"url":"%s"}`, longURL))
	res, body := testRequest(t, ts, "POST", "/api/shorten", bytes.NewReader(request), nil) //nolint:bodyclose
	defer res.Body.Close()

	assert.Equal(t, http.StatusConflict, res.StatusCode)
	assert.JSONEq(t, body, `{"result":"http://localhost:8080/100"}`)
}

func TestShortenerController_GetUserLinks(t *testing.T) {
	longURL := `https://yandex.ru/search/?lr=2&text=abc`
	db := map[string]entity.LinkEntity{
		"100": {
			ID:          "100",
			OriginalURL: "http://ya.ru/123",
			UID:         "100500",
		},
	}
	linksService := shortener.NewService(baseURL, shortener.WithRepository(repository.NewInMemoryLinksRepository(context.TODO(), db)))
	controller := New(linksService)
	ts := httptest.NewServer(controller.Mux)
	defer ts.Close()

	// сохраняем длинный урл
	resGet, shortLink := testRequest(t, ts, "POST", "/", bytes.NewReader([]byte(longURL)), nil) //nolint:bodyclose
	defer resGet.Body.Close()
	assert.NotEmpty(t, shortLink)

	uidCookie := extractUIDCookie(t, resGet)

	res, respBody := testRequest(t, ts, "GET", "/user/urls", nil, uidCookie)
	res.Body.Close()

	var actual []UserLinksResponseEntry
	err := json.Unmarshal([]byte(respBody), &actual)
	assert.NoError(t, err)

	assert.Len(t, actual, 1)
	assert.Equal(t, longURL, actual[0].OriginalURL)
}

//nolint:funlen
func TestShortenerController_ShortenBatch(t *testing.T) {
	type want struct {
		code        int
		contentType string
		len         int
	}

	tests := []struct {
		name        string
		db          map[string]entity.LinkEntity
		queryString string
		body        []byte
		contentType string
		want        want
		correctURL  bool
	}{
		{
			name: "correct url",
			db: map[string]entity.LinkEntity{
				"100": {
					ID:          "100",
					OriginalURL: "http://ya.ru/123",
				},
			},
			queryString: "/api/shorten/batch",
			body:        []byte(`[{"original_url": "https://ya.ru/?dsfsfsdf", "correlation_id": "1"}, {"original_url": "https://ya.ru/?12345", "correlation_id": "2"}]`),
			want: want{
				code:        http.StatusCreated,
				contentType: "application/json",
				len:         2,
			},
			correctURL: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linksService := shortener.NewService(baseURL, shortener.WithRepository(repository.NewInMemoryLinksRepository(context.TODO(), tt.db)))
			controller := New(linksService)
			ts := httptest.NewServer(controller.Mux)
			defer ts.Close()

			res, respBody := testRequest(t, ts, "POST", tt.queryString, bytes.NewReader(tt.body), nil) //nolint:bodyclose
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			if tt.correctURL {
				var actual ShortenBatchResponse
				err := json.Unmarshal([]byte(respBody), &actual)
				require.NoError(t, err)
				assert.Len(t, actual, tt.want.len)
				assert.NotEqual(t, actual[0].ShortURL, actual[1].ShortURL)
			}
		})
	}
}

func TestShortenerController_DeleteUserLinks(t *testing.T) {
	db := map[string]entity.LinkEntity{
		"100": {
			ID:          "100",
			OriginalURL: "http://ya.ru/123",
			UID:         "100500",
		},
	}

	linksService := shortener.NewService(baseURL, shortener.WithRepository(repository.NewInMemoryLinksRepository(context.TODO(), db)))
	controller := New(linksService)
	ts := httptest.NewServer(controller.Mux)
	defer ts.Close()

	// Добавляем ссылки
	links := shortenLinks(t, ts, 5)
	linksToDelete := make([]LinkInfo, 0, 3)
	linksNotDeleted := make([]LinkInfo, 0, len(links)-cap(linksToDelete))
	for _, linkInfo := range links {
		if len(linksToDelete) == cap(linksToDelete) {
			linksNotDeleted = append(linksNotDeleted, linkInfo)
		} else {
			linksToDelete = append(linksToDelete, linkInfo)
		}
	}
	deleteReq := []byte(fmt.Sprintf(`["%s", "%s", "%s"]`, linksToDelete[0].ShortID, linksToDelete[1].ShortID, linksToDelete[2].ShortID))
	// удаляем
	resDel, _ := testRequest(t, ts, "DELETE", "/api/user/urls", bytes.NewReader(deleteReq), linksToDelete[0].Cookie) //nolint:bodyclose
	defer resDel.Body.Close()
	require.Equal(t, http.StatusAccepted, resDel.StatusCode, "Запрос на удаление ссылок успешен")

	// Проверяем статусы по удаленным ссылкам
	for _, deletedLink := range linksToDelete {
		deletedLink := deletedLink
		require.Eventually(t, func() bool {
			res, _ := testRequest(t, ts, "GET", fmt.Sprintf("/%s", deletedLink.ShortID), nil, nil)
			res.Body.Close()
			return res.StatusCode == http.StatusGone
		}, 1*time.Second, 100*time.Millisecond, "links are gone")
	}

	// Проверяем статусы по ссылкам, которые не удаляли
	for _, link := range linksNotDeleted {
		res, _ := testRequest(t, ts, "GET", fmt.Sprintf("/%s", link.ShortID), nil, nil)
		res.Body.Close()
		assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
	}

	// В репо ничего не удалилось
	count, err := linksService.Count(context.TODO())
	require.NoError(t, err)
	assert.Equal(t, len(links)+1, count)

	// Ручка получения ссылок юзера не выдает удаленные ссылки
	res, respBody := testRequest(t, ts, "GET", "/user/urls", nil, linksToDelete[0].Cookie)
	res.Body.Close()

	var actual []UserLinksResponseEntry
	err = json.Unmarshal([]byte(respBody), &actual)
	assert.NoError(t, err)
	assert.Len(t, actual, len(linksNotDeleted))
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader, cookie *http.Cookie) (*http.Response, string) {
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

	if cookie != nil {
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

func extractUIDCookie(t *testing.T, r *http.Response) *http.Cookie {
	t.Helper()

	var uidCookie *http.Cookie
	for _, cookie := range r.Cookies() {
		if cookie.Name == "SHORTENER_UID" {
			uidCookie = cookie
			break
		}
	}
	require.NotNil(t, uidCookie)
	return uidCookie
}

func shortenLinks(t *testing.T, ts *httptest.Server, count int) map[string]LinkInfo {
	t.Helper()

	var uidCookie *http.Cookie
	links := make(map[string]LinkInfo, 5)
	for i := 0; i < count; i++ {
		longURL := fmt.Sprintf(`https://yandex.ru/search/?lr=2&text=abc%d`, i)
		res, shortURL := testRequest(t, ts, "POST", "/", bytes.NewReader([]byte(longURL)), uidCookie) //nolint:bodyclose
		res.Body.Close()

		assert.NotEmpty(t, shortURL)
		parsedURL, err := url.Parse(shortURL)
		assert.NoError(t, err)
		shortID := strings.TrimPrefix(parsedURL.Path, "/")

		if i == 0 {
			// извлекаем куку из первого запроса, чтобы остальные сделать с ней же
			uidCookie = extractUIDCookie(t, res)
		}

		links[shortID] = LinkInfo{
			LongURL:  longURL,
			ShortURL: shortURL,
			ShortID:  shortID,
			Cookie:   uidCookie,
		}
	}

	return links
}

type LinkInfo struct {
	LongURL  string
	ShortURL string
	ShortID  string
	Cookie   *http.Cookie
}
