package logging

import (
	"log/slog"
	"os"
	"strings"
)

func Configure() *slog.Logger {
	logger := New()
	slog.SetDefault(logger)
	return logger
}

func New() *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	handlerOptions := &slog.HandlerOptions{Level: level}

	format := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
	if format == "text" {
		return slog.New(slog.NewTextHandler(os.Stdout, handlerOptions))
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, handlerOptions))
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
