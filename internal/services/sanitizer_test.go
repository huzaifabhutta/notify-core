package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizer_SanitizeHTML(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove script tags",
			input:    "Hello <script>alert('xss')</script> World",
			expected: "Hello  World",
		},
		{
			name:     "Remove iframe tags",
			input:    "Content <iframe src='evil.com'></iframe> more",
			expected: "Content  more",
		},
		{
			name:     "Remove event handlers",
			input:    "<div onclick='alert()'>Click me</div>",
			expected: "<div'alert()'>Click me</div>",
		},
		{
			name:     "Remove data URLs and scripts",
			input:    "Test data:text/html,<script>alert('xss')</script>",
			expected: "Test ,",
		},
		{
			name:     "Safe HTML unchanged",
			input:    "<p>Hello <b>World</b></p>",
			expected: "<p>Hello <b>World</b></p>",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeHTML(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizer_SanitizeTemplateData(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "Escape HTML in strings",
			input: map[string]interface{}{
				"name": "<script>alert('xss')</script>John",
			},
			expected: map[string]interface{}{
				"name": "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;John",
			},
		},
		{
			name: "Nested map sanitization",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "<b>Bold</b>",
				},
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "&lt;b&gt;Bold&lt;/b&gt;",
				},
			},
		},
		{
			name: "Array sanitization",
			input: map[string]interface{}{
				"items": []interface{}{"<script>", "safe", "<iframe>"},
			},
			expected: map[string]interface{}{
				"items": []interface{}{"&lt;script&gt;", "safe", "&lt;iframe&gt;"},
			},
		},
		{
			name: "Non-string values unchanged",
			input: map[string]interface{}{
				"count":  42,
				"active": true,
			},
			expected: map[string]interface{}{
				"count":  42,
				"active": true,
			},
		},
		{
			name:     "Nil map",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeTemplateData(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizer_SanitizeEmail(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Lowercase and trim",
			input:    "  User@Example.COM  ",
			expected: "user@example.com",
		},
		{
			name:     "Remove HTML tags",
			input:    "user<script>@example.com",
			expected: "user@example.com",
		},
		{
			name:     "Normal email",
			input:    "user@example.com",
			expected: "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeEmail(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizer_SanitizeSubject(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove newlines",
			input:    "Subject\nwith\nnewlines",
			expected: "Subjectwithnewlines",
		},
		{
			name:     "Remove carriage returns",
			input:    "Subject\r\nInjection",
			expected: "SubjectInjection",
		},
		{
			name:     "Trim whitespace",
			input:    "  Subject  ",
			expected: "Subject",
		},
		{
			name:     "Normal subject",
			input:    "Hello World",
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeSubject(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizer_SanitizePhoneNumber(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove special characters",
			input:    "+1-234-567-8900",
			expected: "+1-234-567-8900",
		},
		{
			name:     "Remove invalid characters",
			input:    "+1<script>2345678900",
			expected: "+12345678900",
		},
		{
			name:     "With parentheses",
			input:    "(123) 456-7890",
			expected: "(123) 456-7890",
		},
		{
			name:     "Trim whitespace",
			input:    "  1234567890  ",
			expected: "1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizePhoneNumber(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizer_ValidateNoScriptInjection(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Safe input",
			input:    "Hello World",
			expected: true,
		},
		{
			name:     "Script tag detected",
			input:    "<script>alert('xss')</script>",
			expected: false,
		},
		{
			name:     "JavaScript protocol",
			input:    "javascript:alert('xss')",
			expected: false,
		},
		{
			name:     "Iframe detected",
			input:    "<iframe src='evil.com'></iframe>",
			expected: false,
		},
		{
			name:     "Event handler detected",
			input:    "onclick=alert('xss')",
			expected: false,
		},
		{
			name:     "Eval detected",
			input:    "eval(malicious)",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.ValidateNoScriptInjection(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
