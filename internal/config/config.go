package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type Env struct {
	ServerHost      string `env:"SERVER_ADDRESS" envDefault:":8080"`
	ServiceHost     string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"kv_store.txt"`
	DatabaseDSN     string `eng:"DATABASE_DSN" envDefault:"postgres://cluser:clpassword@localhost:5432/cldev"`
}

func SetEnvOptionPriority() (Env, error) {
	var config Env
	if err := env.Parse(&config); err != nil {
		return config, err
	}

	serverHost := flag.String("a", config.ServerHost, "define server address")
	serviceHost := flag.String("b", config.ServiceHost, "define base URL")
	fileStoragePath := flag.String("f", config.FileStoragePath, "define file storage path")
	databaseDSN := flag.String("d", config.DatabaseDSN, "define DSN connection")
	flag.Parse()

	config.ServerHost = *serverHost
	config.ServiceHost = *serviceHost
	config.FileStoragePath = *fileStoragePath
	config.DatabaseDSN = *databaseDSN
	return config, nil
}
