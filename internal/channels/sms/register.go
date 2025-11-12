package sms

import (
	"github.com/huzaifabhutta/notify-core/internal/channels"
)

func init() {
	// Auto-register SMS channel on import
	channels.MustRegister(
		channels.Metadata{
			Name:        "sms",
			Type:        channels.TypeMessaging,
			Description: "SMS notification channel with AWS SNS support",
			Version:     "1.0.0",
			Adapters:    []string{"sns"},
			Features: []channels.Feature{
				channels.FeatureTemplates,
				channels.FeatureDeliveryReceipt,
				channels.FeatureBatching,
			},
		},
		NewChannel,
	)
}
