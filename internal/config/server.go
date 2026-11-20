package config

import (
	"flag"
	"fmt"
	"strconv"
)

const defaultServerAddres = "localhost:8080"
const defaultStoreInterval = 300            // for autotest
const defaultFileStoragePath = "store.back" // for autotest
const defaultRestore = false

// ServerConfig holds the runtime configuration for the metrics server.
type ServerConfig struct {
	// Address is the network address to listen on
	Address string `json:"address"`
	// AuditFile is the local file path for audit logging.
	AuditFile *string `json:"audit_file"`
	// AuditURL is the remote HTTP endpoint for audit logging.
	AuditURL *string `json:"audit_url"`
	// CacheTTLSec is the cache TTL in seconds (0 disables caching).
	CacheTTLSec int `json:"cache_ttl_sec"`
	// Key is the secret key used for HMAC-SHA256 payload signing.
	Key *string `json:"key"`
	// DatabaseDSN is the PostgreSQL connection string.
	// If empty, the server falls back to in-memory storage.
	DatabaseDSN *string `json:"database_dsn"`
	// FileStoragePath is the path for periodic file-based persistence.
	FileStoragePath string `json:"file_storage_path"`
	// Restore indicates whether to load metrics from FileStoragePath on startup.
	Restore bool `json:"restore"`
	// StoreInterval is the period in seconds for saving metrics to FileStoragePath.
	StoreInterval int `json:"store_interval"`
	// CryptoKey is the local file path for crypto key.
	CryptoKey *string `json:"crypto_key"`
}

type configPathSource struct {
	FlagAliases []string
	EnvName     string
}

var defaultConfigPathSource = configPathSource{
	FlagAliases: []string{"-c", "-config"},
	EnvName:     "CONFIG",
}

// ServerParseConfig builds server configuration by merging command-line flags
// and environment variables, with environment variables taking precedence.
func ServerParseConfig(fs *flag.FlagSet, args []string, env EnvSource) (*ServerConfig, error) {
	cnf := &ServerConfig{
		Address:         defaultServerAddres,
		StoreInterval:   defaultStoreInterval,
		FileStoragePath: defaultFileStoragePath,
		Restore:         defaultRestore,
	}

	flagPath, flagIsSet, err := scanConfigFlag(args, defaultConfigPathSource.FlagAliases)
	if err != nil {
		return nil, fmt.Errorf("%s %v: %w", "not valid value one of flags", defaultConfigPathSource.FlagAliases, err)
	}

	envPath, envIsSet, err := lookupConfigEnv(env, defaultConfigPathSource.EnvName)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", "not valid env value", defaultConfigPathSource.EnvName, ErrConfigPathEmpty)
	}

	if envIsSet {
		err = loadJSONConfig(envPath, cnf)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", "failed load json config env variable", err)
		}
	} else if flagIsSet {
		err = loadJSONConfig(flagPath, cnf)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", "failed load json config flag", err)
		}
	}

	fs.StringVar(&cnf.Address, "a", cnf.Address, "адрес эндпоинта HTTP-сервера")
	fs.IntVar(&cnf.StoreInterval, "i", cnf.StoreInterval, "интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск")
	fs.StringVar(&cnf.FileStoragePath, "f", cnf.FileStoragePath, "путь до файла, куда сохраняются текущие значения")
	fs.BoolVar(&cnf.Restore, "r", cnf.Restore, "следует ли загружать ранее сохранённые значения")

	auditFile := OptionalStringFlag(fs, "audit-file", "путь к файлу, в который сохраняются логи аудита")
	auditURL := OptionalStringFlag(fs, "audit-url", "полный URL, по которому отправляются логи аудита")
	dsn := OptionalSecretFlag(fs, "d", "строка подключения к БД (DSN)")
	signKey := OptionalSecretFlag(fs, "k", "ключ подписи данных")
	cryptoKey := OptionalSecretFlag(fs, "crypto-key", "путь к файлу, передайте путь до файла с приватным ключом")

	// Stubs for -c/-config: the values have already been processed by scanConfigFlag.
	fs.Func("c", "путь к файлу конфигураци, передайте путь до файла с конфигурацией", func(string) error { return nil })
	fs.Func("config", "путь к файлу конфигурации(аналог c), передайте путь до файла с конфигурацией", func(string) error { return nil })

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	applyIfSetPtr(&cnf.AuditFile, auditFile.Value)
	applyIfSetPtr(&cnf.AuditURL, auditURL.Value)
	applyIfSetPtr(&cnf.DatabaseDSN, dsn.Value)
	applyIfSetPtr(&cnf.Key, signKey.Value)
	applyIfSetPtr(&cnf.CryptoKey, cryptoKey.Value)

	if addr, err := lookupNonEmptyEnv(env, "ADDRESS"); err != nil {
		return nil, err
	} else if addr != nil {
		cnf.Address = *addr
	}

	if raw, exists := env.LookupEnv("STORE_INTERVAL"); exists {
		si, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("env STORE_INTERVAL: %w", err)
		}
		cnf.StoreInterval = si
	}

	if fsp, err := lookupNonEmptyEnv(env, "FILE_STORAGE_PATH"); err != nil {
		return nil, err
	} else if fsp != nil {
		cnf.FileStoragePath = *fsp
	}

	if raw, exists := env.LookupEnv("RESTORE"); exists {
		r, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("env RESTORE: %w", err)
		}
		cnf.Restore = r
	}

	{
		aFile, err := lookupNonEmptyEnv(env, "AUDIT_FILE")
		if err != nil {
			return nil, err
		}
		applyIfSetPtr(&cnf.AuditFile, aFile)
	}

	{
		aURL, err := lookupNonEmptyEnv(env, "AUDIT_URL")
		if err != nil {
			return nil, err
		}
		applyIfSetPtr(&cnf.AuditURL, aURL)
	}

	{
		envDsn, err := lookupNonEmptyEnv(env, "DATABASE_DSN")
		if err != nil {
			return nil, err
		}
		applyIfSetPtr(&cnf.DatabaseDSN, envDsn)
	}

	{
		envKey, err := lookupNonEmptyEnv(env, "KEY")
		if err != nil {
			return nil, err
		}
		applyIfSetPtr(&cnf.Key, envKey)
	}

	if raw, exists := env.LookupEnv("CACHE_TTL_SEC"); exists {
		cache, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("env CACHE_TTL_SEC: %w", err)
		}
		cnf.CacheTTLSec = cache
	}

	{
		cFile, err := lookupNonEmptyEnv(env, "CRYPTO_KEY")
		if err != nil {
			return nil, err
		}
		applyIfSetPtr(&cnf.CryptoKey, cFile)
	}

	if cnf.StoreInterval < 0 {
		return nil, fmt.Errorf("store interval must be >= 0, got %d", cnf.StoreInterval)
	}
	if cnf.CacheTTLSec < 0 {
		return nil, fmt.Errorf("cache TTL must be >= 0, got %d", cnf.CacheTTLSec)
	}

	return cnf, nil
}
