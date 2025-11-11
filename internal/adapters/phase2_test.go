package adapters_test

import (
	"testing"

	"github.com/huzaifabhutta/notify-core/internal/adapters"
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/email/ses"   // Register SES
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/email/smtp"  // Register SMTP
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/sms/sns"     // Register SNS
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/whatsapp/cloud" // Register WhatsApp
)

// TestPhase2AdapterRegistration verifies all Phase 2 adapters are registered
func TestPhase2AdapterRegistration(t *testing.T) {
	tests := []struct {
		name         string
		adapterName  string
		adapterType  adapters.Type
		wantRegister bool
	}{
		{
			name:         "SMTP adapter registered",
			adapterName:  "smtp",
			adapterType:  adapters.TypeEmail,
			wantRegister: true,
		},
		{
			name:         "SES adapter registered",
			adapterName:  "ses",
			adapterType:  adapters.TypeEmail,
			wantRegister: true,
		},
		{
			name:         "WhatsApp Cloud adapter registered",
			adapterName:  "whatsapp-cloud",
			adapterType:  adapters.TypeWhatsApp,
			wantRegister: true,
		},
		{
			name:         "SNS adapter registered",
			adapterName:  "sns",
			adapterType:  adapters.TypeSMS,
			wantRegister: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if adapter is registered
			if got := adapters.Has(tt.adapterName); got != tt.wantRegister {
				t.Errorf("adapters.Has(%q) = %v, want %v", tt.adapterName, got, tt.wantRegister)
			}

			// Check metadata
			meta := adapters.GetMetadata(tt.adapterName)
			if tt.wantRegister {
				if meta == nil {
					t.Errorf("Expected metadata for %s, got nil", tt.adapterName)
					return
				}

				if meta.Type != tt.adapterType {
					t.Errorf("Expected type %s, got %s", tt.adapterType, meta.Type)
				}

				if meta.Name != tt.adapterName {
					t.Errorf("Expected name %s, got %s", tt.adapterName, meta.Name)
				}

				if meta.Description == "" {
					t.Error("Expected non-empty description")
				}

				if meta.Version == "" {
					t.Error("Expected non-empty version")
				}
			}
		})
	}
}

// TestListAdaptersByTypePhase2 verifies adapters can be listed by type
func TestListAdaptersByTypePhase2(t *testing.T) {
	// Email adapters
	emailAdapters := adapters.ListByType(adapters.TypeEmail)
	if len(emailAdapters) < 2 {
		t.Errorf("Expected at least 2 email adapters (smtp, ses), got %d", len(emailAdapters))
	}

	expectedEmail := map[string]bool{
		"smtp": false,
		"ses":  false,
	}

	for _, name := range emailAdapters {
		if _, exists := expectedEmail[name]; exists {
			expectedEmail[name] = true
		}
	}

	for name, found := range expectedEmail {
		if !found {
			t.Errorf("Expected email adapter %s not found", name)
		}
	}

	// WhatsApp adapters
	whatsappAdapters := adapters.ListByType(adapters.TypeWhatsApp)
	if len(whatsappAdapters) == 0 {
		t.Error("Expected at least one whatsapp adapter")
	}

	found := false
	for _, name := range whatsappAdapters {
		if name == "whatsapp-cloud" {
			found = true
			break
		}
	}

	if !found {
		t.Error("WhatsApp Cloud adapter not found")
	}

	// SMS adapters
	smsAdapters := adapters.ListByType(adapters.TypeSMS)
	if len(smsAdapters) == 0 {
		t.Error("Expected at least one SMS adapter")
	}

	found = false
	for _, name := range smsAdapters {
		if name == "sns" {
			found = true
			break
		}
	}

	if !found {
		t.Error("SNS adapter not found")
	}
}

// TestListAllAdaptersPhase2 verifies all Phase 2 adapters are listed
func TestListAllAdaptersPhase2(t *testing.T) {
	allAdapters := adapters.List()

	if len(allAdapters) < 4 {
		t.Errorf("Expected at least 4 adapters, got %d", len(allAdapters))
	}

	// Check for expected adapters
	expectedAdapters := map[string]bool{
		"smtp":           false,
		"ses":            false,
		"whatsapp-cloud": false,
		"sns":            false,
	}

	for _, name := range allAdapters {
		if _, exists := expectedAdapters[name]; exists {
			expectedAdapters[name] = true
		}
	}

	for name, found := range expectedAdapters {
		if !found {
			t.Errorf("Expected adapter %s not found in registry", name)
		}
	}
}

// TestAdapterMetadataPhase2 verifies adapter metadata is correct
func TestAdapterMetadataPhase2(t *testing.T) {
	tests := []struct {
		name            string
		adapterName     string
		expectedType    adapters.Type
		shouldContain   string // Description should contain this
	}{
		{
			name:          "SMTP metadata",
			adapterName:   "smtp",
			expectedType:  adapters.TypeEmail,
			shouldContain: "SMTP",
		},
		{
			name:          "SES metadata",
			adapterName:   "ses",
			expectedType:  adapters.TypeEmail,
			shouldContain: "AWS SES",
		},
		{
			name:          "WhatsApp Cloud metadata",
			adapterName:   "whatsapp-cloud",
			expectedType:  adapters.TypeWhatsApp,
			shouldContain: "WhatsApp",
		},
		{
			name:          "SNS metadata",
			adapterName:   "sns",
			expectedType:  adapters.TypeSMS,
			shouldContain: "AWS SNS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := adapters.GetMetadata(tt.adapterName)
			if meta == nil {
				t.Fatalf("Expected metadata for %s, got nil", tt.adapterName)
			}

			if meta.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, meta.Type)
			}

			if meta.Version == "" {
				t.Error("Version should not be empty")
			}

			// Check description contains expected text
			if tt.shouldContain != "" {
				if !contains(meta.Description, tt.shouldContain) {
					t.Errorf("Description %q should contain %q", meta.Description, tt.shouldContain)
				}
			}
		})
	}
}

// contains checks if a string contains a substring (case-insensitive check could be added)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
