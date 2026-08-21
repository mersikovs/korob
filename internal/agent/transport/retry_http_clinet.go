package transport

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RetryClient struct {
	client *http.Client
	delays []time.Duration
}

func New(delays []time.Duration) *RetryClient {
	return &RetryClient{
		client: http.DefaultClient,
		delays: delays,
	}
}

func (c *RetryClient) Do(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte

	if req.Body != nil && req.Body != http.NoBody {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
	}

	if bodyBytes != nil {
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
		req.ContentLength = int64(len(bodyBytes))
	}

	retries := len(c.delays)
	maxAttempts := retries + 1

	for attempt := range maxAttempts {
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := c.client.Do(req)

		if !isRetryable(err) {
			return resp, err
		}

		isLastAttempt := attempt == maxAttempts-1

		if isLastAttempt {
			return resp, err
		}

		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		time.Sleep(c.delays[attempt])
	}

	return nil, nil
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	var uerr *url.Error
	if errors.As(err, &uerr) {
		if uerr.Err != nil {
			err = uerr.Err
		}
	}

	//  no such host
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// сonnection refused
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		msg := strings.ToLower(opErr.Error())
		if strings.Contains(msg, "connection refused") {
			return true
		}
	}

	// timeout при попытке подключения (dial)
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "dial") {
				return true
			}
		}
	}

	return false
}
