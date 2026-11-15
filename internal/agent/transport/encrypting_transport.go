package transport

import (
	"bytes"
	"crypto/hpke"
	"fmt"
	"io"
	"net/http"

	"github.com/mersikovs/korob.git/internal/crypt"
)

type EncryptingTransport struct {
	Base http.RoundTripper
	Pub  hpke.PublicKey
}

func NewEncryptingTransport(base http.RoundTripper, pub hpke.PublicKey) *EncryptingTransport {
	return &EncryptingTransport{Base: base, Pub: pub}
}

func (t *EncryptingTransport) RoundTrip(req *http.Request) (*http.Response, error) {

	if req.Body == nil || req.Body == http.NoBody {
		return t.Base.RoundTrip(req)
	}

	gzBody, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	_ = req.Body.Close()

	ciphertext, err := crypt.Encrypt(t.Pub, gzBody)
	if err != nil {
		return nil, fmt.Errorf("encrypt body: %w", err)
	}

	newReq := req.Clone(req.Context())
	newReq.Body = io.NopCloser(bytes.NewReader(ciphertext))
	newReq.ContentLength = int64(len(ciphertext))
	newReq.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(ciphertext)), nil
	}

	if newReq.Header.Get("Content-Encoding") == "gzip" {
		newReq.Header.Set(crypt.HeaderInnerEncoding, "gzip")
	}

	newReq.Header.Del("Content-Encoding")

	newReq.Header.Set(crypt.HeaderEncrypted, crypt.HeaderEncryptedValue)

	if ct := newReq.Header.Get("Content-Type"); ct != "" {
		newReq.Header.Set(crypt.HeaderOriginalContentType, ct)
	}
	newReq.Header.Set("Content-Type", "application/octet-stream")

	return t.Base.RoundTrip(newReq)
}
