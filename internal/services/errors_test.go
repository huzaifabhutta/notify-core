package services_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/huzaifabhutta/notify-core/internal/services"
)

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		wantText string
		dontWant string
	}{
		{
			name:     "sanitize password",
			input:    errors.New("failed to connect: password=secretpass123"),
			wantText: "***REDACTED***",
			dontWant: "secretpass123",
		},
		{
			name:     "sanitize token",
			input:    errors.New("auth failed: token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"),
			wantText: "***REDACTED***",
			dontWant: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:     "sanitize api_key",
			input:    errors.New("invalid api_key=sk-1234567890abcdef"),
			wantText: "***REDACTED***",
			dontWant: "sk-1234567890abcdef",
		},
		{
			name:     "sanitize bearer token",
			input:    errors.New("unauthorized: bearer abcdef123456"),
			wantText: "***REDACTED***",
			dontWant: "abcdef123456",
		},
		{
			name:     "nil error returns nil",
			input:    nil,
			wantText: "",
			dontWant: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := services.SanitizeError(tt.input)

			if tt.input == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			resultStr := result.Error()

			if tt.wantText != "" && !strings.Contains(resultStr, tt.wantText) {
				t.Errorf("expected result to contain %q, got %q", tt.wantText, resultStr)
			}

			if tt.dontWant != "" && strings.Contains(resultStr, tt.dontWant) {
				t.Errorf("result should NOT contain sensitive data %q, but got %q", tt.dontWant, resultStr)
			}
		})
	}
}

func TestWrapErrorSafe(t *testing.T) {
	original := errors.New("database connection failed: password=mysecret")
	wrapped := services.WrapErrorSafe("failed to initialize", original)

	result := wrapped.Error()

	if !strings.Contains(result, "failed to initialize") {
		t.Errorf("expected context in error, got %q", result)
	}

	if strings.Contains(result, "mysecret") {
		t.Errorf("sensitive data should be redacted, got %q", result)
	}

	if !strings.Contains(result, "***REDACTED***") {
		t.Errorf("expected redaction marker, got %q", result)
	}
}
