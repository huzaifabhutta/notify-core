package config

import (
	"fmt"
	"net/mail"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server    ServerConfig
	SMTP      SMTPConfig
	WhatsApp  WhatsAppConfig
	SMS       SMSConfig
	Database  DatabaseConfig
	Templates TemplatesConfig
	Security  SecurityConfig
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	EncryptionKey string
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port string
	Env  string
}

// SMTPConfig holds SMTP/Email configuration
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// WhatsAppConfig holds WhatsApp Business API configuration
type WhatsAppConfig struct {
	Token       string
	PhoneID     string
	WebhookURL  string
	VerifyToken string
}

// SMSConfig holds SMS provider configuration
type SMSConfig struct {
	Provider string
	APIKey   string
	From     string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int
	MaxIdle  int
}

// TemplatesConfig holds template system configuration
type TemplatesConfig struct {
	Dir string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file (optional - don't error if not found)
	_ = godotenv.Load()

	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	dbMaxConns, err := strconv.Atoi(getEnv("DB_MAX_CONNS", "25"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_CONNS: %w", err)
	}

	dbMaxIdle, err := strconv.Atoi(getEnv("DB_MAX_IDLE", "5"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_IDLE: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Env:  getEnv("ENV", "development"),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     smtpPort,
			User:     getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASS", ""),
			From:     getEnv("SMTP_FROM", getEnv("SMTP_USER", "")),
		},
		WhatsApp: WhatsAppConfig{
			Token:       getEnv("WA_TOKEN", ""),
			PhoneID:     getEnv("WA_PHONE_ID", ""),
			WebhookURL:  getEnv("WA_WEBHOOK_URL", ""),
			VerifyToken: getEnv("WA_VERIFY_TOKEN", ""),
		},
		SMS: SMSConfig{
			Provider: getEnv("SMS_PROVIDER", "twilio"),
			APIKey:   getEnv("SMS_API_KEY", ""),
			From:     getEnv("SMS_FROM", ""),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "notify"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
			MaxConns: dbMaxConns,
			MaxIdle:  dbMaxIdle,
		},
		Templates: TemplatesConfig{
			Dir: getEnv("TEMPLATES_DIR", "./templates"),
		},
		Security: SecurityConfig{
			EncryptionKey: getEnv("ENCRYPTION_KEY", ""),
		},
	}

	return cfg, nil
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Validate checks if required configuration is present and valid
func (c *Config) Validate() error {
	// SMTP validation
	if c.SMTP.Host == "" {
		return fmt.Errorf("SMTP_HOST is required")
	}
	if c.SMTP.User == "" {
		return fmt.Errorf("SMTP_USER is required")
	}
	if c.SMTP.Password == "" {
		return fmt.Errorf("SMTP_PASS is required")
	}
	if c.SMTP.Port < 1 || c.SMTP.Port > 65535 {
		return fmt.Errorf("invalid SMTP_PORT: must be between 1-65535, got %d", c.SMTP.Port)
	}

	// Validate SMTP_FROM email format
	if c.SMTP.From != "" {
		if _, err := mail.ParseAddress(c.SMTP.From); err != nil {
			return fmt.Errorf("invalid SMTP_FROM email address: %w", err)
		}
	}

	// Validate SMTP_USER email format (if it looks like an email)
	if c.SMTP.User != "" {
		if _, err := mail.ParseAddress(c.SMTP.User); err != nil {
			// SMTP_USER might not be an email (could be username), so just warn
			// Don't fail validation
		}
	}

	// Templates directory validation
	if c.Templates.Dir != "" {
		if _, err := os.Stat(c.Templates.Dir); os.IsNotExist(err) {
			return fmt.Errorf("templates directory does not exist: %s", c.Templates.Dir)
		}
	}

	// Server port validation
	if port, err := strconv.Atoi(c.Server.Port); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid PORT: must be between 1-65535, got %s", c.Server.Port)
	}

	// Database validation (Week 3+)
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid DB_PORT: must be between 1-65535, got %d", c.Database.Port)
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if c.Database.MaxConns < 1 {
		return fmt.Errorf("DB_MAX_CONNS must be at least 1, got %d", c.Database.MaxConns)
	}
	if c.Database.MaxIdle < 0 || c.Database.MaxIdle > c.Database.MaxConns {
		return fmt.Errorf("DB_MAX_IDLE must be between 0 and DB_MAX_CONNS (%d), got %d", c.Database.MaxConns, c.Database.MaxIdle)
	}

	// Security validation
	if c.Security.EncryptionKey == "" {
		return fmt.Errorf("ENCRYPTION_KEY is required for encrypting sensitive data")
	}
	if len(c.Security.EncryptionKey) < 32 {
		return fmt.Errorf("ENCRYPTION_KEY must be at least 32 characters for AES-256 encryption, got %d", len(c.Security.EncryptionKey))
	}

	return nil
}
