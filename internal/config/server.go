package config

import (
	"flag"
	"strconv"

	models "github.com/mersikovs/korob.git/internal/model"
)

func ServerParseConfig(fs *flag.FlagSet, args []string, env EnvSource) (*models.ServerConfig, error) {
	cnf := &models.ServerConfig{}
	key := ""
	fs.StringVar(&cnf.Address, "a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	fs.StringVar(&cnf.DatabaseDSN, "d", "", "строка подключения к БД (DSN)")
	fs.StringVar(&key, "k", "", "ключ подписи данных")
	fs.IntVar(&cnf.StoreInterval, "i", 300, "интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск")
	fs.StringVar(&cnf.FileStoragePath, "f", "store.back", "путь до файла, куда сохраняются текущие значения")
	fs.BoolVar(&cnf.Restore, "r", false, "следует ли загружать ранее сохранённые значения")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if addr, exists := env.LookupEnv("ADDRESS"); exists && addr != "" {
		cnf.Address = addr
	}

	if dsn, exists := env.LookupEnv("DATABASE_DSN"); exists && dsn != "" {
		cnf.DatabaseDSN = dsn
	}

	if envKey, exists := env.LookupEnv("KEY"); exists && envKey != "" {
		key = envKey
	}

	if len(key) > 0 {
		cnf.Key = []byte(key)
	}

	if si, exists := env.LookupEnv("STORE_INTERVAL"); exists {
		si, err := strconv.Atoi(si)
		if err == nil {
			cnf.StoreInterval = si
		}
	}

	if fsp, exists := env.LookupEnv("FILE_STORAGE_PATH"); exists {
		cnf.FileStoragePath = fsp
	}

	if r, exists := env.LookupEnv("RESTORE"); exists {
		if r == "true" {
			cnf.Restore = true
		}

		if r == "false" {
			cnf.Restore = false
		}
	}

	return cnf, nil
}
