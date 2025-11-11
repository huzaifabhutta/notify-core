package cloud

import (
	"github.com/huzaifabhutta/notify-core/internal/adapters"
	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/whatsapp"
)

func init() {
	// Auto-register WhatsApp Cloud API adapter in global registry on import
	// This follows Go best practice of self-registration in init()
	adapters.MustRegister(
		adapters.Metadata{
			Name:        "whatsapp-cloud",
			Type:        adapters.TypeWhatsApp,
			Description: "WhatsApp Cloud API adapter - vendor-agnostic (Facebook, 360dialog, Twilio, on-premises)",
			Version:     "1.0.0",
		},
		NewAdapter,
	)
}

// Config holds WhatsApp adapter configuration
type Config = config.WhatsAppConfig

// NewAdapter creates a new WhatsApp Cloud API adapter
// This is the factory function registered with the adapter registry
func NewAdapter(cfg interface{}) (adapters.Adapter, error) {
	waCfg, ok := cfg.(*config.WhatsAppConfig)
	if !ok {
		return nil, adapters.ErrInvalidAdapter
	}

	// Create the underlying whatsapp adapter
	return whatsapp.NewAdapter(waCfg), nil
}
