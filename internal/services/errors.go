package services

import (
	"fmt"
	"strings"
)

// SanitizeError removes sensitive information from error messages
// to prevent credential exposure in logs and API responses
func SanitizeError(err error, sensitiveKeywords ...string) error {
	if err == nil {
		return nil
	}

	msg := err.Error()

	// Default sensitive keywords to redact
	defaultKeywords := []string{
		"password",
		"token",
		"secret",
		"api_key",
		"apikey",
		"api-key",
		"bearer",
		"authorization",
		"credential",
		"key",
	}

	// Combine default and custom keywords
	keywords := append(defaultKeywords, sensitiveKeywords...)

	// Redact sensitive information (case-insensitive)
	for _, keyword := range keywords {
		// Find keyword in various formats
		patterns := []string{
			keyword + "=",
			keyword + ":",
			keyword + " ",
			"\"" + keyword + "\"",
		}

		for _, pattern := range patterns {
			lowerMsg := strings.ToLower(msg)
			lowerPattern := strings.ToLower(pattern)

			if idx := strings.Index(lowerMsg, lowerPattern); idx != -1 {
				// Find the end of the value (space, quote, or end of string)
				start := idx + len(pattern)
				end := start
				for end < len(msg) && msg[end] != ' ' && msg[end] != '"' && msg[end] != '\n' {
					end++
				}

				// Replace sensitive value with redacted marker
				if end > start {
					msg = msg[:start] + "***REDACTED***" + msg[end:]
				}
			}
		}
	}

	return fmt.Errorf("%s", msg)
}

// WrapErrorSafe wraps an error with context while sanitizing sensitive data
func WrapErrorSafe(context string, err error) error {
	if err == nil {
		return nil
	}
	sanitized := SanitizeError(err)
	return fmt.Errorf("%s: %w", context, sanitized)
}
