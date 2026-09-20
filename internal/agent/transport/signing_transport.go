package transport

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/mersikovs/korob.git/internal/signing"
)

type SigningTransport struct {
	Base http.RoundTripper
	Key  []byte
}

func (t *SigningTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Key == nil {
		return t.Base.RoundTrip(req)
	}

	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения тела запроса %w", err)
		}
	}

	req.Body = io.NopCloser(bytes.NewReader(body))

	sign := signing.CalcHash(body, t.Key)
	req.Header.Set(signing.HeaderName, sign)
	return t.Base.RoundTrip(req)
}
