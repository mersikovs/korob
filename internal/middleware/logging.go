package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/mersikovs/korob.git/internal/logger"
)

type responseRecorder struct {
	http.ResponseWriter
	status  int
	size    int
	written bool
}

func (rr *responseRecorder) WriteHeader(code int) {
	if rr.written {
		return
	}
	rr.status = code
	rr.written = true
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	if !rr.written {
		rr.status = http.StatusOK
		rr.written = true
	}
	n, err := rr.ResponseWriter.Write(b)
	rr.size += n
	return n, err
}

func (rr *responseRecorder) Status() int {
	if !rr.written {
		return http.StatusOK
	}
	return rr.status
}

func (rr *responseRecorder) BytesWritten() int {
	return rr.size
}

var recorderPool = sync.Pool{
	New: func() any {
		return &responseRecorder{}
	},
}

func Logger(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			rr := recorderPool.Get().(*responseRecorder)

			rr.ResponseWriter = w
			rr.status = 0
			rr.size = 0
			rr.written = false

			defer func() {
				rr.ResponseWriter = nil
				recorderPool.Put(rr)
			}()

			start := time.Now()
			next.ServeHTTP(rr, r)

			log.Info(
				"server",
				"req.method", r.Method,
				"req.uri", r.RequestURI,
				"req.duration", time.Since(start).Seconds(),
				"res.status", rr.Status(),
				"res.size", rr.BytesWritten(),
			)
		})
	}
}
