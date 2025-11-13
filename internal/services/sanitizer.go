package services

import (
	"html"
	"regexp"
	"strings"
)

// Sanitizer provides input sanitization to prevent XSS and injection attacks
type Sanitizer struct {
	// Patterns to detect and remove potentially malicious content
	scriptPattern  *regexp.Regexp
	iframePattern  *regexp.Regexp
	objectPattern  *regexp.Regexp
	embedPattern   *regexp.Regexp
	onEventPattern *regexp.Regexp
	dataPattern    *regexp.Regexp
}

// NewSanitizer creates a new input sanitizer
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		scriptPattern:  regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`),
		iframePattern:  regexp.MustCompile(`(?i)<iframe[^>]*>[\s\S]*?</iframe>`),
		objectPattern:  regexp.MustCompile(`(?i)<object[^>]*>[\s\S]*?</object>`),
		embedPattern:   regexp.MustCompile(`(?i)<embed[^>]*>`),
		onEventPattern: regexp.MustCompile(`(?i)\son\w+\s*=`),
		dataPattern:    regexp.MustCompile(`(?i)data:\s*text/html`),
	}
}

// SanitizeHTML removes potentially dangerous HTML tags and attributes
// Used for email body content that may contain HTML
func (s *Sanitizer) SanitizeHTML(input string) string {
	if input == "" {
		return input
	}

	// Remove script tags
	input = s.scriptPattern.ReplaceAllString(input, "")

	// Remove iframe tags
	input = s.iframePattern.ReplaceAllString(input, "")

	// Remove object tags
	input = s.objectPattern.ReplaceAllString(input, "")

	// Remove embed tags
	input = s.embedPattern.ReplaceAllString(input, "")

	// Remove event handlers (onclick, onload, etc.)
	input = s.onEventPattern.ReplaceAllString(input, "")

	// Remove data URLs that could contain HTML
	input = s.dataPattern.ReplaceAllString(input, "")

	return input
}

// SanitizeTemplateData sanitizes template data map to prevent XSS
// Escapes HTML in all string values
func (s *Sanitizer) SanitizeTemplateData(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	sanitized := make(map[string]interface{})
	for key, value := range data {
		sanitized[key] = s.sanitizeValue(value)
	}

	return sanitized
}

// sanitizeValue sanitizes a single value recursively
func (s *Sanitizer) sanitizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// Escape HTML entities
		return html.EscapeString(v)
	case map[string]interface{}:
		// Recursively sanitize nested maps
		return s.SanitizeTemplateData(v)
	case []interface{}:
		// Sanitize array elements
		sanitized := make([]interface{}, len(v))
		for i, item := range v {
			sanitized[i] = s.sanitizeValue(item)
		}
		return sanitized
	default:
		// Numbers, bools, etc. are safe
		return v
	}
}

// SanitizeEmail validates and sanitizes email addresses
func (s *Sanitizer) SanitizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)

	// Remove any HTML tags
	email = s.stripHTMLTags(email)

	return email
}

// SanitizeSubject sanitizes email subject to prevent header injection
func (s *Sanitizer) SanitizeSubject(subject string) string {
	if subject == "" {
		return subject
	}

	// Remove newlines and carriage returns (prevent email header injection)
	subject = strings.ReplaceAll(subject, "\r", "")
	subject = strings.ReplaceAll(subject, "\n", "")

	// Trim whitespace
	subject = strings.TrimSpace(subject)

	return subject
}

// SanitizePhoneNumber sanitizes phone numbers for SMS
func (s *Sanitizer) SanitizePhoneNumber(phone string) string {
	phone = strings.TrimSpace(phone)

	// Remove all characters except digits, +, -, (, ), and spaces
	var sanitized strings.Builder
	for _, char := range phone {
		if (char >= '0' && char <= '9') || char == '+' || char == '-' || char == '(' || char == ')' || char == ' ' {
			sanitized.WriteRune(char)
		}
	}

	return sanitized.String()
}

// stripHTMLTags removes all HTML tags from a string
func (s *Sanitizer) stripHTMLTags(input string) string {
	tagPattern := regexp.MustCompile(`<[^>]*>`)
	return tagPattern.ReplaceAllString(input, "")
}

// ValidateNoScriptInjection checks if input contains potential script injection
// Returns true if input is safe, false if it contains suspicious content
func (s *Sanitizer) ValidateNoScriptInjection(input string) bool {
	if input == "" {
		return true
	}

	lower := strings.ToLower(input)

	// Check for script tags
	if strings.Contains(lower, "<script") {
		return false
	}

	// Check for javascript: protocol
	if strings.Contains(lower, "javascript:") {
		return false
	}

	// Check for data: URLs with HTML
	if strings.Contains(lower, "data:text/html") {
		return false
	}

	// Check for common XSS patterns
	xssPatterns := []string{
		"<iframe",
		"<object",
		"<embed",
		"onerror=",
		"onload=",
		"onclick=",
		"eval(",
	}

	for _, pattern := range xssPatterns {
		if strings.Contains(lower, pattern) {
			return false
		}
	}

	return true
}
