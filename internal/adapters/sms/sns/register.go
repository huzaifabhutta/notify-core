package sns

import "github.com/huzaifabhutta/notify-core/internal/adapters"

func init() {
	// Auto-register AWS SNS adapter in global registry on import
	// This follows Go best practice of self-registration in init()
	adapters.MustRegister(
		adapters.Metadata{
			Name:        "sns",
			Type:        adapters.TypeSMS,
			Description: "AWS SNS SMS adapter with native SDK - cost-effective SMS delivery ($0.00645/SMS in US)",
			Version:     "1.0.0",
		},
		NewAdapter,
	)
}
