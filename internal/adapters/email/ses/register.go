package ses

import "github.com/huzaifabhutta/notify-core/internal/adapters"

func init() {
	// Auto-register AWS SES adapter in global registry on import
	// This follows Go best practice of self-registration in init()
	adapters.MustRegister(
		adapters.Metadata{
			Name:        "ses",
			Type:        adapters.TypeEmail,
			Description: "AWS SES email adapter with native SDK - production-ready with bounce handling, configuration sets, and full SES features",
			Version:     "1.0.0",
		},
		NewAdapter,
	)
}
