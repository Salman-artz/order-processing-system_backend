// Package logger provides a shared structured logger using zerolog.
// All services import this package to get consistent log formatting
// with correlation IDs, service names, and log levels.
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config holds logger configuration.
type Config struct {
	// ServiceName is injected into every log line as "service" field.
	ServiceName string
	// Level: trace, debug, info, warn, error, fatal, panic
	Level string
	// Pretty enables human-readable console output (dev only).
	// In production set to false for JSON output.
	Pretty bool
}

// Init initialises the global zerolog logger.
// Call this once at application startup in main().
func Init(cfg Config) zerolog.Logger {
	// Parse log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Timestamp format
	zerolog.TimeFieldFormat = time.RFC3339

	var baseLogger zerolog.Logger
	if cfg.Pretty {
		// Dev: coloured, human-readable output
		baseLogger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		})
	} else {
		// Production: JSON output
		baseLogger = zerolog.New(os.Stdout)
	}

	baseLogger = baseLogger.
		With().
		Timestamp().
		Str("service", cfg.ServiceName).
		Logger()

	// Replace the global logger
	log.Logger = baseLogger

	return baseLogger
}

// WithOrderID returns a child logger with the order_id field set.
// Use this for all log lines within an order saga context.
func WithOrderID(logger zerolog.Logger, orderID string) zerolog.Logger {
	return logger.With().Str("order_id", orderID).Logger()
}

// WithCorrelationID returns a child logger with a correlation_id field.
func WithCorrelationID(logger zerolog.Logger, correlationID string) zerolog.Logger {
	return logger.With().Str("correlation_id", correlationID).Logger()
}
