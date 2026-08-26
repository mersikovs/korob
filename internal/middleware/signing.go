package middleware

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/mersikovs/korob.git/internal/logger"
	"github.com/mersikovs/korob.git/internal/signing"
)

const maxResponseSize = 1 * 1024 * 1024

type responseCapture struct {
	http.ResponseWriter
	buf         *bytes.Buffer
	wroteHeader bool
	statusCode  int
	isExceeded  bool
}

func (rw *responseCapture) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = code
	rw.wroteHeader = true
}

func (rw *responseCapture) Write(p []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}

	if rw.buf.Len()+len(p) > maxResponseSize {
		rw.isExceeded = true
		return 0, errors.New("response size exceeds maximum allowed buffer limit")
	}

	return rw.buf.Write(p)
}

func SignResponseMiddleware(key []byte, log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseCapture{
				ResponseWriter: w,
				buf:            &bytes.Buffer{},
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			if !rw.wroteHeader {
				rw.WriteHeader(http.StatusOK)
			}

			if rw.isExceeded {
				log.Info("Response size exceeded limit", "path", r.URL.Path)

				rw.buf.Reset()

				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte("Internal Server Error: Response too large to sign"))
				if err != nil {
					log.Info("Error response write", "error", err)
				}
				return
			}

			if key != nil && rw.buf.Len() > 0 {
				hash := signing.CalcHash(rw.buf.Bytes(), key)
				if hash != "" {
					rw.Header().Set(signing.HeaderName, hash)
				}
			}

			w.WriteHeader(rw.statusCode)

			if rw.buf.Len() > 0 {
				_, err := w.Write(rw.buf.Bytes())
				if err != nil {
					log.Info("Error buffer write", "error", err)
				}
			}
		})
	}
}
