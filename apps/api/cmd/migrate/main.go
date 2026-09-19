package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/marcos-vinicius14/radar-enfermagem-rs/apps/api/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: go run ./cmd/migrate [up|down|version]")
	}

	command := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	dsn := fmt.Sprintf("pgx5://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode)

	migrationsPath := "file://migrations"
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		if _, err := os.Stat("apps/api/migrations"); err == nil {
			migrationsPath = "file://apps/api/migrations"
		}
	}

	m, err := migrate.New(migrationsPath, dsn)
	if err != nil {
		log.Fatalf("failed to initialize migrate: %v", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			log.Printf("falha ao fechar fonte de migrações: %v", sourceErr)
		}
		if dbErr != nil {
			log.Printf("falha ao fechar conexão de migrações: %v", dbErr)
		}
	}()

	switch command {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("failed to apply migrations: %v", err)
		}
		log.Println("Migrations applied successfully!")
	case "down":
		if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("failed to rollback migration: %v", err)
		}
		log.Println("Migration rolled back successfully!")
	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				log.Println("Current migration version: 0 (no migrations applied)")
				return
			}
			log.Fatalf("failed to get migration version: %v", err)
		}
		log.Printf("Current migration version: %d (dirty: %v)\n", version, dirty)
	default:
		log.Fatalf("unknown command: %s (expected up, down, or version)", command)
	}
}
