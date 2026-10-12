package middleware

import (
	"bytes"
	"crypto/hmac"
	"io"
	"net/http"

	"github.com/mersikovs/korob.git/internal/logger"
	"github.com/mersikovs/korob.git/internal/signing"
)

// VerifySha256Middleware validates the HMAC-SHA256 signature of the request body.
//
// It bypasses validation if the key or [signing.HeaderName] is empty,
// safely restores the request body for downstream handlers, and returns
// 400 Bad Request on hash mismatch.
func VerifySha256Middleware(key []byte, log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if len(key) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			headerHash := r.Header.Get(signing.HeaderName)
			if headerHash == "" {
				next.ServeHTTP(w, r)
				return
			}

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "не удается прочитать тело", http.StatusInternalServerError)
				return
			}

			if err := r.Body.Close(); err != nil {
				log.Info("ошибка закрытия тела запроса ", "error = ", err)
			}

			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			expectedHash := signing.CalcHash(bodyBytes, key)
			if expectedHash == "" {
				http.Error(w, "Internal error computing hash", http.StatusInternalServerError)
				return
			}

			if !hmac.Equal([]byte(expectedHash), []byte(headerHash)) {
				http.Error(w, "Hash mismatch", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
