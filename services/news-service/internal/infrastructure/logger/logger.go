package logger

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/Conte777/NewsFlow/services/news-service/config"
)

// NewLogger creates a new logger from config
func NewLogger(cfg *config.LoggingConfig) zerolog.Logger {
	return New(cfg.Level)
}

// New creates a new logger with specified level
func New(level string) zerolog.Logger {
	logLevel := parseLogLevel(level)
	zerolog.SetGlobalLevel(logLevel)

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
		With().
		Timestamp().
		Caller().
		Logger()

	return logger
}

// parseLogLevel parses log level string to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}
