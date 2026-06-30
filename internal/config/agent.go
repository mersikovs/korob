package config

import (
	"flag"
)

type Config struct {
	Address        string
	ReportInterval int
	PollInterval   int
}

func Parse() (*Config, error) {
	cnf := &Config{}

	flag.StringVar(&cnf.Address, "a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	flag.IntVar(&cnf.ReportInterval, "r", 10, "частота отправки метрик на сервер")
	flag.IntVar(&cnf.PollInterval, "p", 2, "частота опроса метрик")

	flag.Parse()

	return cnf, nil
}
