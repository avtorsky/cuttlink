package config

type Env struct {
	ServerPort      int    `env:"SERVER_ADDRESS" envDefault:"8080"`
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"cuttlink.txt"`
}
