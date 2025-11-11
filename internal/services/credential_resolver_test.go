package services

import (
	"testing"

	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/models"
)

func TestResolveEmailCredentials_TenantSpecific(t *testing.T) {
	cfg := &config.Config{
		SMTP: config.SMTPConfig{
			Host:     "smtp.global.com",
			Port:     587,
			User:     "global@example.com",
			Password: "globalpass",
			From:     "noreply@global.com",
		},
	}

	resolver := NewCredentialResolver(cfg)

	tenant := &models.Tenant{
		SMTPHost:     "smtp.tenant.com",
		SMTPPort:     465,
		SMTPUser:     "tenant@example.com",
		SMTPPassword: "tenantpass",
		SMTPFrom:     "noreply@tenant.com",
	}

	creds, err := resolver.ResolveEmailCredentials(tenant)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if creds.Host != "smtp.tenant.com" {
		t.Errorf("Expected tenant host, got %s", creds.Host)
	}

	if creds.Source != "tenant" {
		t.Errorf("Expected source 'tenant', got %s", creds.Source)
	}
}

func TestResolveEmailCredentials_GlobalFallback(t *testing.T) {
	cfg := &config.Config{
		SMTP: config.SMTPConfig{
			Host:     "smtp.global.com",
			Port:     587,
			User:     "global@example.com",
			Password: "globalpass",
			From:     "noreply@global.com",
		},
	}

	resolver := NewCredentialResolver(cfg)

	// Tenant without SMTP config
	tenant := &models.Tenant{
		Name: "test-tenant",
	}

	creds, err := resolver.ResolveEmailCredentials(tenant)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if creds.Host != "smtp.global.com" {
		t.Errorf("Expected global host, got %s", creds.Host)
	}

	if creds.Source != "global" {
		t.Errorf("Expected source 'global', got %s", creds.Source)
	}
}

func TestResolveWhatsAppCredentials_VendorAgnostic(t *testing.T) {
	cfg := &config.Config{
		WhatsApp: config.WhatsAppConfig{
			Token:      "global-token",
			PhoneID:    "global-phone-id",
			BaseURL:    "https://graph.facebook.com",
			APIVersion: "v21.0",
		},
	}

	resolver := NewCredentialResolver(cfg)

	// Test 1: Tenant with 360dialog
	tenant360 := &models.Tenant{
		WAToken:      "360dialog-token",
		WAPhoneID:    "360dialog-phone",
		WABaseURL:    "https://waba.360dialog.io",
		WAAPIVersion: "v21.0",
	}

	creds360, err := resolver.ResolveWhatsAppCredentials(tenant360)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if creds360.BaseURL != "https://waba.360dialog.io" {
		t.Errorf("Expected 360dialog URL, got %s", creds360.BaseURL)
	}

	if creds360.Source != "tenant" {
		t.Errorf("Expected source 'tenant', got %s", creds360.Source)
	}

	// Test 2: Tenant with Twilio
	tenantTwilio := &models.Tenant{
		WAToken:      "twilio-token",
		WAPhoneID:    "twilio-phone",
		WABaseURL:    "https://api.twilio.com",
		WAAPIVersion: "v1",
	}

	credsTwilio, err := resolver.ResolveWhatsAppCredentials(tenantTwilio)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if credsTwilio.BaseURL != "https://api.twilio.com" {
		t.Errorf("Expected Twilio URL, got %s", credsTwilio.BaseURL)
	}

	// Test 3: Global fallback to Facebook
	tenantEmpty := &models.Tenant{
		Name: "empty-tenant",
	}

	credsFacebook, err := resolver.ResolveWhatsAppCredentials(tenantEmpty)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if credsFacebook.BaseURL != "https://graph.facebook.com" {
		t.Errorf("Expected Facebook URL, got %s", credsFacebook.BaseURL)
	}

	if credsFacebook.Source != "global" {
		t.Errorf("Expected source 'global', got %s", credsFacebook.Source)
	}
}

func TestResolveWhatsAppCredentials_EmptyBaseURL(t *testing.T) {
	cfg := &config.Config{
		WhatsApp: config.WhatsAppConfig{
			Token:      "global-token",
			PhoneID:    "global-phone-id",
			BaseURL:    "https://graph.facebook.com",
			APIVersion: "v21.0",
		},
	}

	resolver := NewCredentialResolver(cfg)

	// Tenant with token but no BaseURL (should still work, adapter will use default)
	tenant := &models.Tenant{
		WAToken:   "tenant-token",
		WAPhoneID: "tenant-phone",
		// No BaseURL or APIVersion
	}

	creds, err := resolver.ResolveWhatsAppCredentials(tenant)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Empty BaseURL is fine - adapter will default to Facebook
	if creds.BaseURL != "" && creds.BaseURL != "https://graph.facebook.com" {
		t.Errorf("Expected empty or Facebook URL, got %s", creds.BaseURL)
	}

	if creds.Source != "tenant" {
		t.Errorf("Expected source 'tenant', got %s", creds.Source)
	}
}

func TestResolveEmailCredentials_PartialConfig(t *testing.T) {
	cfg := &config.Config{
		SMTP: config.SMTPConfig{
			Host:     "smtp.global.com",
			Port:     587,
			User:     "global@example.com",
			Password: "globalpass",
			From:     "noreply@global.com",
		},
	}

	resolver := NewCredentialResolver(cfg)

	// Tenant with only host, no user/password (should fallback to global)
	tenant := &models.Tenant{
		SMTPHost: "smtp.tenant.com",
		// Missing SMTPUser and SMTPPassword
	}

	creds, err := resolver.ResolveEmailCredentials(tenant)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should fallback to global because config is incomplete
	if creds.Source != "global" {
		t.Errorf("Expected global fallback for partial config, got %s", creds.Source)
	}
}

func TestTenantHasWhatsAppConfig(t *testing.T) {
	tests := []struct {
		name     string
		tenant   *models.Tenant
		expected bool
	}{
		{
			name: "Full WhatsApp config",
			tenant: &models.Tenant{
				WAToken:   "token",
				WAPhoneID: "phone-id",
			},
			expected: true,
		},
		{
			name: "Missing token",
			tenant: &models.Tenant{
				WAPhoneID: "phone-id",
			},
			expected: false,
		},
		{
			name: "Missing phone ID",
			tenant: &models.Tenant{
				WAToken: "token",
			},
			expected: false,
		},
		{
			name:     "Empty tenant",
			tenant:   &models.Tenant{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tenant.HasWhatsAppConfig()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTenantHasEmailConfig(t *testing.T) {
	tests := []struct {
		name     string
		tenant   *models.Tenant
		expected bool
	}{
		{
			name: "Full email config",
			tenant: &models.Tenant{
				SMTPHost:     "smtp.example.com",
				SMTPUser:     "user",
				SMTPPassword: "pass",
			},
			expected: true,
		},
		{
			name: "Missing password",
			tenant: &models.Tenant{
				SMTPHost: "smtp.example.com",
				SMTPUser: "user",
			},
			expected: false,
		},
		{
			name: "Missing user",
			tenant: &models.Tenant{
				SMTPHost:     "smtp.example.com",
				SMTPPassword: "pass",
			},
			expected: false,
		},
		{
			name: "Missing host",
			tenant: &models.Tenant{
				SMTPUser:     "user",
				SMTPPassword: "pass",
			},
			expected: false,
		},
		{
			name:     "Empty tenant",
			tenant:   &models.Tenant{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tenant.HasEmailConfig()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
