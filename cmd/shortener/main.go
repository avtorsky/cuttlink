package main

import (
	"fmt"
	"github.com/avtorsky/cuttlink/internal/proxy"
	"github.com/avtorsky/cuttlink/internal/server"
	"github.com/avtorsky/cuttlink/internal/storage"
)

func main() {
	localStorage := storage.New()
	localProxy := proxy.New(localStorage)
	localServer := server.New(localProxy, "http://localhost:8080", 8080)

	fmt.Println("Server is running at localhost, port 8080.")
	localServer.Run()
}
