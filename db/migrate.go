package main

import (
	"log"
	"os"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"

	"github.com/golang-migrate/migrate/v4"
)

func main() {
	godotenv.Load()
	m, err := migrate.New(
		"file://db/migrations",
		os.Getenv("DATABASE_URL"),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := m.Up(); err != nil && err.Error() != "no change" {
		log.Fatal(err)
	}

	log.Println("migrations applied")
}
