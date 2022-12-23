package main

import (
	"fmt"
	"github.com/avtorsky/cuttlink/internal/server"
	"github.com/avtorsky/cuttlink/internal/services"
	"github.com/avtorsky/cuttlink/internal/storage"
)

func main() {
	localStorage := storage.New()
	localProxyService := services.New(localStorage)
	localServer := server.New(localProxyService, "http://localhost:8080", 8080)

	fmt.Println("Server is running at localhost, port 8080.")
	localServer.Run()
}
