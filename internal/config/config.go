package config

type Env struct {
	ServerHost      string `env:"SERVER_ADDRESS"`
	ServiceHost     string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}
