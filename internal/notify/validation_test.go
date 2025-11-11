package notify

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "user@example.com", false},
		{"valid with subdomain", "user@mail.example.com", false},
		{"valid with plus", "user+tag@example.com", false},
		{"valid with dots", "first.last@example.com", false},
		{"empty email", "", true},
		{"no @ symbol", "userexample.com", true},
		{"no domain", "user@", true},
		{"no user", "@example.com", true},
		{"invalid format", "user @example.com", true},
		{"double @", "user@@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{"valid with plus", "+1234567890", false},
		{"valid without plus", "1234567890", false},
		{"valid long", "+123456789012345", false},
		{"empty phone", "", true},
		{"too short", "+1", true},
		{"too long", "+1234567890123456", true},
		{"invalid chars", "+123-456-7890", true},
		{"starts with zero", "+0123456789", true},
		{"letters", "+12345abcde", true},
		{"spaces", "+1 234 567 890", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePhone(tt.phone)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePhone() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRequest_Email(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name    string
		req     *SendRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid email request",
			req: &SendRequest{
				To:       "user@example.com",
				Channel:  ChannelEmail,
				Template: "welcome",
				Subject:  "Welcome",
				Data:     map[string]interface{}{"name": "John"},
			},
			wantErr: false,
		},
		{
			name: "missing recipient",
			req: &SendRequest{
				Channel:  ChannelEmail,
				Template: "welcome",
			},
			wantErr: true,
			errMsg:  "recipient",
		},
		{
			name: "invalid email",
			req: &SendRequest{
				To:       "invalid-email",
				Channel:  ChannelEmail,
				Template: "welcome",
			},
			wantErr: true,
			errMsg:  "invalid email",
		},
		{
			name: "missing channel",
			req: &SendRequest{
				To:       "user@example.com",
				Template: "welcome",
			},
			wantErr: true,
			errMsg:  "channel",
		},
		{
			name: "missing template",
			req: &SendRequest{
				To:      "user@example.com",
				Channel: ChannelEmail,
			},
			wantErr: true,
			errMsg:  "template",
		},
		{
			name: "invalid template name with path traversal",
			req: &SendRequest{
				To:       "user@example.com",
				Channel:  ChannelEmail,
				Template: "../../../etc/passwd",
			},
			wantErr: true,
			errMsg:  "invalid template name",
		},
		{
			name: "subject too long",
			req: &SendRequest{
				To:       "user@example.com",
				Channel:  ChannelEmail,
				Template: "welcome",
				Subject:  strings.Repeat("a", MaxSubjectLength+1),
			},
			wantErr: true,
			errMsg:  "subject too long",
		},
		{
			name: "data payload too large",
			req: &SendRequest{
				To:       "user@example.com",
				Channel:  ChannelEmail,
				Template: "welcome",
				Data: map[string]interface{}{
					"huge": strings.Repeat("x", MaxDataSize+1),
				},
			},
			wantErr: true,
			errMsg:  "payload too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateRequest() error = %v, should contain %q", err, tt.errMsg)
			}
		})
	}
}

func TestValidateRequest_Phone(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name    string
		req     *SendRequest
		wantErr bool
	}{
		{
			name: "valid whatsapp request",
			req: &SendRequest{
				To:       "+1234567890",
				Channel:  ChannelWhatsApp,
				Template: "welcome",
			},
			wantErr: false,
		},
		{
			name: "valid sms request",
			req: &SendRequest{
				To:       "+1234567890",
				Channel:  ChannelSMS,
				Template: "otp",
			},
			wantErr: false,
		},
		{
			name: "invalid phone for whatsapp",
			req: &SendRequest{
				To:       "invalid-phone",
				Channel:  ChannelWhatsApp,
				Template: "welcome",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEstimateDataSize(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		maxSize  int
		minSize  int
	}{
		{
			name: "simple data",
			data: map[string]interface{}{
				"name": "John",
				"age":  30,
			},
			minSize: 20,
			maxSize: 100,
		},
		{
			name: "nested data",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
					"age":  30,
				},
			},
			minSize: 30,
			maxSize: 150,
		},
		{
			name: "array data",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			minSize: 20,
			maxSize: 100,
		},
		{
			name:    "nil data",
			data:    nil,
			minSize: 0,
			maxSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := estimateDataSize(tt.data)
			if size < tt.minSize || size > tt.maxSize {
				t.Errorf("estimateDataSize() = %d, want between %d and %d", size, tt.minSize, tt.maxSize)
			}
		})
	}
}

func TestEstimateValueSize(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		min   int
		max   int
	}{
		{"string", "hello", 5, 10},
		{"int", 12345, 5, 25},
		{"bool", true, 4, 6},
		{"nil", nil, 4, 4},
		{"float", 3.14159, 5, 35},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := estimateValueSize(tt.value)
			if size < tt.min || size > tt.max {
				t.Errorf("estimateValueSize() = %d, want between %d and %d", size, tt.min, tt.max)
			}
		})
	}
}
