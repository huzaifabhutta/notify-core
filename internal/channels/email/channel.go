package email

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/huzaifabhutta/notify-core/internal/adapters"
	emailAdapter "github.com/huzaifabhutta/notify-core/internal/adapters/email/smtp"
	sesAdapter "github.com/huzaifabhutta/notify-core/internal/adapters/email/ses"
	"github.com/huzaifabhutta/notify-core/internal/channels"
	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/rs/zerolog"
)

// Channel implements the email notification channel
type Channel struct {
	config         *Config
	adapterName    string
	credResolver   CredentialResolver
	logger         zerolog.Logger
}

// Config holds email channel configuration
type Config struct {
	DefaultAdapter string                // "smtp" or "ses"
	SMTP           *config.SMTPConfig    // SMTP configuration
	SES            *SESConfig            // SES configuration
	Credentials    CredentialResolver   // For tenant-specific credentials
	Logger         *zerolog.Logger       // Logger instance
}

// SESConfig holds SES-specific configuration
type SESConfig struct {
	Region           string
	FromEmail        string
	ConfigurationSet string
	AccessKeyID      string
	SecretAccessKey  string
	RoleARN          string
}

// CredentialResolver resolves email credentials for tenants
type CredentialResolver interface {
	ResolveEmail(ctx context.Context) (*EmailCredentials, error)
}

// EmailCredentials represents resolved email credentials
type EmailCredentials struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	Source   string // "tenant" or "global"
}

// NewChannel creates a new email channel
func NewChannel(cfg interface{}) (channels.Channel, error) {
	emailCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("%w: expected *email.Config", channels.ErrInvalidChannel)
	}

	if emailCfg.DefaultAdapter == "" {
		emailCfg.DefaultAdapter = "smtp" // Default to SMTP
	}

	// Validate adapter name
	if emailCfg.DefaultAdapter != "smtp" && emailCfg.DefaultAdapter != "ses" {
		return nil, fmt.Errorf("%w: unsupported adapter %s", channels.ErrInvalidChannel, emailCfg.DefaultAdapter)
	}

	logger := zerolog.Nop()
	if emailCfg.Logger != nil {
		logger = emailCfg.Logger.With().Str("channel", "email").Logger()
	}

	return &Channel{
		config:       emailCfg,
		adapterName:  emailCfg.DefaultAdapter,
		credResolver: emailCfg.Credentials,
		logger:       logger,
	}, nil
}

// Name returns the channel name
func (c *Channel) Name() string {
	return "email"
}

// Type returns the channel type
func (c *Channel) Type() channels.Type {
	return channels.TypeMessaging
}

// Validate validates an email send request
func (c *Channel) Validate(ctx context.Context, req *channels.SendRequest) error {
	if req.To == "" {
		return fmt.Errorf("email recipient (to) is required")
	}

	// Basic email validation
	if !isValidEmail(req.To) {
		return fmt.Errorf("invalid email address: %s", req.To)
	}

	// Subject validation
	if req.Subject == "" && req.Template == "" {
		return fmt.Errorf("email subject or template is required")
	}

	// Body validation
	if req.Body == "" && req.Template == "" {
		return fmt.Errorf("email body or template is required")
	}

	// Validate attachments if present
	for i, att := range req.Attachments {
		if att.Filename == "" {
			return fmt.Errorf("attachment %d: filename is required", i)
		}
		if len(att.Content) == 0 && att.URL == "" {
			return fmt.Errorf("attachment %d: content or URL is required", i)
		}
	}

	return nil
}

// Send sends an email notification
func (c *Channel) Send(ctx context.Context, req *channels.SendRequest) (*channels.SendResponse, error) {
	// Validate request
	if err := c.Validate(ctx, req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Resolve credentials
	var creds *EmailCredentials
	var err error
	if c.credResolver != nil {
		creds, err = c.credResolver.ResolveEmail(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve credentials: %w", err)
		}
	} else {
		// Use default configuration if no resolver
		creds = &EmailCredentials{
			Host:   c.config.SMTP.Host,
			Port:   c.config.SMTP.Port,
			User:   c.config.SMTP.User,
			Password: c.config.SMTP.Password,
			From:   c.config.SMTP.From,
			Source: "global",
		}
	}

	// Create adapter configuration based on selected adapter
	var adapterConfig interface{}
	var adapter adapters.Adapter

	switch c.adapterName {
	case "smtp":
		adapterConfig = &emailAdapter.Config{
			SMTPConfig: &config.SMTPConfig{
				Host:     creds.Host,
				Port:     creds.Port,
				User:     creds.User,
				Password: creds.Password,
				From:     creds.From,
			},
			TemplateConfig: &config.TemplatesConfig{},
		}

		adapter, err = adapters.Get("smtp", adapterConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get SMTP adapter: %w", err)
		}

	case "ses":
		if c.config.SES == nil {
			return nil, fmt.Errorf("SES configuration is required when using SES adapter")
		}

		adapterConfig = &sesAdapter.Config{
			Region:           c.config.SES.Region,
			FromEmail:        creds.From,
			ConfigurationSet: c.config.SES.ConfigurationSet,
			AccessKeyID:      c.config.SES.AccessKeyID,
			SecretAccessKey:  c.config.SES.SecretAccessKey,
			RoleARN:          c.config.SES.RoleARN,
		}

		adapter, err = adapters.Get("ses", adapterConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get SES adapter: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported adapter: %s", c.adapterName)
	}

	// Convert channel request to adapter request
	adapterReq := convertToAdapterRequest(req, creds.From)

	// Send via adapter
	c.logger.Debug().
		Str("adapter", c.adapterName).
		Str("to", maskEmail(req.To)).
		Str("from", maskEmail(creds.From)).
		Msg("Sending email via adapter")

	messageID, err := adapter.Send(ctx, adapterReq)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("adapter", c.adapterName).
			Str("to", maskEmail(req.To)).
			Msg("Failed to send email")
		return nil, fmt.Errorf("adapter send failed: %w", err)
	}

	c.logger.Info().
		Str("adapter", c.adapterName).
		Str("to", maskEmail(req.To)).
		Str("message_id", messageID).
		Msg("Email sent successfully")

	return &channels.SendResponse{
		MessageID: messageID,
		Channel:   "email",
		Adapter:   c.adapterName,
		Metadata: map[string]string{
			"source": creds.Source,
		},
	}, nil
}

// GetAdapters returns supported adapters
func (c *Channel) GetAdapters() []string {
	return []string{"smtp", "ses"}
}

// SupportsFeature checks if a feature is supported
func (c *Channel) SupportsFeature(feature channels.Feature) bool {
	switch feature {
	case channels.FeatureAttachments:
		return c.adapterName == "smtp" // SES attachments require different handling
	case channels.FeatureRichText:
		return true
	case channels.FeatureTemplates:
		return true
	case channels.FeatureDeliveryReceipt:
		return c.adapterName == "ses" // SES supports SNS notifications
	case channels.FeatureRetry:
		return false // Could be implemented in future
	case channels.FeatureBatching:
		return c.adapterName == "ses" // SES supports bulk sending
	case channels.FeatureScheduling:
		return false // Not supported yet
	default:
		return false
	}
}

// Helper functions

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	// Basic email regex pattern
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// maskEmail masks email for logging (keeps first char and domain)
func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}

	local := parts[0]
	domain := parts[1]

	if len(local) <= 2 {
		return local[0:1] + "***@" + domain
	}

	return local[0:1] + "***" + local[len(local)-1:] + "@" + domain
}

// convertToAdapterRequest converts channel request to adapter-specific request
func convertToAdapterRequest(req *channels.SendRequest, from string) map[string]interface{} {
	adapterReq := map[string]interface{}{
		"to":      req.To,
		"from":    from,
		"subject": req.Subject,
		"body":    req.Body,
	}

	// Add optional fields
	if req.Template != "" {
		adapterReq["template"] = req.Template
		adapterReq["data"] = req.Data
	}

	// Add attachments if present
	if len(req.Attachments) > 0 {
		adapterReq["attachments"] = req.Attachments
	}

	return adapterReq
}
