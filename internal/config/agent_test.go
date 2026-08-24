package config_test

import (
	"flag"
	"testing"

	"github.com/mersikovs/korob.git/internal/config"
	"github.com/stretchr/testify/assert"
)

type FakeEnv map[string]string

func (f FakeEnv) LookupEnv(key string) (string, bool) {
	value, exists := f[key]
	return value, exists
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		fs      *flag.FlagSet
		args    []string
		env     config.EnvSource
		want    *config.Config
		wantErr bool
	}{
		{
			name: "default values",
			fs:   flag.NewFlagSet("agent_test", flag.ContinueOnError),
			args: []string{},
			env:  FakeEnv{},
			want: &config.Config{
				Address:        "localhost:8080",
				ReportInterval: 10,
				PollInterval:   2,
			},
		},
		{
			name: "flags values",
			fs:   flag.NewFlagSet("agent_test", flag.ContinueOnError),
			args: []string{"-a", "192.168.1.1:8080", "-r", "5", "-p", "1"},
			env:  FakeEnv{},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 5,
				PollInterval:   1,
				Key:            nil,
			},
		},
		{
			name: "flags value report interval string",
			fs:   flag.NewFlagSet("agent_test", flag.ContinueOnError),
			args: []string{"-a", "192.168.1.1:8080", "-r", "abc", "-p", "1"},
			env:  FakeEnv{},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   1,
				Key:            nil,
			},
		},
		{
			name: "env over flags values",
			fs:   flag.NewFlagSet("agent_test", flag.ContinueOnError),
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
			},
		},
		{
			name: "env over default values",
			fs:   flag.NewFlagSet("agent_test", flag.ContinueOnError),
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
			},
		},
		{
			name: "invalidFlag",
			fs:   flag.NewFlagSet("agent_test", flag.ContinueOnError),
			args: []string{"-a", "192.168.1.1:8080", "-r", "5", "-p", "1", "-invalidFlag", "true"},
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
			},
			wantErr: true,
		},
		{
			name: "empty env values",
			fs:   flag.NewFlagSet("agent_test", flag.ContinueOnError),
			args: []string{"-a", "192.168.1.1:8080", "-r", "5", "-p", "1"},
			env: FakeEnv{
				"ADDRESS":         "",
				"REPORT_INTERVAL": "",
				"POLL_INTERVAL":   "",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 5,
				PollInterval:   1,
				Key:            nil,
			},
		},
		{
			name: "mixied env and flags values",
			fs:   flag.NewFlagSet("agent_test", flag.ContinueOnError),
			args: []string{"-r", "5", "-p", "1"},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 5,
				PollInterval:   1,
				Key:            nil,
			},
		},
		{
			name: "mixied env and flags and default values",
			fs:   flag.NewFlagSet("test", flag.ContinueOnError),
			args: []string{"-p", "1"},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   1,
				Key:            nil,
			},
		},
		{
			name: "key value in args",
			fs:   flag.NewFlagSet("test", flag.ContinueOnError),
			args: []string{"-k", "1"},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   2,
				Key:            []byte("1"),
			},
		},
		{
			name: "key value in env",
			fs:   flag.NewFlagSet("test", flag.ContinueOnError),
			args: []string{"-k", "1"},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
				"KEY":     "2",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   2,
				Key:            []byte("2"),
			},
		},
		{
			name: "key value in empty",
			fs:   flag.NewFlagSet("test", flag.ContinueOnError),
			args: []string{"-k", ""},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   2,
				Key:            nil,
			},
		},
		{
			name: "key value in env only",
			fs:   flag.NewFlagSet("test", flag.ContinueOnError),
			args: []string{"-k", ""},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
				"KEY":     "3",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   2,
				Key:            []byte("3"),
			},
		},
		{
			name: "key value in env only",
			fs:   flag.NewFlagSet("test", flag.ContinueOnError),
			args: []string{"-p", "-10"},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
				"KEY":     "3",
			},
			want: &config.Config{
				Address:        "192.168.1.1:8080",
				ReportInterval: 10,
				PollInterval:   2,
				Key:            []byte("3"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := config.Parse(tt.fs, tt.args, tt.env)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Parse() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Parse() succeeded unexpectedly")
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
