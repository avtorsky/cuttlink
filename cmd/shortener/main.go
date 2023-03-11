package main

import (
	"errors"
	"github.com/avtorsky/cuttlink/internal/config"
	"github.com/avtorsky/cuttlink/internal/server"
	"github.com/avtorsky/cuttlink/internal/storage"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func main() {
	cfg, err := config.SetEnvOptionPriority()
	if err != nil {
		panic(err)
	}

	fileStorage, _ := storage.NewFile(cfg.FileStoragePath)
	defer fileStorage.CloseFS()

	var localStorage storage.Storager
	switch {
	case cfg.DatabaseDSN != "":
		db, err := sqlx.Open("pgx", cfg.DatabaseDSN)
		if err != nil {
			log.Fatalf("unable to init sqlx: %v", err)
		}
		driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
		if err != nil {
			log.Fatalf("unable to init db driver: %v", err)
		}
		m, err := migrate.NewWithDatabaseInstance(cfg.MigrationsPath, "cldev", driver)
		if err != nil {
			log.Fatalf("unable to init db migrator: %v", err)
		}
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("unable to migrate: %v", err)
		}
		localStorage, _ = storage.NewDB(db)
		defer db.Close()

	case fileStorage != nil && cfg.FileStoragePath != "":
		localStorage, _ = storage.NewFileStorage(fileStorage)

	default:
		localStorage, _ = storage.NewInMemoryStorage()
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
