package config

import (
	"flag"
	"fmt"
	"os"
)

const defaultAddress = "localhost:8080"
const defaultReportInterval = 10
const defaultPollInterval = 2
const defaultRateLimit = 1

// Config holds the client agent configuration, including server address,
// sign key, and operational intervals.
type Config struct {
	// Address is the network address to sent metrics info
	Address string `json:"address"`
	// CryptoKey is the local file path for crypto key.
	CryptoKey *string `json:"crypto_key"`
	// Key string to sign message
	Key *string `json:"key"`
	// ReportInterval metric data submission interval
	ReportInterval int `json:"report_interval"`
	// PollInterval metric collection interval
	PollInterval int `json:"poll_interval"`
	//RateLimit limit on the number of requests
	RateLimit int `json:"rate_limit"`
}

// EnvSource abstracts environment variable access, allowing for testable
// configuration parsing without direct OS dependencies.
type EnvSource interface {
	LookupEnv(key string) (string, bool)
}

// OSenv is a production implementation of EnvSource that reads
// environment variables directly from the operating system.
type OSenv struct{}

// LookupEnv retrieves the value of the environment variable named by the key.
func (e OSenv) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// Parse builds a Config by merging command-line flags and environment variables,
// with environment variables taking precedence. It also validates that interval
// and rate limit values are positive integers.
func Parse(fs *flag.FlagSet, args []string, env EnvSource) (*Config, error) {
	cnf := &Config{
		Address:        defaultAddress,
		ReportInterval: defaultReportInterval,
		PollInterval:   defaultPollInterval,
		RateLimit:      defaultRateLimit,
	}

	flagPath, flagIsSet, err := scanConfigFlag(args, defaultConfigPathSource.FlagAliases)
	if err != nil {
		return nil, fmt.Errorf("config flag %v: %w", defaultConfigPathSource.FlagAliases, err)
	}
	envPath, envIsSet, err := lookupConfigEnv(env, defaultConfigPathSource.EnvName)
	if err != nil {
		return nil, fmt.Errorf("config env %s: %w", defaultConfigPathSource.EnvName, err)
	}
	if envIsSet {
		if err := loadJSONConfig(envPath, cnf); err != nil {
			return nil, fmt.Errorf("load json config from env %s: %w", defaultConfigPathSource.EnvName, err)
		}
	} else if flagIsSet {
		if err := loadJSONConfig(flagPath, cnf); err != nil {
			return nil, fmt.Errorf("load json config from flag: %w", err)
		}
	}

	fs.StringVar(&cnf.Address, "a", cnf.Address, "адрес эндпоинта HTTP-сервера")
	signKey := OptionalStringFlag(fs, "k", "ключ подписи данных")
	cryptoKey := OptionalStringFlag(fs, "crypto-key", "путь к файлу с публичным ключом")

	fs.Func("r", "частота отправки метрик на сервер", positiveIntFlag("r", &cnf.ReportInterval))
	fs.Func("p", "частота опроса метрик", positiveIntFlag("p", &cnf.PollInterval))
	fs.Func("l", "количество одновременно исходящих запросов на сервер", positiveIntFlag("l", &cnf.RateLimit))

	// Stubs for -c/-config: the values have already been processed by scanConfigFlag.
	fs.Func("c", "путь к файлу конфигурации", func(string) error { return nil })
	fs.Func("config", "путь к файлу конфигурации (аналог -c)", func(string) error { return nil })

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	applyIfSetPtr(&cnf.Key, signKey.Value)
	applyIfSetPtr(&cnf.CryptoKey, cryptoKey.Value)

	{
		addr, err := lookupNonEmptyEnv(env, "ADDRESS")
		if err != nil {
			return nil, err
		}
		applyIfSet(&cnf.Address, addr)
	}

	{
		envKey, err := lookupNonEmptyEnv(env, "KEY")
		if err != nil {
			return nil, err
		}
		applyIfSetPtr(&cnf.Key, envKey)
	}

	{
		envCrypto, err := lookupNonEmptyEnv(env, "CRYPTO_KEY")
		if err != nil {
			return nil, err
		}
		applyIfSetPtr(&cnf.CryptoKey, envCrypto)
	}

	if reportInterval, exists := env.LookupEnv("REPORT_INTERVAL"); exists {
		ri, err := getPositiveIntEnv("REPORT_INTERVAL", reportInterval)
		if err != nil {
			return nil, err
		}
		cnf.ReportInterval = ri
	}

	if pollInterval, exists := env.LookupEnv("POLL_INTERVAL"); exists {
		pi, err := getPositiveIntEnv("POLL_INTERVAL", pollInterval)
		if err != nil {
			return nil, err
		}
		cnf.PollInterval = pi
	}

	if rateLimit, exists := env.LookupEnv("RATE_LIMIT"); exists {
		rl, err := getPositiveIntEnv("RATE_LIMIT", rateLimit)
		if err != nil {
			return nil, err
		}
		cnf.RateLimit = rl
	}

	if cnf.ReportInterval <= 0 {
		return nil, fmt.Errorf("report interval must be positive, got %d", cnf.ReportInterval)
	}
	if cnf.PollInterval <= 0 {
		return nil, fmt.Errorf("poll interval must be positive, got %d", cnf.PollInterval)
	}
	if cnf.RateLimit <= 0 {
		return nil, fmt.Errorf("rate limit must be positive, got %d", cnf.RateLimit)
	}

	return cnf, nil
}
