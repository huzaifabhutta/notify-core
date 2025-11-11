package notify

import (
	"fmt"
	"net/mail"
	"regexp"
)

const (
	// RFC 5322 specifies max 998 characters per line
	MaxSubjectLength = 998
	// Limit data payload to 100KB to prevent DoS
	MaxDataSize = 1024 * 100
	// Maximum recipient length
	MaxRecipientLength = 254 // RFC 5321
)

var (
	// validTemplateNameRegex ensures template names are safe (no path traversal)
	validTemplateNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	// validPhoneRegex for basic phone number validation (E.164 format)
	validPhoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
)

// validateRequest validates the send request with comprehensive checks
func (s *Service) validateRequest(req *SendRequest) error {
	// Validate recipient
	if req.To == "" {
		return fmt.Errorf("recipient (to) is required")
	}

	if len(req.To) > MaxRecipientLength {
		return fmt.Errorf("recipient too long: maximum %d characters", MaxRecipientLength)
	}

	// Channel-specific recipient validation
	switch req.Channel {
	case ChannelEmail:
		if err := validateEmail(req.To); err != nil {
			return fmt.Errorf("invalid email address: %w", err)
		}
	case ChannelWhatsApp, ChannelSMS:
		if err := validatePhone(req.To); err != nil {
			return fmt.Errorf("invalid phone number: %w", err)
		}
	}

	// Validate channel
	if req.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	// Validate template name (prevent path traversal)
	if req.Template == "" {
		return fmt.Errorf("template is required")
	}

	if !validTemplateNameRegex.MatchString(req.Template) {
		return fmt.Errorf("invalid template name: must contain only letters, numbers, hyphens, and underscores")
	}

	// Validate subject length (for email)
	if len(req.Subject) > MaxSubjectLength {
		return fmt.Errorf("subject too long: maximum %d characters", MaxSubjectLength)
	}

	// Validate "from" field if provided
	if req.From != "" {
		switch req.Channel {
		case ChannelEmail:
			if err := validateEmail(req.From); err != nil {
				return fmt.Errorf("invalid from email address: %w", err)
			}
		case ChannelWhatsApp, ChannelSMS:
			if err := validatePhone(req.From); err != nil {
				return fmt.Errorf("invalid from phone number: %w", err)
			}
		}
	}

	// Validate data payload size (prevent DoS)
	if req.Data != nil {
		dataSize := estimateDataSize(req.Data)
		if dataSize > MaxDataSize {
			return fmt.Errorf("data payload too large: maximum %d bytes, got approximately %d bytes", MaxDataSize, dataSize)
		}
	}

	return nil
}

// validateEmail validates an email address format
func validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email address is empty")
	}

	// Use Go's built-in email parser for RFC 5322 compliance
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid format: %w", err)
	}

	// Additional check: ensure no display name (just the address)
	if addr.Address != email {
		// This allows "Name <email@example.com>" format
		// If you want to be stricter, uncomment:
		// return fmt.Errorf("email must not include display name")
	}

	return nil
}

// validatePhone validates a phone number (basic E.164 format check)
func validatePhone(phone string) error {
	if phone == "" {
		return fmt.Errorf("phone number is empty")
	}

	// Basic validation for E.164 format
	// +[country code][subscriber number]
	// E.164 allows 1-15 digits
	if !validPhoneRegex.MatchString(phone) {
		return fmt.Errorf("invalid format: must be in E.164 format (e.g., +1234567890)")
	}

	return nil
}

// estimateDataSize estimates the size of the data payload in bytes
func estimateDataSize(data map[string]interface{}) int {
	if data == nil {
		return 0
	}

	size := 0

	for key, value := range data {
		// Add key size
		size += len(key)

		// Add value size (rough estimate)
		size += estimateValueSize(value)

		// Add overhead for JSON structure (colons, commas, quotes, etc.)
		size += 10
	}

	return size
}

// estimateValueSize estimates the size of a single value
func estimateValueSize(value interface{}) int {
	if value == nil {
		return 4 // "null"
	}

	switch v := value.(type) {
	case string:
		return len(v) + 2 // +2 for quotes
	case int, int8, int16, int32, int64:
		return 20 // Max digits for int64
	case uint, uint8, uint16, uint32, uint64:
		return 20
	case float32, float64:
		return 30 // Max precision
	case bool:
		return 5 // "true" or "false"
	case map[string]interface{}:
		return estimateDataSize(v)
	case []interface{}:
		size := 2 // brackets
		for _, item := range v {
			size += estimateValueSize(item) + 2 // +2 for comma and space
		}
		return size
	default:
		// For unknown types, estimate based on string representation
		return len(fmt.Sprintf("%v", v))
	}
}
