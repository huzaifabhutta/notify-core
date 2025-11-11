package smtp

import (
	"github.com/huzaifabhutta/notify-core/internal/adapters"
	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/email"
)

func init() {
	// Auto-register SMTP adapter in global registry on import
	// This follows Go best practice of self-registration in init()
	adapters.MustRegister(
		adapters.Metadata{
			Name:        "smtp",
			Type:        adapters.TypeEmail,
			Description: "SMTP email adapter - works with any SMTP server (Gmail, SendGrid, Mailgun, AWS SES SMTP, etc.)",
			Version:     "1.0.0",
		},
		NewAdapter,
	)
}

// Config holds SMTP adapter configuration
type Config struct {
	SMTPConfig     *config.SMTPConfig
	TemplateConfig *config.TemplatesConfig
}

// NewAdapter creates a new SMTP email adapter
// This is the factory function registered with the adapter registry
func NewAdapter(cfg interface{}) (adapters.Adapter, error) {
	smtpCfg, ok := cfg.(*Config)
	if !ok {
		// Try to cast to SMTPConfig directly for backward compatibility
		if directCfg, ok := cfg.(*config.SMTPConfig); ok {
			return email.NewAdapter(directCfg, &config.TemplatesConfig{}), nil
		}
		return nil, adapters.ErrInvalidAdapter
	}

	// Create the underlying email adapter
	return email.NewAdapter(smtpCfg.SMTPConfig, smtpCfg.TemplateConfig), nil
}
