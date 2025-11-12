package sms

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/huzaifabhutta/notify-core/internal/adapters"
	snsAdapter "github.com/huzaifabhutta/notify-core/internal/adapters/sms/sns"
	"github.com/huzaifabhutta/notify-core/internal/channels"
	"github.com/rs/zerolog"
)

// Channel implements the SMS notification channel
type Channel struct {
	config       *Config
	adapterName  string
	credResolver CredentialResolver
	logger       zerolog.Logger
}

// Config holds SMS channel configuration
type Config struct {
	DefaultAdapter string              // Currently only "sns"
	SNS            *SNSConfig          // SNS configuration
	Credentials    CredentialResolver  // For tenant-specific credentials
	Logger         *zerolog.Logger     // Logger instance
}

// SNSConfig holds SNS-specific configuration
type SNSConfig struct {
	Region          string
	SenderID        string
	SMSType         string // "Transactional" or "Promotional"
	AccessKeyID     string
	SecretAccessKey string
}

// CredentialResolver resolves SMS credentials for tenants
type CredentialResolver interface {
	ResolveSMS(ctx context.Context) (*SMSCredentials, error)
}

// SMSCredentials represents resolved SMS credentials
type SMSCredentials struct {
	Provider string // "sns", "twilio", etc.
	Source   string // "tenant" or "global"
	// Provider-specific fields can be added here
}

// NewChannel creates a new SMS channel
func NewChannel(cfg interface{}) (channels.Channel, error) {
	smsCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("%w: expected *sms.Config", channels.ErrInvalidChannel)
	}

	if smsCfg.DefaultAdapter == "" {
		smsCfg.DefaultAdapter = "sns" // Default to SNS
	}

	// Validate adapter name
	if smsCfg.DefaultAdapter != "sns" {
		return nil, fmt.Errorf("%w: unsupported adapter %s (only 'sns' is currently supported)", channels.ErrInvalidChannel, smsCfg.DefaultAdapter)
	}

	logger := zerolog.Nop()
	if smsCfg.Logger != nil {
		logger = smsCfg.Logger.With().Str("channel", "sms").Logger()
	}

	return &Channel{
		config:       smsCfg,
		adapterName:  smsCfg.DefaultAdapter,
		credResolver: smsCfg.Credentials,
		logger:       logger,
	}, nil
}

// Name returns the channel name
func (c *Channel) Name() string {
	return "sms"
}

// Type returns the channel type
func (c *Channel) Type() channels.Type {
	return channels.TypeMessaging
}

// Validate validates an SMS send request
func (c *Channel) Validate(ctx context.Context, req *channels.SendRequest) error {
	if req.To == "" {
		return fmt.Errorf("SMS recipient (to) is required")
	}

	// Basic phone number validation
	if !isValidPhoneNumber(req.To) {
		return fmt.Errorf("invalid phone number: %s (must be in E.164 format, e.g., +1234567890)", req.To)
	}

	// Body validation
	if req.Body == "" && req.Template == "" {
		return fmt.Errorf("SMS body or template is required")
	}

	// Check message length (SMS limit is typically 160 characters for GSM, 70 for Unicode)
	if len(req.Body) > 1600 {
		return fmt.Errorf("SMS body too long (%d characters, max 1600 for concatenated messages)", len(req.Body))
	}

	// SMS doesn't support attachments
	if len(req.Attachments) > 0 {
		return fmt.Errorf("SMS does not support attachments")
	}

	return nil
}

// Send sends an SMS notification
func (c *Channel) Send(ctx context.Context, req *channels.SendRequest) (*channels.SendResponse, error) {
	// Validate request
	if err := c.Validate(ctx, req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Resolve credentials
	var creds *SMSCredentials
	var err error
	if c.credResolver != nil {
		creds, err = c.credResolver.ResolveSMS(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve credentials: %w", err)
		}
	} else {
		// Use default configuration
		creds = &SMSCredentials{
			Provider: "sns",
			Source:   "global",
		}
	}

	// Create adapter configuration
	var adapterConfig interface{}
	var adapter adapters.Adapter

	switch c.adapterName {
	case "sns":
		if c.config.SNS == nil {
			return nil, fmt.Errorf("SNS configuration is required when using SNS adapter")
		}

		adapterConfig = &snsAdapter.Config{
			Region:          c.config.SNS.Region,
			SenderID:        c.config.SNS.SenderID,
			SMSType:         c.config.SNS.SMSType,
			AccessKeyID:     c.config.SNS.AccessKeyID,
			SecretAccessKey: c.config.SNS.SecretAccessKey,
		}

		adapter, err = adapters.Get("sns", adapterConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get SNS adapter: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported adapter: %s", c.adapterName)
	}

	// Convert channel request to adapter request
	adapterReq := convertToAdapterRequest(req)

	// Send via adapter
	c.logger.Debug().
		Str("adapter", c.adapterName).
		Str("to", maskPhoneNumber(req.To)).
		Msg("Sending SMS via adapter")

	messageID, err := adapter.Send(ctx, adapterReq)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("adapter", c.adapterName).
			Str("to", maskPhoneNumber(req.To)).
			Msg("Failed to send SMS")
		return nil, fmt.Errorf("adapter send failed: %w", err)
	}

	c.logger.Info().
		Str("adapter", c.adapterName).
		Str("to", maskPhoneNumber(req.To)).
		Str("message_id", messageID).
		Msg("SMS sent successfully")

	return &channels.SendResponse{
		MessageID: messageID,
		Channel:   "sms",
		Adapter:   c.adapterName,
		Metadata: map[string]string{
			"source": creds.Source,
		},
	}, nil
}

// GetAdapters returns supported adapters
func (c *Channel) GetAdapters() []string {
	return []string{"sns"}
}

// SupportsFeature checks if a feature is supported
func (c *Channel) SupportsFeature(feature channels.Feature) bool {
	switch feature {
	case channels.FeatureAttachments:
		return false // SMS doesn't support attachments
	case channels.FeatureRichText:
		return false // SMS is plain text only
	case channels.FeatureTemplates:
		return true  // Templates can be used for SMS
	case channels.FeatureDeliveryReceipt:
		return true  // SNS supports delivery receipts
	case channels.FeatureRetry:
		return false // Could be implemented in future
	case channels.FeatureBatching:
		return true  // SNS supports batch sending
	case channels.FeatureScheduling:
		return false // Not supported yet
	default:
		return false
	}
}

// Helper functions

// isValidPhoneNumber performs basic phone number validation
// Requires E.164 format: +[country code][number]
func isValidPhoneNumber(phone string) bool {
	// E.164 format: + followed by 1-15 digits
	phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	return phoneRegex.MatchString(phone)
}

// maskPhoneNumber masks phone number for logging
func maskPhoneNumber(phone string) string {
	if len(phone) < 4 {
		return "***"
	}

	// Keep country code and last 4 digits
	if strings.HasPrefix(phone, "+") {
		if len(phone) <= 7 {
			return phone[0:2] + "***" + phone[len(phone)-2:]
		}
		return phone[0:3] + "***" + phone[len(phone)-4:]
	}

	return "***" + phone[len(phone)-4:]
}

// convertToAdapterRequest converts channel request to adapter-specific request
func convertToAdapterRequest(req *channels.SendRequest) map[string]interface{} {
	adapterReq := map[string]interface{}{
		"to":   req.To,
		"body": req.Body,
	}

	// Add optional fields
	if req.Template != "" {
		adapterReq["template"] = req.Template
		adapterReq["data"] = req.Data
	}

	// Add metadata
	if len(req.Metadata) > 0 {
		adapterReq["metadata"] = req.Metadata
	}

	return adapterReq
}
