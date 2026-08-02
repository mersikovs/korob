package config_test

import (
	"flag"
	"testing"

	"github.com/mersikovs/korob.git/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestServerParseConfig(t *testing.T) {
	tests := []struct {
		name    string
		fs      *flag.FlagSet
		args    []string
		env     config.EnvSource
		want    *config.ServerConfig
		wantErr bool
	}{
		{
			name: "default values",
			fs:   flag.NewFlagSet("server_test", flag.ContinueOnError),
			args: []string{},
			env:  FakeEnv{},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
			},
		},
		{
			name: "flags values",
			fs:   flag.NewFlagSet("server_test", flag.ContinueOnError),
			args: []string{"-a", "192.168.1.1:8080", "-i", "50", "-f", "file.backup", "-r", "true"},
			env:  FakeEnv{},
			want: &config.ServerConfig{
				Address:         "192.168.1.1:8080",
				StoreInterval:   50,
				FileStoragePath: "file.backup",
				Restore:         true,
			},
		},
		{
			name: "env over flags values",
			fs:   flag.NewFlagSet("server_test", flag.ContinueOnError),
			args: []string{"-a", "192.168.1.1:8080", "-i", "50", "-f", "file.backup", "-r", "true"},
			env: FakeEnv{
				"ADDRESS":           "192.168.1.1:8081",
				"STORE_INTERVAL":    "100",
				"FILE_STORAGE_PATH": "env.backup",
				"RESTORE":           "false",
			},
			want: &config.ServerConfig{
				Address:         "192.168.1.1:8081",
				StoreInterval:   100,
				FileStoragePath: "env.backup",
				Restore:         false,
			},
		},
		{
			name: "env over default values",
			fs:   flag.NewFlagSet("server_test", flag.ContinueOnError),
			args: []string{},
			env: FakeEnv{
				"ADDRESS":           "192.168.1.1:8082",
				"STORE_INTERVAL":    "150",
				"FILE_STORAGE_PATH": "env2.backup",
				"RESTORE":           "true",
			},
			want: &config.ServerConfig{
				Address:         "192.168.1.1:8082",
				StoreInterval:   150,
				FileStoragePath: "env2.backup",
				Restore:         true,
			},
		},
		{
			name: "invalidFlag",
			fs:   flag.NewFlagSet("server_test", flag.ContinueOnError),
			args: []string{"-a", "192.168.1.1:8080", "-invalidFlag", "true"},
			env: FakeEnv{
				"ADDRESS":           "192.168.1.1:8081",
				"STORE_INTERVAL":    "6",
				"FILE_STORAGE_PATH": "2",
				"RESTORE":           "true",
			},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
			},
			wantErr: true,
		},
		{
			name: "empty env values",
			fs:   flag.NewFlagSet("server_test", flag.ContinueOnError),
			args: []string{"-a", "192.168.1.1:8080", "-i", "50", "-f", "file.backup", "-r", "true"},
			env: FakeEnv{
				"ADDRESS":           "",
				"STORE_INTERVAL":    "",
				"FILE_STORAGE_PATH": "",
				"RESTORE":           "",
			},
			want: &config.ServerConfig{
				Address:         "192.168.1.1:8080",
				StoreInterval:   50,
				FileStoragePath: "",
				Restore:         true,
			},
		},
		{
			name: "mixied env and flags values",
			fs:   flag.NewFlagSet("server_test", flag.ContinueOnError),
			args: []string{"-r", "true", "-p", "1"},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
			},
			want: &config.ServerConfig{
				Address:         "192.168.1.1:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         true,
			},
		},
		{
			name: "mixied env and flags and default values",
			fs:   flag.NewFlagSet("server_test", flag.ContinueOnError),
			args: []string{"-f", "file.backup", "-r"},
			env: FakeEnv{
				"ADDRESS": "192.168.1.1:8080",
			},
			want: &config.ServerConfig{
				Address:         "192.168.1.1:8080",
				StoreInterval:   300,
				FileStoragePath: "file.backup",
				Restore:         true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := config.ServerParseConfig(tt.fs, tt.args, tt.env)

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
