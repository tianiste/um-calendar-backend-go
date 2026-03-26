package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"um-calendar-backend/internal/cache"
	"um-calendar-backend/internal/handlers"
	"um-calendar-backend/internal/logging"
	"um-calendar-backend/internal/middleware"
	"um-calendar-backend/internal/repo"
	"um-calendar-backend/internal/scraper"
	"um-calendar-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"

	_ "github.com/lib/pq"
)

var calendarRepo *repo.CalendarRepo

func main() {
	_ = godotenv.Load()
	logging.Configure()

	if err := setupSync(); err != nil {
		slog.Warn("calendar sync setup skipped, using in-memory scraper fallback", "error", err)
		scraper.CalendarLinks = make(map[string]string)
		scraper.GetCalendarLinks()
	}

	router := gin.Default()
	router.Use(middleware.NewIPRateLimiter(rateLimit(), rateBurst(), time.Minute, 10*time.Minute))
	inMemoryCache := cache.NewInMemoryCache(envDuration("IN_MEMORY_CACHE_TTL", 5*time.Minute))
	handler := handlers.New(calendarRepo, inMemoryCache)
	router.GET("/health", handler.HealthCheck)
	router.GET("/data/names", handler.ServeCalendarNames)
	router.GET("/data/cal/:name", handler.ServeCalendarICSByName)
	if err := router.Run(":8080"); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}

func setupSync() error {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}

	if err := runMigrations(databaseURL); err != nil {
		return err
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return err
	}

	configureDBPool(db)

	if err := db.Ping(); err != nil {
		return err
	}

	calendarRepo = repo.NewCalendarRepo(db)
	syncService := services.NewCalendarSyncService(calendarRepo)

	if err := syncService.SyncCalendars(); err != nil {
		return err
	}

	syncService.StartHourly()
	slog.Info("calendar sync initialized", "mode", "immediate+hourly")
	return nil
}

func runMigrations(databaseURL string) error {
	migrationsPath := strings.TrimSpace(os.Getenv("MIGRATIONS_PATH"))
	if migrationsPath == "" {
		migrationsPath = "db/migrations"
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("resolve migration path: %w", err)
	}

	sourceURL := "file://" + filepath.ToSlash(absPath)
	migrator, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer func() {
		_, _ = migrator.Close()
	}()

	err = migrator.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	slog.Info("migrations ready", "source", sourceURL)
	return nil
}

func configureDBPool(db *sql.DB) {
	db.SetMaxOpenConns(envInt("DB_MAX_OPEN_CONNS", 25))
	db.SetMaxIdleConns(envInt("DB_MAX_IDLE_CONNS", 10))
	db.SetConnMaxLifetime(envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute))
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func rateLimit() rate.Limit {
	value := strings.TrimSpace(os.Getenv("RATE_LIMIT_RPS"))
	if value == "" {
		return rate.Limit(5)
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed <= 0 {
		return rate.Limit(5)
	}

	return rate.Limit(parsed)
}

func rateBurst() int {
	value := strings.TrimSpace(os.Getenv("RATE_LIMIT_BURST"))
	if value == "" {
		return 20
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 20
	}

	return parsed
}
