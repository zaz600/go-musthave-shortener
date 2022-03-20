package httpcontroller

import (
	"compress/gzip"
	"io"
	"net/http"
)

// gzDecompressResponseReader обертка над http.Body, которая выдает распакованные данные
type gzDecompressResponseReader struct {
	*gzip.Reader
	io.Closer
}

func (gz gzDecompressResponseReader) Close() error {
	return gz.Closer.Close()
}

// GzDecompressor middleware для распаковки тела запроса, упакованного сжатием gzip
func GzDecompressor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") == "application/x-gzip" {
			r.Header.Del("Content-Length")
			zr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "can't read gzipped data", http.StatusInternalServerError)
				return
			}
			r.Body = gzDecompressResponseReader{zr, r.Body}
		}
		next.ServeHTTP(w, r)
	})
}
