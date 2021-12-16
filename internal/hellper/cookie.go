package hellper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/zaz600/go-musthave-shortener/internal/random"
)

const uidCookieName = "SHORTENER_UID"

var secret = "mysecret" // Прочитать из env/конфига

func calcHash(data string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

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
func ExtractUID(cookies []*http.Cookie) string {
	for _, cookie := range cookies {
		if cookie.Name == uidCookieName {
			parts := strings.Split(cookie.Value, ":")
			if len(parts) != 2 {
				return random.UserID()
			}
			uid, hash := parts[0], parts[1]
			if checkHash(uid, hash) {
				return uid
			}

		}
	}
	return random.UserID()
}

// SetUIDCookie сохраняет в куку uid пользователя вместе с его hmac
func SetUIDCookie(w http.ResponseWriter, uid string) {
	uuidSigned := fmt.Sprintf("%s:%s", uid, calcHash(uid))

	http.SetCookie(w, &http.Cookie{
		Name:   uidCookieName,
		Value:  uuidSigned,
		MaxAge: 3000000,
	})
}
