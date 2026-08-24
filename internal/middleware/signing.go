package middleware

import (
	"bytes"
	"net/http"

	"github.com/mersikovs/korob.git/internal/logger"
	"github.com/mersikovs/korob.git/internal/signing"
)

type responseCapture struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

func (rw *responseCapture) Write(p []byte) (int, error) {
	return rw.buf.Write(p)
}

func SignResponseMiddleware(key []byte, log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseCapture{
				ResponseWriter: w,
				buf:            &bytes.Buffer{},
			}
			next.ServeHTTP(rw, r)

			if key != nil && rw.buf.Len() > 0 {
				hash := signing.CalcHash(rw.buf.Bytes(), key)
				if hash != "" {
					rw.Header().Set(signing.HeaderName, hash)
				}
			}

			if rw.buf.Len() > 0 {
				_, _ = w.Write(rw.buf.Bytes())
			}
		})
	}
}
