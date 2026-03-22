package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"um-calendar-backend/internal/handlers"
	"um-calendar-backend/internal/repo"
	"um-calendar-backend/internal/scraper"
	"um-calendar-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

var calendarRepo *repo.CalendarRepo

func main() {
	_ = godotenv.Load()

	if err := setupSync(); err != nil {
		log.Printf("calendar sync setup skipped: %v", err)
		scraper.CalendarLinks = make(map[string]string)
		scraper.GetCalendarLinks()
	}

	router := gin.Default()
	handler := handlers.New(calendarRepo)
	router.GET("/health", handler.HealthCheck)
	router.GET("/data/names", handler.ServeCalendarNames)
	router.GET("/data/cal/:name", handler.ServeCalendarICSByName)
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func setupSync() error {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	calendarRepo = repo.NewCalendarRepo(db)
	syncService := services.NewCalendarSyncService(calendarRepo)

	if err := syncService.SyncCalendars(); err != nil {
		return err
	}

	syncService.StartHourly()
	log.Println("calendar sync initialized (immediate + hourly)")
	return nil
}
