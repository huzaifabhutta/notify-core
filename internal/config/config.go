package config

import (
	"fmt"
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
	URL string
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
			URL: getEnv("DB_URL", ""),
		},
		Templates: TemplatesConfig{
			Dir: getEnv("TEMPLATES_DIR", "./templates"),
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

// Validate checks if required configuration is present
func (c *Config) Validate() error {
	if c.SMTP.Host == "" || c.SMTP.User == "" || c.SMTP.Password == "" {
		return fmt.Errorf("SMTP configuration is incomplete")
	}
	return nil
}
