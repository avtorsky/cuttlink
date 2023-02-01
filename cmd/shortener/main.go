package main

import (
	"flag"
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

	serverHost := flag.String("a", cfg.ServerHost, "define server address")
	serviceHost := flag.String("b", cfg.ServiceHost, "define base URL")
	fileStoragePath := flag.String("f", cfg.FileStoragePath, "define file storage path")
	flag.Parse()

	fileStorage := storage.NewFileStorage(*fileStoragePath)
	defer fileStorage.CloseFS()
	localStorage := storage.New(fileStorage)
	localServer := server.New(localStorage, *serverHost, *serviceHost)

	fmt.Println("Server is running at", *serverHost)
	localServer.Run()
}
