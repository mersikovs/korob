package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

const defaultAddress = "localhost:8080"
const defaultReportInterval = 10
const defaultPollInterval = 2
const defaultKey = ""
const defaultRateLimit = 1

// Config holds the client agent configuration, including server address,
// sign key, and operational intervals.
type Config struct {
	Address        string
	Key            []byte
	ReportInterval int
	PollInterval   int
	RateLimit      int
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

func positiveIntFlag(target *int) func(string) error {
	return func(s string) error {
		v, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("параметр не число %q: %w", s, err)
		}
		if v <= 0 {
			return fmt.Errorf("параметр не положительное число %d", v)
		}
		*target = v
		return nil
	}
}

func getPositiveIntEnv(key, value string) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil {
		return v, fmt.Errorf("параметр не число %q: %w", key, err)
	}
	if v <= 0 {
		return v, fmt.Errorf("параметр %s не положительное число %d", key, v)
	}

	return v, nil
}

// Parse builds a Config by merging command-line flags and environment variables,
// with environment variables taking precedence. It also validates that interval
// and rate limit values are positive integers.
func Parse(fs *flag.FlagSet, args []string, env EnvSource) (*Config, error) {
	cnf := &Config{}
	key := ""
	fs.StringVar(&cnf.Address, "a", defaultAddress, "адрес эндпоинта HTTP-сервера")
	fs.StringVar(&key, "k", defaultKey, "ключ подписи данных")

	cnf.ReportInterval = defaultReportInterval
	fs.Func("r", "частота отправки метрик на сервер", positiveIntFlag(&cnf.ReportInterval))

	cnf.PollInterval = defaultPollInterval
	fs.Func("p", "частота опроса метрик", positiveIntFlag(&cnf.PollInterval))

	cnf.RateLimit = defaultRateLimit
	fs.Func("l", "количество одновременно исходящих запросов на сервер", positiveIntFlag(&cnf.RateLimit))

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if addr, exists := env.LookupEnv("ADDRESS"); exists && addr != "" {
		cnf.Address = addr
	}

	if envKey, exists := env.LookupEnv("KEY"); exists && envKey != "" {
		key = envKey
	}

	if len(key) > 0 {
		cnf.Key = []byte(key)
	}

	if reportInterval, exists := env.LookupEnv("REPORT_INTERVAL"); exists {
		ri, err := getPositiveIntEnv("REPORT_INTERVAL", reportInterval)
		if err != nil {
			return nil, fmt.Errorf("ошибка в env переменной %q: %w", "REPORT_INTERVAL", err)
		}
		cnf.ReportInterval = ri
	}

	if pollInterval, exists := env.LookupEnv("POLL_INTERVAL"); exists {
		pi, err := getPositiveIntEnv("POLL_INTERVAL", pollInterval)
		if err != nil {
			return nil, fmt.Errorf("ошибка в env переменной %q: %w", "POLL_INTERVAL", err)
		}
		cnf.PollInterval = pi
	}

	if rateLimit, exists := env.LookupEnv("RATE_LIMIT"); exists {
		rl, err := getPositiveIntEnv("RATE_LIMIT", rateLimit)
		if err != nil {
			return nil, fmt.Errorf("ошибка в env переменной %q: %w", "RATE_LIMIT", err)
		}
		cnf.RateLimit = rl
	}

	return cnf, nil
}
