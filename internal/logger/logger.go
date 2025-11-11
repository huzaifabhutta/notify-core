package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ContextKey is the key for logger context
type ContextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
	// TenantIDKey is the context key for tenant ID
	TenantIDKey ContextKey = "tenant_id"
)

var (
	// Logger is the global logger instance
	Logger zerolog.Logger
)

// Config holds logger configuration
type Config struct {
	Level      string // debug, info, warn, error
	JSONFormat bool   // true for JSON, false for console
}

// Init initializes the global logger
func Init(cfg Config) {
	// Set log level
	level := zerolog.InfoLevel
	switch cfg.Level {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	}
	zerolog.SetGlobalLevel(level)

	// Set output format
	var output io.Writer = os.Stdout
	if !cfg.JSONFormat {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	// Create logger with timestamp
	Logger = zerolog.New(output).With().Timestamp().Logger()
	log.Logger = Logger
}

// FromContext extracts logger with context fields
func FromContext(ctx context.Context) *zerolog.Logger {
	logger := Logger

	// Add request ID if present
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		l := logger.With().Str("request_id", requestID).Logger()
		logger = l
	}

	// Add tenant ID if present
	if tenantID, ok := ctx.Value(TenantIDKey).(string); ok {
		l := logger.With().Str("tenant_id", tenantID).Logger()
		logger = l
	}

	return &logger
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithTenantID adds tenant ID to context
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantIDKey, tenantID)
}

// MaskEmail masks email address for logging (only show domain)
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}
	// Show only domain for privacy
	// user@example.com -> ***@example.com
	for i := 0; i < len(email); i++ {
		if email[i] == '@' {
			return "***" + email[i:]
		}
	}
	return "***"
}

// MaskPhone masks phone number for logging (only show last 4 digits)
func MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return "****" + phone[len(phone)-4:]
}
