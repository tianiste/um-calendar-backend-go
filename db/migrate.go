package main

import (
	"log/slog"
	"os"
	"um-calendar-backend/internal/logging"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"

	"github.com/golang-migrate/migrate/v4"
)

func main() {
	godotenv.Load()
	logging.Configure()
	m, err := migrate.New(
		"file://db/migrations",
		os.Getenv("DATABASE_URL"),
	)
	if err != nil {
		slog.Error("failed to create migrator", "error", err)
		os.Exit(1)
	}

	if err := m.Up(); err != nil && err.Error() != "no change" {
		slog.Error("failed to apply migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations applied")
}
