package config

import (
	"flag"
	"os"
	"strconv"
)

type Config struct {
	Address        string
	ReportInterval int
	PollInterval   int
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

	fs.StringVar(&cnf.Address, "a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	fs.IntVar(&cnf.ReportInterval, "r", 10, "частота отправки метрик на сервер")
	fs.IntVar(&cnf.PollInterval, "p", 2, "частота опроса метрик")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if addr, exists := env.LookupEnv("ADDRESS"); exists && addr != "" {
		cnf.Address = addr
	}

	if reportInterval, exists := env.LookupEnv("REPORT_INTERVAL"); exists {
		ri, err := strconv.Atoi(reportInterval)
		if err == nil {
			cnf.ReportInterval = ri
		}
	}

	if pollInterval, exists := env.LookupEnv("POLL_INTERVAL"); exists {
		pi, err := strconv.Atoi(pollInterval)
		if err == nil {
			cnf.PollInterval = pi
		}
	}

	return cnf, nil
}
