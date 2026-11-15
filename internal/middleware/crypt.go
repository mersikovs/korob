package middleware

import (
	"bytes"
	"crypto/hpke"
	"io"
	"net/http"

	"github.com/mersikovs/korob.git/internal/crypt"
)

func DecryptMiddleware(privateKey hpke.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(crypt.HeaderEncrypted) != crypt.HeaderEncryptedValue {
				next.ServeHTTP(w, r)
				return
			}

			r.Header.Del("Content-Encoding")

			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
			ciphertext, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "cannot read body", http.StatusBadRequest)
				return
			}

			plaintext, err := crypt.Decrypt(privateKey, ciphertext)
			if err != nil {
				http.Error(w, "invalid ciphertext", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(plaintext))
			r.ContentLength = int64(len(plaintext))

			if r.Header.Get(crypt.HeaderInnerEncoding) == "gzip" {
				r.Header.Set("Content-Encoding", "gzip")
			}
			if ct := r.Header.Get(crypt.HeaderOriginalContentType); ct != "" {
				r.Header.Set("Content-Type", ct)
			}
			r.Header.Del(crypt.HeaderOriginalContentType)
			r.Header.Del(crypt.HeaderEncrypted)
			r.Header.Del(crypt.HeaderInnerEncoding)

			next.ServeHTTP(w, r)
		})
	}
}
