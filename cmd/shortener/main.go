package main

import (
	"fmt"
	"github.com/avtorsky/cuttlink/internal/config"
	"github.com/avtorsky/cuttlink/internal/server"
	"github.com/avtorsky/cuttlink/internal/storage"

	"github.com/caarlos0/env/v6"
)

func main() {
	cfg := config.Env{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
	}
	fileStorage := storage.NewFileStorage(cfg.FileStoragePath)
	defer fileStorage.CloseFS()
	localStorage := storage.New(fileStorage)
	localServer := server.New(localStorage, cfg.ServerHost, cfg.ServiceHost)

	fmt.Println("Server is running at", cfg.ServerHost)
	localServer.Run()
}
