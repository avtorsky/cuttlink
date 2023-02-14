package main

import (
	"database/sql"
	"github.com/avtorsky/cuttlink/internal/config"
	"github.com/avtorsky/cuttlink/internal/server"
	"github.com/avtorsky/cuttlink/internal/storage"

	_ "github.com/jackc/pgx/v5/stdlib"
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

	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var localStorage *storage.StorageDB
	switch {
	case db == nil, cfg.DatabaseDSN == "":
		localStorage, _ = storage.NewKV(fileStorage)
	default:
		if err = storage.Migrate(db); err != nil {
			panic(err)
		}
		localStorage, _ = storage.NewDB(db)
	}

	localServer, err := server.New(
		localStorage,
		server.WithServerHost(cfg.ServerHost),
		server.WithServiceHost(cfg.ServiceHost),
	)
	if err != nil {
		panic(err)
	}

	localServer.ListenAndServe()
}
