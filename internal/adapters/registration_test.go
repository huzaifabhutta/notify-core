package adapters_test

import (
	"testing"

	"github.com/huzaifabhutta/notify-core/internal/adapters"
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/email/smtp"       // Import to trigger init()
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/whatsapp/cloud" // Import to trigger init()
)

func TestAdapterRegistration(t *testing.T) {
	// Test that adapters are auto-registered via init()

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
			name:         "WhatsApp Cloud adapter registered",
			adapterName:  "whatsapp-cloud",
			adapterType:  adapters.TypeWhatsApp,
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

func TestListAdaptersByType(t *testing.T) {
	emailAdapters := adapters.ListByType(adapters.TypeEmail)
	if len(emailAdapters) == 0 {
		t.Error("Expected at least one email adapter")
	}

	found := false
	for _, name := range emailAdapters {
		if name == "smtp" {
			found = true
			break
		}
	}

	if !found {
		t.Error("SMTP adapter not found in email adapters list")
	}

	whatsappAdapters := adapters.ListByType(adapters.TypeWhatsApp)
	if len(whatsappAdapters) == 0 {
		t.Error("Expected at least one whatsapp adapter")
	}

	found = false
	for _, name := range whatsappAdapters {
		if name == "whatsapp-cloud" {
			found = true
			break
		}
	}

	if !found {
		t.Error("WhatsApp Cloud adapter not found in whatsapp adapters list")
	}
}

func TestListAllAdapters(t *testing.T) {
	allAdapters := adapters.List()

	if len(allAdapters) < 2 {
		t.Errorf("Expected at least 2 adapters, got %d", len(allAdapters))
	}

	// Check for expected adapters
	expectedAdapters := map[string]bool{
		"smtp":           false,
		"whatsapp-cloud": false,
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
