package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Set test environment variables
	os.Setenv("PORT", "9090")
	os.Setenv("SMTP_HOST", "smtp.test.com")
	os.Setenv("SMTP_PORT", "587")
	os.Setenv("SMTP_USER", "test@test.com")
	os.Setenv("SMTP_PASS", "testpass")
	os.Setenv("WA_TOKEN", "test-token")

	defer func() {
		// Clean up
		os.Unsetenv("PORT")
		os.Unsetenv("SMTP_HOST")
		os.Unsetenv("SMTP_PORT")
		os.Unsetenv("SMTP_USER")
		os.Unsetenv("SMTP_PASS")
		os.Unsetenv("WA_TOKEN")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test server config
	if cfg.Server.Port != "9090" {
		t.Errorf("Expected port 9090, got %s", cfg.Server.Port)
	}

	// Test SMTP config
	if cfg.SMTP.Host != "smtp.test.com" {
		t.Errorf("Expected SMTP host smtp.test.com, got %s", cfg.SMTP.Host)
	}

	if cfg.SMTP.Port != 587 {
		t.Errorf("Expected SMTP port 587, got %d", cfg.SMTP.Port)
	}

	if cfg.SMTP.User != "test@test.com" {
		t.Errorf("Expected SMTP user test@test.com, got %s", cfg.SMTP.User)
	}

	// Test WhatsApp config
	if cfg.WhatsApp.Token != "test-token" {
		t.Errorf("Expected WA token test-token, got %s", cfg.WhatsApp.Token)
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	// Test with set value
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	value := getEnv("TEST_VAR", "default")
	if value != "test-value" {
		t.Errorf("Expected test-value, got %s", value)
	}

	// Test with unset value (should use default)
	value = getEnv("NONEXISTENT_VAR", "default")
	if value != "default" {
		t.Errorf("Expected default, got %s", value)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				SMTP: SMTPConfig{
					Host:     "smtp.test.com",
					User:     "test@test.com",
					Password: "password",
				},
			},
			wantErr: false,
		},
		{
			name: "missing SMTP host",
			config: &Config{
				SMTP: SMTPConfig{
					Host:     "",
					User:     "test@test.com",
					Password: "password",
				},
			},
			wantErr: true,
		},
		{
			name: "missing SMTP user",
			config: &Config{
				SMTP: SMTPConfig{
					Host:     "smtp.test.com",
					User:     "",
					Password: "password",
				},
			},
			wantErr: true,
		},
		{
			name: "missing SMTP password",
			config: &Config{
				SMTP: SMTPConfig{
					Host:     "smtp.test.com",
					User:     "test@test.com",
					Password: "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
