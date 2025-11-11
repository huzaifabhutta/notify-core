package email

import (
	"context"
	"testing"

	"github.com/huzaifabhutta/notify-core/internal/config"
)

func TestAdapter_Name(t *testing.T) {
	adapter := NewAdapter(&config.SMTPConfig{}, &config.TemplatesConfig{})

	if adapter.Name() != "email" {
		t.Errorf("Expected adapter name 'email', got %s", adapter.Name())
	}
}

func TestAdapter_RenderSimpleTemplate(t *testing.T) {
	adapter := NewAdapter(&config.SMTPConfig{}, &config.TemplatesConfig{})

	data := map[string]interface{}{
		"Name":    "John Doe",
		"Message": "Welcome!",
	}

	html, err := adapter.renderSimpleTemplate(data)
	if err != nil {
		t.Fatalf("Failed to render simple template: %v", err)
	}

	if html == "" {
		t.Error("Expected non-empty HTML output")
	}

	// Check if data is included in output
	if !contains(html, "John Doe") {
		t.Error("Expected HTML to contain 'John Doe'")
	}

	if !contains(html, "Welcome!") {
		t.Error("Expected HTML to contain 'Welcome!'")
	}
}

func TestAdapter_ComposeEmail(t *testing.T) {
	adapter := NewAdapter(&config.SMTPConfig{}, &config.TemplatesConfig{})

	from := "sender@example.com"
	to := "recipient@example.com"
	subject := "Test Subject"
	body := "<html><body>Test Body</body></html>"

	msg := adapter.composeEmail(from, to, subject, body)

	// Check headers
	if !contains(msg, "From: "+from) {
		t.Error("Expected From header in message")
	}

	if !contains(msg, "To: "+to) {
		t.Error("Expected To header in message")
	}

	if !contains(msg, "Subject: "+subject) {
		t.Error("Expected Subject header in message")
	}

	if !contains(msg, "MIME-Version: 1.0") {
		t.Error("Expected MIME-Version header in message")
	}

	if !contains(msg, "Content-Type: text/html") {
		t.Error("Expected Content-Type header in message")
	}

	if !contains(msg, body) {
		t.Error("Expected body in message")
	}
}

func TestAdapter_Send_InvalidRequest(t *testing.T) {
	adapter := NewAdapter(
		&config.SMTPConfig{
			Host:     "smtp.test.com",
			Port:     587,
			User:     "test@test.com",
			Password: "password",
			From:     "noreply@test.com",
		},
		&config.TemplatesConfig{
			Dir: "./templates",
		},
	)

	ctx := context.Background()

	// Test with invalid request type
	err := adapter.Send(ctx, "invalid")
	if err == nil {
		t.Error("Expected error for invalid request type")
	}
}

func TestAdapter_Send_ValidRequest(t *testing.T) {
	// Note: This test won't actually send an email
	// It will fail at the SMTP connection stage, which is expected
	adapter := NewAdapter(
		&config.SMTPConfig{
			Host:     "smtp.test.com",
			Port:     587,
			User:     "test@test.com",
			Password: "password",
			From:     "noreply@test.com",
		},
		&config.TemplatesConfig{
			Dir: "../../templates",
		},
	)

	ctx := context.Background()

	req := &SendRequest{
		To:       "recipient@test.com",
		Template: "nonexistent",
		Subject:  "Test Subject",
		Data: map[string]interface{}{
			"Name": "Test User",
		},
	}

	// This will fail at template render or SMTP send, which is fine for this test
	// We're just testing that the function runs without panicking
	_ = adapter.Send(ctx, req)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
