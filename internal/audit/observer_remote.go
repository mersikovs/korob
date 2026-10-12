package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/mersikovs/korob.git/internal/logger"
	models "github.com/mersikovs/korob.git/internal/model"
)

type RemoteObserver struct {
	ID     string
	client *retryablehttp.Client
	url    string
	logger logger.Logger
}

func NewRemoteObserver(id, targetURL string, log logger.Logger) *RemoteObserver {

	client := retryablehttp.NewClient()
	client.HTTPClient.Timeout = 10 * time.Second
	client.RetryMax = 3
	client.RetryWaitMin = 500 * time.Millisecond
	client.RetryWaitMax = 30 * time.Second
	client.Logger = log

	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			return true, nil
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			return true, nil
		}

		if resp.StatusCode >= 500 {
			return true, nil
		}

		if resp.StatusCode >= 400 {
			return false, fmt.Errorf("client error: %d", resp.StatusCode)
		}

		return false, nil
	}

	return &RemoteObserver{
		ID:     id,
		client: client,
		url:    targetURL,
		logger: log,
	}
}

func (r *RemoteObserver) Notify(ctx context.Context, event models.AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		r.logger.Info("remote observer: failed to marshal event", "error", err)
		return
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, r.url, bytes.NewReader(data))
	if err != nil {
		r.logger.Info("remote observer: failed to create request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		r.logger.Info("remote observer: request failed after all retries", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		r.logger.Info("remote observer: received error status",
			"status", resp.StatusCode,
			"request_ts", event.Ts)
		return
	}

	r.logger.Info("remote observer: event sent successfully",
		"status", resp.StatusCode,
		"request_ts", event.Ts)
}

func (r *RemoteObserver) GetID() string {
	return r.ID
}

func (r *RemoteObserver) Close() error {
	if r.client != nil && r.client.HTTPClient != nil {
		r.client.HTTPClient.CloseIdleConnections()
	}
	return nil
}
