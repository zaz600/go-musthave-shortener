package httpcontroller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const uidCookieName = "SHORTENER_UID"

var (
	ErrInvalidCookieValue  = errors.New("invalid cookie value")
	ErrInvalidCookieDigest = errors.New("invalid cookie digest")
	ErrNoCookie            = errors.New("no cookie")
)

var secret = "mysecret" // Прочитать из env/конфига

// CalcHash вычисление HMAC-SHA256 для переданной строки
func CalcHash(data string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// checkHash проверка хеша
func checkHash(data string, hash string) bool {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	sign, err := hex.DecodeString(hash)
	if err != nil {
		return false
	}
	return hmac.Equal(sign, h.Sum(nil))
}

// ExtractUID извлекает из куки uid пользователя и валидирует его.
// Если хэш корректный, возвращает uid.
// Если куки нет или хеш невалидный - генерирует новый uid
func ExtractUID(cookies []*http.Cookie) (string, error) {
	for _, cookie := range cookies {
		if cookie.Name == uidCookieName {
			parts := strings.Split(cookie.Value, ":")
			if len(parts) != 2 {
				return "", ErrInvalidCookieValue
			}
			uid, hash := parts[0], parts[1]
			if checkHash(uid, hash) {
				return uid, nil
			}
			return "", ErrInvalidCookieDigest
		}
	}
	return "", ErrNoCookie
}

// SetUIDCookie сохраняет в куку uid пользователя вместе с его hmac
func SetUIDCookie(w http.ResponseWriter, uid string) {
	uuidSigned := fmt.Sprintf("%s:%s", uid, CalcHash(uid))

	http.SetCookie(w, &http.Cookie{
		Name:   uidCookieName,
		Value:  uuidSigned,
		MaxAge: 3000000,
	})
}
