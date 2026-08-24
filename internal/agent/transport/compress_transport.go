package transport

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"

	"github.com/mersikovs/korob.git/internal/logger"
)

type CompressTransport struct {
	Base   http.RoundTripper
	Logger logger.Logger
}

func NewCompressTransport(base http.RoundTripper, logger logger.Logger) *CompressTransport {
	return &CompressTransport{
		Base:   base,
		Logger: logger,
	}
}

func (c *CompressTransport) RoundTrip(req *http.Request) (*http.Response, error) {

	if req.Body == nil {
		return c.Base.RoundTrip(req)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		if req.Body != nil {
			if err = req.Body.Close(); err != nil {
				c.Logger.Info("ошибка закрытия тела запроса", "err", err)
			}
		}
		return nil, fmt.Errorf("read request body: %w", err)
	}

	if req.Body != nil {
		if closeErr := req.Body.Close(); closeErr != nil {
			c.Logger.Info("ошибка закрытия тела запроса", "err", closeErr)
		}
	}

	if len(body) == 0 {
		req.Body = io.NopCloser(bytes.NewReader(body))
		return c.Base.RoundTrip(req)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(body); err != nil {
		_ = gz.Close()
		return nil, fmt.Errorf("gzip write:: %w", err)
	}

	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	req.Body = io.NopCloser(&buf)
	req.Header.Set("Content-Encoding", "gzip")
	req.ContentLength = -1
	return c.Base.RoundTrip(req)
}
