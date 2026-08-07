package repository

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isPreExecutionError(t *testing.T) {
	dnsErr := &net.DNSError{
		Err:  "no such host",
		Name: "practicum.yandex.ru",
	}

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "random error",
			err:  errors.New("random error"),
			want: false,
		},
		{
			name: "dns error",
			err:  dnsErr,
			want: true,
		},
		{
			name: "wrapped dns error",
			err:  fmt.Errorf("request failed: %w", dnsErr),
			want: true,
		},
		{
			name: "url error with dns error",
			err: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: dnsErr,
			},
			want: true,
		},
		{
			name: "connection refused",
			err:  errors.New("dial tcp 127.0.0.1:5432: connect: connection refused"),
			want: true,
		},
		{
			name: "connection refused in different case",
			err:  errors.New("Connection Refused"),
			want: true,
		},
		{
			name: "dial timeout",
			err:  errors.New("dial tcp 10.0.0.1:5432: i/o timeout"),
			want: true,
		},
		{
			name: "dial only",
			err:  errors.New("dial"),
			want: false,
		},
		{
			name: "timeout only",
			err:  errors.New("timeout"),
			want: false,
		},
		{
			name: "connection reset by peer",
			err:  errors.New("connection reset by peer"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPreExecutionError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}
