package main

import (
	"fmt"
	"github.com/avtorsky/cuttlink/internal/config"
	"github.com/avtorsky/cuttlink/internal/server"
	"github.com/avtorsky/cuttlink/internal/storage"
)

func main() {
	cfg, err := config.SetEnvOptionPriority()
	if err != nil {
		panic(err)
	}
	fileStorage, err := storage.NewFileStorage(cfg.FileStoragePath)
	if err != nil {
		panic(err)
	}
	defer fileStorage.CloseFS()
	localStorage, err := storage.New(fileStorage)
	if err != nil {
		panic(err)
	}
	localServer, err := server.New(
		localStorage,
		server.WithServerHost(cfg.ServerHost),
		server.WithServiceHost(cfg.ServiceHost),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("Server is running at", cfg.ServerHost)
	localServer.Run()
}
