package notify

import (
	"context"
	"testing"

	"github.com/huzaifabhutta/notify-core/internal/config"
)

// MockAdapter is a mock implementation of the Adapter interface for testing
type MockAdapter struct {
	name      string
	sendError error
	sendCount int
}

func (m *MockAdapter) Name() string {
	return m.name
}

func (m *MockAdapter) Send(ctx context.Context, req interface{}) error {
	m.sendCount++
	return m.sendError
}

func TestNewService(t *testing.T) {
	cfg := &config.Config{
		SMTP: config.SMTPConfig{
			Host:     "smtp.test.com",
			Port:     587,
			User:     "test@test.com",
			Password: "password",
			From:     "noreply@test.com",
		},
		Templates: config.TemplatesConfig{
			Dir: "./templates",
		},
	}

	service := NewService(cfg)

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.config != cfg {
		t.Error("Expected service config to match input config")
	}

	if len(service.adapters) == 0 {
		t.Error("Expected at least one adapter to be registered")
	}
}

func TestService_RegisterAdapter(t *testing.T) {
	cfg := &config.Config{}
	service := NewService(cfg)

	mockAdapter := &MockAdapter{name: "mock"}
	service.RegisterAdapter("test", mockAdapter)

	if len(service.adapters) < 1 {
		t.Error("Expected adapter to be registered")
	}

	adapter, ok := service.adapters["test"]
	if !ok {
		t.Error("Expected 'test' adapter to be registered")
	}

	if adapter.Name() != "mock" {
		t.Errorf("Expected adapter name 'mock', got %s", adapter.Name())
	}
}

func TestService_ValidateRequest(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name    string
		req     *SendRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &SendRequest{
				To:       "user@example.com",
				Channel:  ChannelEmail,
				Template: "welcome",
			},
			wantErr: false,
		},
		{
			name: "missing recipient",
			req: &SendRequest{
				To:       "",
				Channel:  ChannelEmail,
				Template: "welcome",
			},
			wantErr: true,
		},
		{
			name: "missing channel",
			req: &SendRequest{
				To:       "user@example.com",
				Channel:  "",
				Template: "welcome",
			},
			wantErr: true,
		},
		{
			name: "missing template",
			req: &SendRequest{
				To:      "user@example.com",
				Channel: ChannelEmail,
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

func TestService_Send(t *testing.T) {
	cfg := &config.Config{}
	service := NewService(cfg)

	// Register mock adapter
	mockAdapter := &MockAdapter{name: "mock"}
	service.RegisterAdapter(ChannelEmail, mockAdapter)

	ctx := context.Background()

	tests := []struct {
		name    string
		req     *SendRequest
		wantErr bool
	}{
		{
			name: "valid send",
			req: &SendRequest{
				To:       "user@example.com",
				Channel:  ChannelEmail,
				Template: "welcome",
				Data:     map[string]interface{}{"Name": "John"},
			},
			wantErr: false,
		},
		{
			name: "invalid request",
			req: &SendRequest{
				To:       "",
				Channel:  ChannelEmail,
				Template: "welcome",
			},
			wantErr: true,
		},
		{
			name: "unsupported channel",
			req: &SendRequest{
				To:       "user@example.com",
				Channel:  "unsupported",
				Template: "welcome",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Send(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Check that mock adapter was called
	if mockAdapter.sendCount == 0 {
		t.Error("Expected mock adapter to be called at least once")
	}
}

func TestService_GetSupportedChannels(t *testing.T) {
	cfg := &config.Config{
		SMTP: config.SMTPConfig{
			Host:     "smtp.test.com",
			Port:     587,
			User:     "test@test.com",
			Password: "password",
		},
		Templates: config.TemplatesConfig{
			Dir: "./templates",
		},
	}
	service := NewService(cfg)

	channels := service.GetSupportedChannels()

	if len(channels) == 0 {
		t.Error("Expected at least one supported channel")
	}

	// Email should be supported by default
	hasEmail := false
	for _, ch := range channels {
		if ch == "email" {
			hasEmail = true
			break
		}
	}

	if !hasEmail {
		t.Error("Expected email channel to be supported")
	}
}
