package main

import (
	"database/sql"
	"errors"
	"github.com/avtorsky/cuttlink/internal/config"
	"github.com/avtorsky/cuttlink/internal/server"
	"github.com/avtorsky/cuttlink/internal/storage"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	db, err := sql.Open("postgres", cfg.DatabaseDSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var localStorage *storage.StorageDB
	switch {
	case db == nil, cfg.DatabaseDSN == "":
		localStorage, _ = storage.NewKV(fileStorage)
	default:
		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			log.Fatalf("unable to init db driver: %v", err)
		}
		m, err := migrate.NewWithDatabaseInstance("file://./migrations", "cldev", driver)
		if err != nil {
			log.Fatalf("unable to init db migrator: %v", err)
		}
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("unable to migrate: %v", err)
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
