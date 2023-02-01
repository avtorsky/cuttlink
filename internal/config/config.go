package config

type Env struct {
	ServerHost      string `env:"SERVER_ADDRESS" envDefault:":8080"`
	ServiceHost     string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"kv_store.txt"`
}
