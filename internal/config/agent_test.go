package config_test

import (
	"flag"
	"io"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/mersikovs/korob.git/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type FakeEnv map[string]string

func (f FakeEnv) LookupEnv(key string) (string, bool) {
	value, exists := f[key]
	return value, exists
}

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		env      config.EnvSource
		jsonBody string
		jsonVia  string
		want     *config.Config
		wantErr  string
	}{
		{
			name: "default values",
			args: []string{},
			env:  FakeEnv{},
			want: &config.Config{
				Address:        "localhost:8080",
				ReportInterval: 10,
				PollInterval:   2,
				RateLimit:      1,
			},
		},
		{
			name: "flags values",
			args: []string{"-a", "192.168.1.1:8080", "-r", "5", "-p", "1"},
			env:  FakeEnv{},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 5,
				PollInterval:   1,
				Key:            nil,
				RateLimit:      1,
			},
		},
		{
			name:    "flags value report interval string",
			args:    []string{"-a", "192.168.1.1:8080", "-r", "abc", "-p", "1"},
			env:     FakeEnv{},
			wantErr: "not a number",
		},
		{
			name: "env over flags values",
			args: []string{"-a", "192.168.1.1:8080", "-r", "5", "-p", "1"},
			env: FakeEnv{
				"ADDRESS":         "192.168.1.1:8081",
				"REPORT_INTERVAL": "6",
				"POLL_INTERVAL":   "2",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8081",
				ReportInterval: 6,
				PollInterval:   2,
				Key:            nil,
				RateLimit:      1,
			},
		},
		{
			name: "env over default values",
			args: []string{},
			env: FakeEnv{
				"ADDRESS":         "192.168.1.1:8081",
				"REPORT_INTERVAL": "6",
				"POLL_INTERVAL":   "2",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8081",
				ReportInterval: 6,
				PollInterval:   2,
				Key:            nil,
				RateLimit:      1,
			},
		},
		{
			name: "invalidFlag",
			args: []string{"-a", "192.168.1.1:8080", "-r", "5", "-p", "1", "-invalidFlag", "true"},
			env: FakeEnv{
				"ADDRESS":         "192.168.1.1:8081",
				"REPORT_INTERVAL": "6",
				"POLL_INTERVAL":   "2",
			},
			wantErr: "flag provided but not defined",
		},
		{
			name: "empty env values",
			args: []string{"-a", "192.168.1.1:8080", "-r", "5", "-p", "1"},
			env: FakeEnv{
				"ADDRESS": "",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 5,
				PollInterval:   1,
				Key:            nil,
				RateLimit:      1,
			},
			wantErr: "env ADDRESS is set but empty",
		},
		{
			name: "mixied env and flags values",
			args: []string{"-r", "5", "-p", "1"},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 5,
				PollInterval:   1,
				Key:            nil,
				RateLimit:      1,
			},
		},
		{
			name: "mixied env and flags and default values",
			args: []string{"-p", "1"},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   1,
				Key:            nil,
				RateLimit:      1,
			},
		},
		{
			name: "key value in args",
			args: []string{"-k", "1"},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   2,
				Key:            new("1"),
				RateLimit:      1,
			},
		},
		{
			name: "key value in env",
			args: []string{"-k", "1"},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
				"KEY":     "2",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   2,
				Key:            new("2"),
				RateLimit:      1,
			},
		},
		{
			name: "key value in empty",
			args: []string{"-k", ""},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   2,
				Key:            new(""),
				RateLimit:      1,
			},
		},
		{
			name: "key value in env only",
			args: []string{"-k", ""},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
				"KEY":     "3",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   2,
				Key:            new("3"),
				RateLimit:      1,
			},
		},
		{
			name: "ratelimit value in env",
			args: []string{"-l", "3"},
			env: FakeEnv{
				"ADDRESS":    "192.168.1.1:8080",
				"RATE_LIMIT": "2",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   2,
				RateLimit:      2,
			},
		},
		{
			name: "ratelimit value empty",
			args: []string{"-l", ""},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   2,
				Key:            nil,
				RateLimit:      1,
			},
			wantErr: "not a number",
		},
		{
			name: "ratelimit in env over",
			args: []string{"-l", ""},
			env: FakeEnv{
				"ADDRESS":    "192.168.1.1:8080",
				"RATE_LIMIT": "5",
			},
			wantErr: "not a number",
		},
		{
			name: "ratelimit value in env only",
			args: []string{"-p", "-10"},
			env: FakeEnv{
				"ADDRESS":    "192.168.1.1:8080",
				"RATE_LIMIT": "3",
			},
			wantErr: "must be a positive integer",
		},
		{
			name:     "json over default",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"address":"json:9090","report_interval":42,"poll_interval":7,"rate_limit":3}`,
			want: &config.Config{
				Address:        "json:9090",
				ReportInterval: 42,
				PollInterval:   7,
				RateLimit:      3,
			},
		},
		{
			name:     "flag over json",
			args:     []string{"-a", "flag:8080"},
			env:      FakeEnv{},
			jsonBody: `{"address":"json:9090","report_interval":42}`,
			want: &config.Config{
				Address:        "flag:8080",
				ReportInterval: 42,
				PollInterval:   2,
				RateLimit:      1,
			},
		},
		{
			name:     "env over json",
			args:     nil,
			env:      FakeEnv{"ADDRESS": "env:7070"},
			jsonBody: `{"address":"json:9090","report_interval":42}`,
			want: &config.Config{
				Address:        "env:7070",
				ReportInterval: 42,
				PollInterval:   2,
				RateLimit:      1,
			},
		},
		{
			name:     "json via CONFIG env",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"address":"json:9090","report_interval":42}`,
			jsonVia:  "env",
			want: &config.Config{
				Address:        "json:9090",
				ReportInterval: 42,
				PollInterval:   2,
				RateLimit:      1,
			},
		},
		{
			name:     "json via -config",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"address":"json:9090","report_interval":42}`,
			jsonVia:  "flag-config",
			want: &config.Config{
				Address:        "json:9090",
				ReportInterval: 42,
				PollInterval:   2,
				RateLimit:      1,
			},
		},
		{
			name:     "invalid JSON via -c",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"address": }`,
			wantErr:  "load json config",
		},
		{
			name:     "invalid JSON via CONFIG env",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"address": }`,
			jsonVia:  "env",
			wantErr:  "load json config",
		},
		{
			name:    "missing config file via -c",
			args:    []string{"-c", "/nonexistent/config.json"},
			env:     FakeEnv{},
			wantErr: "load json config",
		},
		{
			name:    "missing config file via CONFIG env",
			args:    nil,
			env:     FakeEnv{"CONFIG": "/nonexistent/config.json"},
			wantErr: "load json config",
		},
		{
			name:    "empty CONFIG env",
			args:    nil,
			env:     FakeEnv{"CONFIG": ""},
			wantErr: "CONFIG",
		},
		{
			name:    "flag -c without value",
			args:    []string{"-c"},
			env:     FakeEnv{},
			wantErr: "requires an argument",
		},
		{
			name: "flag crypto key",
			args: []string{"-crypto-key", "/tmp/pub.key"},
			env:  FakeEnv{},
			want: &config.Config{
				Address:        "localhost:8080",
				ReportInterval: 10,
				PollInterval:   2,
				RateLimit:      1,
				CryptoKey:      new("/tmp/pub.key"),
			},
		},
		{
			name: "env crypto key",
			args: nil,
			env:  FakeEnv{"CRYPTO_KEY": "/tmp/env.key"},
			want: &config.Config{
				Address:        "localhost:8080",
				ReportInterval: 10,
				PollInterval:   2,
				RateLimit:      1,
				CryptoKey:      new("/tmp/env.key"),
			},
		},
		{
			name: "env over flag crypto key",
			args: []string{"-crypto-key", "/tmp/flag.key"},
			env:  FakeEnv{"CRYPTO_KEY": "/tmp/env.key"},
			want: &config.Config{
				Address:        "localhost:8080",
				ReportInterval: 10,
				PollInterval:   2,
				RateLimit:      1,
				CryptoKey:      new("/tmp/env.key"),
			},
		},
		{
			name:    "empty CRYPTO_KEY env",
			args:    nil,
			env:     FakeEnv{"CRYPTO_KEY": ""},
			wantErr: "env CRYPTO_KEY is set but empty",
		},
		{
			name:    "invalid REPORT_INTERVAL env",
			args:    nil,
			env:     FakeEnv{"REPORT_INTERVAL": "abc"},
			wantErr: "not a number",
		},
		{
			name:    "zero REPORT_INTERVAL env",
			args:    nil,
			env:     FakeEnv{"REPORT_INTERVAL": "0"},
			wantErr: "must be a positive integer",
		},
		{
			name:    "negative REPORT_INTERVAL env",
			args:    nil,
			env:     FakeEnv{"REPORT_INTERVAL": "-1"},
			wantErr: "must be a positive integer",
		},
		{
			name:    "invalid POLL_INTERVAL env",
			args:    nil,
			env:     FakeEnv{"POLL_INTERVAL": "abc"},
			wantErr: "not a number",
		},
		{
			name:    "invalid RATE_LIMIT env",
			args:    nil,
			env:     FakeEnv{"RATE_LIMIT": "abc"},
			wantErr: "not a number",
		},
		{
			name:    "negative RATE_LIMIT env",
			args:    nil,
			env:     FakeEnv{"RATE_LIMIT": "-1"},
			wantErr: "must be a positive integer",
		},
		{
			name:    "empty ADDRESS env",
			args:    nil,
			env:     FakeEnv{"ADDRESS": ""},
			wantErr: "env ADDRESS is set but empty",
		},
		{
			name:    "empty KEY env",
			args:    nil,
			env:     FakeEnv{"KEY": ""},
			wantErr: "env KEY is set but empty",
		},
		{
			name:     "negative report interval via JSON",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"report_interval":-5}`,
			wantErr:  "report interval must be positive",
		},
		{
			name:     "zero poll interval via JSON",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"poll_interval":0}`,
			wantErr:  "poll interval must be positive",
		},
		{
			name:     "negative rate limit via JSON",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"rate_limit":-1}`,
			wantErr:  "rate limit must be positive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			env := tt.env

			if tt.jsonBody != "" {
				dir := t.TempDir()
				path := filepath.Join(dir, "config.json")
				require.NoError(t, os.WriteFile(path, []byte(tt.jsonBody), 0o600))

				switch tt.jsonVia {
				case "env":
					newEnv := make(FakeEnv, len(tt.env.(FakeEnv))+1)
					maps.Copy(newEnv, tt.env.(FakeEnv))
					newEnv["CONFIG"] = path
					env = newEnv
				case "flag-config":
					args = append(args, "-config", path)
				default:
					args = append(args, "-c", path)
				}
			}

			fs := flag.NewFlagSet("agent_test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)

			got, err := config.Parse(fs, args, env)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
