package models

import "testing"

func TestTenant_HasEmailConfig(t *testing.T) {
	tests := []struct {
		name   string
		tenant *Tenant
		want   bool
	}{
		{
			name: "full email config",
			tenant: &Tenant{
				SMTPHost:     "smtp.test.com",
				SMTPUser:     "user@test.com",
				SMTPPassword: "password",
			},
			want: true,
		},
		{
			name: "missing host",
			tenant: &Tenant{
				SMTPHost:     "",
				SMTPUser:     "user@test.com",
				SMTPPassword: "password",
			},
			want: false,
		},
		{
			name: "missing user",
			tenant: &Tenant{
				SMTPHost:     "smtp.test.com",
				SMTPUser:     "",
				SMTPPassword: "password",
			},
			want: false,
		},
		{
			name: "missing password",
			tenant: &Tenant{
				SMTPHost:     "smtp.test.com",
				SMTPUser:     "user@test.com",
				SMTPPassword: "",
			},
			want: false,
		},
		{
			name:   "empty tenant",
			tenant: &Tenant{},
			want:   false,
		},
		{
			name: "partial config - only host",
			tenant: &Tenant{
				SMTPHost: "smtp.test.com",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tenant.HasEmailConfig(); got != tt.want {
				t.Errorf("Tenant.HasEmailConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenant_HasWhatsAppConfig(t *testing.T) {
	tests := []struct {
		name   string
		tenant *Tenant
		want   bool
	}{
		{
			name: "full whatsapp config",
			tenant: &Tenant{
				WAToken:   "test-token",
				WAPhoneID: "123456789",
			},
			want: true,
		},
		{
			name: "missing token",
			tenant: &Tenant{
				WAToken:   "",
				WAPhoneID: "123456789",
			},
			want: false,
		},
		{
			name: "missing phone id",
			tenant: &Tenant{
				WAToken:   "test-token",
				WAPhoneID: "",
			},
			want: false,
		},
		{
			name:   "empty tenant",
			tenant: &Tenant{},
			want:   false,
		},
		{
			name: "partial config - only token",
			tenant: &Tenant{
				WAToken: "test-token",
			},
			want: false,
		},
		{
			name: "partial config - only phone id",
			tenant: &Tenant{
				WAPhoneID: "123456789",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tenant.HasWhatsAppConfig(); got != tt.want {
				t.Errorf("Tenant.HasWhatsAppConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenant_HasSMSConfig(t *testing.T) {
	tests := []struct {
		name   string
		tenant *Tenant
		want   bool
	}{
		{
			name: "full SMS config",
			tenant: &Tenant{
				SMSProvider: "twilio",
				SMSAPIKey:   "api-key",
			},
			want: true,
		},
		{
			name: "missing provider",
			tenant: &Tenant{
				SMSProvider: "",
				SMSAPIKey:   "api-key",
			},
			want: false,
		},
		{
			name: "missing api key",
			tenant: &Tenant{
				SMSProvider: "twilio",
				SMSAPIKey:   "",
			},
			want: false,
		},
		{
			name:   "empty tenant",
			tenant: &Tenant{},
			want:   false,
		},
		{
			name: "partial config - only provider",
			tenant: &Tenant{
				SMSProvider: "twilio",
			},
			want: false,
		},
		{
			name: "partial config - only api key",
			tenant: &Tenant{
				SMSAPIKey: "api-key",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tenant.HasSMSConfig(); got != tt.want {
				t.Errorf("Tenant.HasSMSConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenant_MultiChannelConfig(t *testing.T) {
	tests := []struct {
		name          string
		tenant        *Tenant
		wantEmail     bool
		wantWhatsApp  bool
		wantSMS       bool
		description   string
	}{
		{
			name: "all channels configured",
			tenant: &Tenant{
				SMTPHost:     "smtp.test.com",
				SMTPUser:     "user@test.com",
				SMTPPassword: "password",
				WAToken:      "wa-token",
				WAPhoneID:    "phone-id",
				SMSProvider:  "twilio",
				SMSAPIKey:    "sms-key",
			},
			wantEmail:    true,
			wantWhatsApp: true,
			wantSMS:      true,
			description:  "tenant with all channels",
		},
		{
			name: "only email configured",
			tenant: &Tenant{
				SMTPHost:     "smtp.test.com",
				SMTPUser:     "user@test.com",
				SMTPPassword: "password",
			},
			wantEmail:    true,
			wantWhatsApp: false,
			wantSMS:      false,
			description:  "tenant with only email",
		},
		{
			name: "only whatsapp configured",
			tenant: &Tenant{
				WAToken:   "wa-token",
				WAPhoneID: "phone-id",
			},
			wantEmail:    false,
			wantWhatsApp: true,
			wantSMS:      false,
			description:  "tenant with only whatsapp",
		},
		{
			name: "only SMS configured",
			tenant: &Tenant{
				SMSProvider: "twilio",
				SMSAPIKey:   "sms-key",
			},
			wantEmail:    false,
			wantWhatsApp: false,
			wantSMS:      true,
			description:  "tenant with only SMS",
		},
		{
			name:         "no channels configured",
			tenant:       &Tenant{},
			wantEmail:    false,
			wantWhatsApp: false,
			wantSMS:      false,
			description:  "tenant with no channels (fallback to global)",
		},
		{
			name: "email and whatsapp only",
			tenant: &Tenant{
				SMTPHost:     "smtp.test.com",
				SMTPUser:     "user@test.com",
				SMTPPassword: "password",
				WAToken:      "wa-token",
				WAPhoneID:    "phone-id",
			},
			wantEmail:    true,
			wantWhatsApp: true,
			wantSMS:      false,
			description:  "tenant with email and whatsapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEmail := tt.tenant.HasEmailConfig()
			gotWhatsApp := tt.tenant.HasWhatsAppConfig()
			gotSMS := tt.tenant.HasSMSConfig()

			if gotEmail != tt.wantEmail {
				t.Errorf("%s: HasEmailConfig() = %v, want %v", tt.description, gotEmail, tt.wantEmail)
			}
			if gotWhatsApp != tt.wantWhatsApp {
				t.Errorf("%s: HasWhatsAppConfig() = %v, want %v", tt.description, gotWhatsApp, tt.wantWhatsApp)
			}
			if gotSMS != tt.wantSMS {
				t.Errorf("%s: HasSMSConfig() = %v, want %v", tt.description, gotSMS, tt.wantSMS)
			}
		})
	}
}
