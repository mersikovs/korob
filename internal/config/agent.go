package config

import (
	"flag"
	"os"
	"strconv"
)

const defaultAddress = "localhost:8080"
const defaultReportInterval = 10
const defaultPollInterval = 2
const defaultKey = ""
const defaultRateLimit = 1

type Config struct {
	Address        string
	Key            []byte
	ReportInterval int
	PollInterval   int
	RateLimit      int
}

type EnvSource interface {
	LookupEnv(key string) (string, bool)
}

type OSenv struct{}

func (e OSenv) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

func Parse(fs *flag.FlagSet, args []string, env EnvSource) (*Config, error) {
	cnf := &Config{}
	key := ""
	fs.StringVar(&cnf.Address, "a", defaultAddress, "адрес эндпоинта HTTP-сервера")
	fs.StringVar(&key, "k", defaultKey, "ключ подписи данных")

	cnf.ReportInterval = defaultReportInterval
	fs.Func("r", "частота отправки метрик на сервер", func(s string) error {
		v, err := strconv.Atoi(s)
		if err == nil && v > 0 {
			cnf.ReportInterval = v
		}
		return nil
	})

	cnf.PollInterval = defaultPollInterval
	fs.Func("p", "частота опроса метрик", func(s string) error {
		v, err := strconv.Atoi(s)
		if err == nil && v > 0 {
			cnf.PollInterval = v
		}
		return nil
	})

	cnf.RateLimit = defaultRateLimit
	fs.Func("l", "количество одновременно исходящих запросов на сервер", func(s string) error {
		v, err := strconv.Atoi(s)
		if err == nil && v > 0 {
			cnf.RateLimit = v
		}
		return nil
	})

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
		ri, err := strconv.Atoi(reportInterval)
		if err == nil && ri > 0 {
			cnf.ReportInterval = ri
		}
	}

	if pollInterval, exists := env.LookupEnv("POLL_INTERVAL"); exists {
		pi, err := strconv.Atoi(pollInterval)
		if err == nil && pi > 0 {
			cnf.PollInterval = pi
		}
	}

	if rateLimit, exists := env.LookupEnv("RATE_LIMIT"); exists {
		rl, err := strconv.Atoi(rateLimit)
		if err == nil && rl > 0 {
			cnf.RateLimit = rl
		}
	}

	return cnf, nil
}
