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

func TestServerParseConfig(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		env      config.EnvSource
		jsonBody string
		jsonVia  string
		want     *config.ServerConfig
		wantErr  string
	}{
		{
			name: "default values",
			args: nil,
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
			args: []string{"-a", "192.168.1.1:8080", "-i", "50", "-f", "file.backup", "-r"},
			env:  FakeEnv{},
			want: &config.ServerConfig{
				Address:         "192.168.1.1:8080",
				StoreInterval:   50,
				FileStoragePath: "file.backup",
				Restore:         true,
			},
		},
		{
			name: "env over flags",
			args: []string{"-a", "192.168.1.1:8080", "-i", "50", "-f", "file.backup", "-r"},
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
			name: "env over default",
			args: nil,
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
			name:     "json over default",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"address":"json:9090","file_storage_path":"json.back","store_interval":42,"restore":true}`,
			want: &config.ServerConfig{
				Address:         "json:9090",
				StoreInterval:   42,
				FileStoragePath: "json.back",
				Restore:         true,
			},
		},
		{
			name:     "flag over json",
			args:     []string{"-a", "flag:8080"},
			env:      FakeEnv{},
			jsonBody: `{"address":"json:9090","file_storage_path":"json.back"}`,
			want: &config.ServerConfig{
				Address:         "flag:8080",
				StoreInterval:   300,
				FileStoragePath: "json.back",
				Restore:         false,
			},
		},
		{
			name: "env over json",
			args: nil,
			env: FakeEnv{
				"ADDRESS": "env:7070",
			},
			jsonBody: `{"address":"json:9090","file_storage_path":"json.back"}`,
			want: &config.ServerConfig{
				Address:         "env:7070",
				StoreInterval:   300,
				FileStoragePath: "json.back",
				Restore:         false,
			},
		},
		{
			name:    "invalid flag",
			args:    []string{"-a", "192.168.1.1:8080", "-invalidFlag", "true"},
			env:     FakeEnv{},
			wantErr: "flag provided but not defined",
		},
		{
			name:    "invalid STORE_INTERVAL",
			args:    nil,
			env:     FakeEnv{"STORE_INTERVAL": "abc"},
			wantErr: "env STORE_INTERVAL",
		},
		{
			name:    "negative STORE_INTERVAL",
			args:    nil,
			env:     FakeEnv{"STORE_INTERVAL": "-1"},
			wantErr: "store interval must be >= 0",
		},
		{
			name:    "invalid RESTORE",
			args:    nil,
			env:     FakeEnv{"RESTORE": "yes"},
			wantErr: "env RESTORE",
		},
		{
			name:    "invalid CACHE_TTL_SEC",
			args:    nil,
			env:     FakeEnv{"CACHE_TTL_SEC": "abc"},
			wantErr: "env CACHE_TTL_SEC",
		},
		{
			name:    "negative CACHE_TTL_SEC",
			args:    nil,
			env:     FakeEnv{"CACHE_TTL_SEC": "-5"},
			wantErr: "cache TTL must be >= 0",
		},
		{
			name:    "empty ADDRESS env",
			args:    nil,
			env:     FakeEnv{"ADDRESS": ""},
			wantErr: "env ADDRESS is set but empty",
		},
		{
			name:    "empty FILE_STORAGE_PATH env",
			args:    nil,
			env:     FakeEnv{"FILE_STORAGE_PATH": ""},
			wantErr: "env FILE_STORAGE_PATH is set but empty",
		},
		{
			name:    "empty AUDIT_FILE env",
			args:    nil,
			env:     FakeEnv{"AUDIT_FILE": ""},
			wantErr: "env AUDIT_FILE is set but empty",
		},
		{
			name:    "empty DATABASE_DSN env",
			args:    nil,
			env:     FakeEnv{"DATABASE_DSN": ""},
			wantErr: "env DATABASE_DSN is set but empty",
		},
		{
			name:     "invalid JSON",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"address": }`,
			wantErr:  "decode config",
		},
		{
			name:    "missing config file",
			args:    []string{"-c", "/nonexistent/config.json"},
			env:     FakeEnv{},
			wantErr: "read config",
		},
		{
			name: "flag secret key",
			args: []string{"-k", "flag-key"},
			env:  FakeEnv{},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
				Key:             new("flag-key"),
			},
		},
		{
			name: "env over flag secret key",
			args: []string{"-k", "flag-key"},
			env:  FakeEnv{"KEY": "env-key"},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
				Key:             new("env-key"),
			},
		},
		{
			name: "env only secret key",
			args: nil,
			env:  FakeEnv{"KEY": "env-only"},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
				Key:             new("env-only"),
			},
		},
		{
			name: "flag audit file",
			args: []string{"--audit-file", "/tmp/audit.log"},
			env:  FakeEnv{},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
				AuditFile:       new("/tmp/audit.log"),
			},
		},
		{
			name: "env over flag audit file",
			args: []string{"--audit-file", "/tmp/flag.log"},
			env:  FakeEnv{"AUDIT_FILE": "/tmp/env.log"},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
				AuditFile:       new("/tmp/env.log"),
			},
		},
		{
			name: "env audit URL only",
			args: nil,
			env:  FakeEnv{"AUDIT_URL": "http://audit.example/"},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
				AuditURL:        new("http://audit.example/"),
			},
		},
		{
			name: "flag database DSN",
			args: []string{"-d", "postgres://flag"},
			env:  FakeEnv{},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
				DatabaseDSN:     new("postgres://flag"),
			},
		},
		{
			name: "env over flag database DSN",
			args: []string{"-d", "postgres://flag"},
			env:  FakeEnv{"DATABASE_DSN": "postgres://env"},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
				DatabaseDSN:     new("postgres://env"),
			},
		},
		{
			name: "flag crypto key",
			args: []string{"--crypto-key", "/tmp/flag.key"},
			env:  FakeEnv{},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
				CryptoKey:       new("/tmp/flag.key"),
			},
		},
		{
			name: "env over flag crypto key",
			args: []string{"--crypto-key", "/tmp/flag.key"},
			env:  FakeEnv{"CRYPTO_KEY": "/tmp/env.key"},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
				CryptoKey:       new("/tmp/env.key"),
			},
		},
		{
			name:     "json via CONFIG env",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"address":"json:9090","file_storage_path":"json.back"}`,
			jsonVia:  "env",
			want: &config.ServerConfig{
				Address:         "json:9090",
				StoreInterval:   300,
				FileStoragePath: "json.back",
				Restore:         false,
			},
		},
		{
			name:     "json via -config alias",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"address":"json:9090","file_storage_path":"json.back"}`,
			jsonVia:  "flag-config",
			want: &config.ServerConfig{
				Address:         "json:9090",
				StoreInterval:   300,
				FileStoragePath: "json.back",
				Restore:         false,
			},
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
			wantErr: "config flag requires an argument",
		},
		{
			name:     "env over json secret",
			args:     nil,
			env:      FakeEnv{"KEY": "env-key"},
			jsonBody: `{"key":"json-key","database_dsn":"postgres://json"}`,
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
				Key:             new("env-key"),
				DatabaseDSN:     new("postgres://json"),
			},
		},
		{
			name:     "flag over json secret",
			args:     []string{"-k", "flag-key"},
			env:      FakeEnv{},
			jsonBody: `{"key":"json-key","database_dsn":"postgres://json"}`,
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				Restore:         false,
				Key:             new("flag-key"),
				DatabaseDSN:     new("postgres://json"),
			},
		},
		{
			name: "flag crypto key",
			args: []string{"--crypto-key", "/tmp/flag.key"},
			env:  FakeEnv{},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				CryptoKey:       new("/tmp/flag.key"),
			},
		},
		{
			name: "env over flag crypto key",
			args: []string{"--crypto-key", "/tmp/flag.key"},
			env:  FakeEnv{"CRYPTO_KEY": "/tmp/env.key"},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				CryptoKey:       new("/tmp/env.key"),
			},
		},
		{
			name:    "negative STORE_INTERVAL",
			args:    nil,
			env:     FakeEnv{"STORE_INTERVAL": "-1"},
			wantErr: "store interval must be >= 0",
		},
		{
			name:    "negative CACHE_TTL_SEC",
			args:    nil,
			env:     FakeEnv{"CACHE_TTL_SEC": "-5"},
			wantErr: "cache TTL must be >= 0",
		},
		{
			name:     "invalid JSON via CONFIG env",
			args:     nil,
			env:      FakeEnv{},
			jsonBody: `{"address": }`, // битый JSON
			jsonVia:  "env",
			wantErr:  "failed load json config env variable",
		},
		{
			name:    "missing config file via CONFIG env",
			args:    nil,
			env:     FakeEnv{"CONFIG": "/nonexistent/config.json"},
			wantErr: "failed load json config env variable",
		},
		{
			name: "flag crypto key",
			args: []string{"--crypto-key", "/tmp/flag.key"},
			env:  FakeEnv{},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				CryptoKey:       new("/tmp/flag.key"),
			},
		},
		{
			name: "env crypto key",
			args: nil,
			env:  FakeEnv{"CRYPTO_KEY": "/tmp/env.key"},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				CryptoKey:       new("/tmp/env.key"),
			},
		},
		{
			name: "flag audit URL",
			args: []string{"--audit-url", "http://flag.example/"},
			env:  FakeEnv{},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				AuditURL:        new("http://flag.example/"),
			},
		},
		{
			name: "env audit URL",
			args: nil,
			env:  FakeEnv{"AUDIT_URL": "http://env.example/"},
			want: &config.ServerConfig{
				Address:         "localhost:8080",
				StoreInterval:   300,
				FileStoragePath: "store.back",
				AuditURL:        new("http://env.example/"),
			},
		},
		{
			name:    "empty AUDIT_FILE env",
			args:    nil,
			env:     FakeEnv{"AUDIT_FILE": ""},
			wantErr: "env AUDIT_FILE is set but empty",
		},
		{
			name:    "empty KEY env",
			args:    nil,
			env:     FakeEnv{"KEY": ""},
			wantErr: "env KEY is set but empty",
		},
		{
			name:    "empty DATABASE_DSN env",
			args:    nil,
			env:     FakeEnv{"DATABASE_DSN": ""},
			wantErr: "env DATABASE_DSN is set but empty",
		},
		{
			name:    "empty CRYPTO_KEY env",
			args:    nil,
			env:     FakeEnv{"CRYPTO_KEY": ""},
			wantErr: "env CRYPTO_KEY is set but empty",
		},
		{
			name:    "empty AUDIT_URL env",
			args:    nil,
			env:     FakeEnv{"AUDIT_URL": ""},
			wantErr: "env AUDIT_URL is set but empty",
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
				default: // "flag"
					args = append(args, "-c", path)
				}
			}

			fs := flag.NewFlagSet("server_test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)

			got, err := config.ServerParseConfig(fs, args, env)

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
