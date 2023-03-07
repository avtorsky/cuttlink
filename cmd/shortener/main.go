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

	fileStorage, _ := storage.NewFile(cfg.FileStoragePath)
	defer fileStorage.CloseFS()

	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var localStorage storage.Storager
	if db != nil && cfg.DatabaseDSN != "" {
		driver, err := postgres.WithInstance(db, &postgres.Config{})
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
	} else if db == nil && cfg.DatabaseDSN == "" && fileStorage != nil && cfg.FileStoragePath != "" {
		localStorage, _ = storage.NewFileStorage(fileStorage)
	} else {
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
