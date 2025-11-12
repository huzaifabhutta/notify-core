package services_test

import (
	"context"
	"testing"

	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/services"
	"github.com/rs/zerolog"
)

func TestNotifyServiceV3_Creation(t *testing.T) {
	cfg := &config.Config{
		Adapters: config.AdaptersConfig{
			Email: config.EmailAdapterConfig{
				Default: "smtp",
			},
			SMS: config.SMSAdapterConfig{
				Default: "sns",
			},
		},
		SMTP: config.SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "test@example.com",
		},
	}

	logger := zerolog.Nop()
	service := services.NewNotifyServiceV3(cfg, nil, logger)

	if service == nil {
		t.Fatal("Expected non-nil service")
	}
}

func TestNotifyServiceV3_Send_ValidationErrors(t *testing.T) {
	cfg := &config.Config{
		Adapters: config.AdaptersConfig{
			Email: config.EmailAdapterConfig{
				Default: "smtp",
			},
		},
		SMTP: config.SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "test@example.com",
		},
	}

	logger := zerolog.Nop()
	service := services.NewNotifyServiceV3(cfg, nil, logger)

	tests := []struct {
		name    string
		request *services.SendRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing channel",
			request: &services.SendRequest{
				To:   "test@example.com",
				Body: "Test message",
			},
			wantErr: true,
			errMsg:  "channel is required",
		},
		{
			name: "missing recipient",
			request: &services.SendRequest{
				Channel: "email",
				Body:    "Test message",
			},
			wantErr: true,
			errMsg:  "recipient (to) is required",
		},
		{
			name: "unsupported channel",
			request: &services.SendRequest{
				Channel: "telegram",
				To:      "user123",
				Body:    "Test message",
			},
			wantErr: true,
			errMsg:  "unsupported channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := service.Send(ctx, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestNotifyServiceV3_ChannelRegistry_Integration(t *testing.T) {
	// This test verifies that V3 service correctly uses the channel registry
	cfg := &config.Config{
		Adapters: config.AdaptersConfig{
			Email: config.EmailAdapterConfig{
				Default: "smtp",
			},
			SMS: config.SMSAdapterConfig{
				Default: "sns",
				SNS: config.SNSConfig{
					Region:  "us-east-1",
					SMSType: "Transactional",
				},
			},
		},
		SMTP: config.SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			User: "test@example.com",
			Password: "password",
			From: "test@example.com",
		},
		WhatsApp: config.WhatsAppConfig{
			Token:      "test-token",
			PhoneID:    "123456",
			APIVersion: "v18.0",
		},
	}

	logger := zerolog.Nop()
	service := services.NewNotifyServiceV3(cfg, nil, logger)

	// Test email channel configuration
	t.Run("email channel config", func(t *testing.T) {
		ctx := context.Background()
		req := &services.SendRequest{
			Channel: "email",
			To:      "invalid-email", // Will fail validation
			Subject: "Test",
			Body:    "Test body",
		}

		_, err := service.Send(ctx, req)
		// Should get a validation error from the email channel
		if err == nil {
			t.Error("Expected validation error for invalid email")
		}
	})

	// Test SMS channel configuration
	t.Run("sms channel config", func(t *testing.T) {
		ctx := context.Background()
		req := &services.SendRequest{
			Channel: "sms",
			To:      "1234", // Will fail validation (not E.164 format)
			Body:    "Test SMS",
		}

		_, err := service.Send(ctx, req)
		// Should get a validation error from the SMS channel
		if err == nil {
			t.Error("Expected validation error for invalid phone number")
		}
	})

	// Test WhatsApp channel configuration
	t.Run("whatsapp channel config", func(t *testing.T) {
		ctx := context.Background()
		req := &services.SendRequest{
			Channel: "whatsapp",
			To:      "invalid", // Will fail validation
			Body:    "Test WhatsApp",
		}

		_, err := service.Send(ctx, req)
		// Should get a validation error from the WhatsApp channel
		if err == nil {
			t.Error("Expected validation error for invalid phone number")
		}
	})
}

func TestNotifyServiceV3_Backward_Compatibility(t *testing.T) {
	// This test ensures V3 has the same interface as V2
	cfg := &config.Config{
		Adapters: config.AdaptersConfig{
			Email: config.EmailAdapterConfig{
				Default: "smtp",
			},
		},
		SMTP: config.SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "test@example.com",
		},
	}

	logger := zerolog.Nop()

	// Create both V2 and V3 services
	serviceV2 := services.NewNotifyServiceV2(cfg, nil, logger)
	serviceV3 := services.NewNotifyServiceV3(cfg, nil, logger)

	// Verify both services have the same Send method signature
	if serviceV2 == nil || serviceV3 == nil {
		t.Fatal("Services should not be nil")
	}

	// Both should accept the same request type
	ctx := context.Background()
	req := &services.SendRequest{
		Channel: "email",
		To:      "invalid-email",
		Subject: "Test",
		Body:    "Test",
	}

	// Both should return errors for invalid input
	_, errV2 := serviceV2.Send(ctx, req)
	_, errV3 := serviceV3.Send(ctx, req)

	if errV2 == nil || errV3 == nil {
		t.Error("Both services should return validation errors")
	}
}

func BenchmarkNotifyServiceV3_Send(b *testing.B) {
	cfg := &config.Config{
		Adapters: config.AdaptersConfig{
			Email: config.EmailAdapterConfig{
				Default: "smtp",
			},
		},
		SMTP: config.SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "test@example.com",
		},
	}

	logger := zerolog.Nop()
	service := services.NewNotifyServiceV3(cfg, nil, logger)

	ctx := context.Background()
	req := &services.SendRequest{
		Channel: "email",
		To:      "invalid",
		Subject: "Test",
		Body:    "Test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Send(ctx, req)
	}
}

func BenchmarkNotifyServiceV2_vs_V3(b *testing.B) {
	cfg := &config.Config{
		Adapters: config.AdaptersConfig{
			Email: config.EmailAdapterConfig{
				Default: "smtp",
			},
		},
		SMTP: config.SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "test@example.com",
		},
	}

	logger := zerolog.Nop()
	ctx := context.Background()
	req := &services.SendRequest{
		Channel: "email",
		To:      "invalid",
		Subject: "Test",
		Body:    "Test",
	}

	b.Run("V2", func(b *testing.B) {
		serviceV2 := services.NewNotifyServiceV2(cfg, nil, logger)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = serviceV2.Send(ctx, req)
		}
	})

	b.Run("V3", func(b *testing.B) {
		serviceV3 := services.NewNotifyServiceV3(cfg, nil, logger)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = serviceV3.Send(ctx, req)
		}
	})
}
