package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mersikovs/korob.git/internal/middleware"
	"github.com/stretchr/testify/require"
)

type testLogger struct {
	logs []map[string]any
}

func (l *testLogger) Info(msg string, keysAndValues ...any) {
	if l.logs == nil {
		l.logs = make([]map[string]any, 0)
	}
	addData := make(map[string]any)
	addData["msg"] = msg
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			addData[keysAndValues[i].(string)] = keysAndValues[i+1]
		}
	}
	l.logs = append(l.logs, addData)
}

func (l *testLogger) Fatal(msg string, keysAndValues ...any) {}

func TestLogger(t *testing.T) {
	type req struct {
		method        string
		path          string
		handlerStatus int
		handlerBody   string
	}

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		requests []req
		wantLogs []map[string]any
	}{
		{
			name: "one request",
			requests: []req{
				{
					method:        http.MethodGet,
					path:          "/update/gauge/test/1.1",
					handlerStatus: http.StatusOK,
					handlerBody:   "Hello, World!",
				},
			},
			wantLogs: []map[string]any{
				{
					"req.method": "GET",
					"req.uri":    "/update/gauge/test/1.1",
					"res.status": 200,
					"res.size":   13,
				},
			},
		},
		{
			name: "two request",
			requests: []req{
				{
					method:        http.MethodGet,
					path:          "/update/gauge/test/1.1",
					handlerStatus: http.StatusOK,
					handlerBody:   "Hello, World!",
				},
				{
					method:        http.MethodGet,
					path:          "/update/gauge/test/1.2",
					handlerStatus: http.StatusOK,
					handlerBody:   "Hello, World!!",
				},
			},
			wantLogs: []map[string]any{
				{
					"req.method": "GET",
					"req.uri":    "/update/gauge/test/1.1",
					"res.status": 200,
					"res.size":   13,
				},
				{
					"req.method": "GET",
					"req.uri":    "/update/gauge/test/1.2",
					"res.status": 200,
					"res.size":   14,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &testLogger{}
			for _, testreq := range tt.requests {

				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if testreq.handlerStatus != 0 {
						w.WriteHeader(testreq.handlerStatus)
					}
					w.Write([]byte(testreq.handlerBody))
				})

				wrapped := middleware.Logger(log)(handler)
				req := httptest.NewRequest(testreq.method, testreq.path, nil)
				rr := httptest.NewRecorder()

				wrapped.ServeHTTP(rr, req)

				if rr.Code != testreq.handlerStatus {
					t.Errorf("response status: got %d, want %d", rr.Code, testreq.handlerStatus)
				}

			}

			require.Equal(t, len(log.logs), len(tt.requests), "number of log entries should match number of requests")

			for _, wantLog := range tt.wantLogs {
				found := false
				for _, logEntry := range log.logs {
					_, durationExist := logEntry["req.duration"]
					if logEntry["req.method"] == wantLog["req.method"] &&
						logEntry["req.uri"] == wantLog["req.uri"] &&
						logEntry["res.status"] == wantLog["res.status"] &&
						logEntry["res.size"] == wantLog["res.size"] &&
						durationExist {

						found = true
						break

					}
				}
				require.True(t, found, "expected log entry not found")
			}

		})
	}
}
