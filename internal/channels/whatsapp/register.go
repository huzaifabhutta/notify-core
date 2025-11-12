package whatsapp

import (
	"github.com/huzaifabhutta/notify-core/internal/channels"
)

func init() {
	// Auto-register WhatsApp channel on import
	channels.MustRegister(
		channels.Metadata{
			Name:        "whatsapp",
			Type:        channels.TypeMessaging,
			Description: "WhatsApp notification channel with Cloud API support",
			Version:     "1.0.0",
			Adapters:    []string{"whatsapp-cloud"},
			Features: []channels.Feature{
				channels.FeatureAttachments,
				channels.FeatureRichText,
				channels.FeatureTemplates,
				channels.FeatureDeliveryReceipt,
			},
		},
		NewChannel,
	)
}
