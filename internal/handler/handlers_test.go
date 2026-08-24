package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
	"github.com/mersikovs/korob.git/internal/repository"
	"github.com/mersikovs/korob.git/internal/router"
	"github.com/mersikovs/korob.git/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestUpdateMetric(t *testing.T) {
	cnf := models.ServerConfig{
		DatabaseDSN:     "",
		FileStoragePath: "test.back",
	}
	ctx := context.Background()
	logger, err := logger.NewZap("debug")
	if err != nil {
		t.Fatalf("failed to create pg logger: %v", err)
	}
	modelStorage, _ := repository.NewStorage(ctx, &cnf, logger)

	service := service.NewMetricService(modelStorage, logger)
	router := router.NewRouter(service, nil)

	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name string
		path string
		want want
	}{
		{
			name: "counter positive",
			path: "/update/counter/test/1",
			want: want{
				code:        http.StatusOK,
				contentType: `text/plain; charset=utf-8`,
			},
		},
		{
			name: "gauge positive",
			path: "/update/gauge/test/1.1",
			want: want{
				code:        http.StatusOK,
				contentType: `text/plain; charset=utf-8`,
			},
		},
		{
			name: "metric not found #1",
			path: "/update/counter//1",
			want: want{
				code: http.StatusNotFound,
			},
		},
		{
			name: "metric not found #2",
			path: "/update/gauge//1",
			want: want{
				code: http.StatusNotFound,
			},
		},
		{
			name: "not valid metric type",
			path: "/update/notValidType/test/1",
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name: "not valid metric value #1",
			path: "/update/gauge/test/1a.1",
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name: "not valid metric value #2",
			path: "/update/counter/test/1.1",
			want: want{
				code: http.StatusBadRequest,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, nil)

			w := httptest.NewRecorder()

			router.ServeHTTP(w, request)

			res := w.Result()

			assert.Equal(t, test.want.code, res.StatusCode)

			contentType := w.Header().Get("Content-Type")
			assert.Equal(t, test.want.contentType, contentType)
		})
	}
}
