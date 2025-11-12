package email

import (
	"github.com/huzaifabhutta/notify-core/internal/channels"
)

func init() {
	// Auto-register email channel on import
	channels.MustRegister(
		channels.Metadata{
			Name:        "email",
			Type:        channels.TypeMessaging,
			Description: "Email notification channel with SMTP/SES support",
			Version:     "1.0.0",
			Adapters:    []string{"smtp", "ses"},
			Features: []channels.Feature{
				channels.FeatureAttachments,
				channels.FeatureRichText,
				channels.FeatureTemplates,
			},
		},
		NewChannel,
	)
}
