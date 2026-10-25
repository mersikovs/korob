package transport

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/mersikovs/korob.git/internal/signing"
)

// SigningTransport is an http.RoundTripper wrapper that automatically
// signs outgoing HTTP requests
type SigningTransport struct {
	Base http.RoundTripper
	Key  []byte
}

// RoundTrip implements the http.RoundTripper interface. It calculates
// the request signature, attaches it to the headers, and safely restores
// the request body for the underlying transport to read.
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
