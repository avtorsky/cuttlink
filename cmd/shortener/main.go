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
	localStorage := storage.New()
	localServer := server.New(localStorage, cfg.BaseURL, cfg.ServerPort)

	fmt.Printf("Server is running at %s, port %v\n", cfg.BaseURL, cfg.ServerPort)
	localServer.Run()
}
