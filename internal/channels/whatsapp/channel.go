package whatsapp

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/huzaifabhutta/notify-core/internal/adapters"
	"github.com/huzaifabhutta/notify-core/internal/channels"
	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/rs/zerolog"
)

// Channel implements the WhatsApp notification channel
type Channel struct {
	config       *Config
	adapterName  string
	credResolver CredentialResolver
	logger       zerolog.Logger
}

// Config holds WhatsApp channel configuration
type Config struct {
	DefaultAdapter string              // Currently only "whatsapp-cloud"
	Cloud          *CloudConfig        // WhatsApp Cloud API configuration
	Credentials    CredentialResolver  // For tenant-specific credentials
	Logger         *zerolog.Logger     // Logger instance
}

// CloudConfig holds WhatsApp Cloud API configuration
// Maps to config.WhatsAppConfig
type CloudConfig = config.WhatsAppConfig

// CredentialResolver resolves WhatsApp credentials for tenants
type CredentialResolver interface {
	ResolveWhatsApp(ctx context.Context) (*WhatsAppCredentials, error)
}

// WhatsAppCredentials represents resolved WhatsApp credentials
type WhatsAppCredentials struct {
	PhoneID string // Maps to config.WhatsAppConfig.PhoneID
	Token   string // Maps to config.WhatsAppConfig.Token
	Source  string // "tenant" or "global"
}

// NewChannel creates a new WhatsApp channel
func NewChannel(cfg interface{}) (channels.Channel, error) {
	whatsappCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("%w: expected *whatsapp.Config", channels.ErrInvalidChannel)
	}

	if whatsappCfg.DefaultAdapter == "" {
		whatsappCfg.DefaultAdapter = "whatsapp-cloud" // Default to Cloud API
	}

	// Validate adapter name
	if whatsappCfg.DefaultAdapter != "whatsapp-cloud" {
		return nil, fmt.Errorf("%w: unsupported adapter %s (only 'whatsapp-cloud' is currently supported)", channels.ErrInvalidChannel, whatsappCfg.DefaultAdapter)
	}

	logger := zerolog.Nop()
	if whatsappCfg.Logger != nil {
		logger = whatsappCfg.Logger.With().Str("channel", "whatsapp").Logger()
	}

	return &Channel{
		config:       whatsappCfg,
		adapterName:  whatsappCfg.DefaultAdapter,
		credResolver: whatsappCfg.Credentials,
		logger:       logger,
	}, nil
}

// Name returns the channel name
func (c *Channel) Name() string {
	return "whatsapp"
}

// Type returns the channel type
func (c *Channel) Type() channels.Type {
	return channels.TypeMessaging
}

// Validate validates a WhatsApp send request
func (c *Channel) Validate(ctx context.Context, req *channels.SendRequest) error {
	if req.To == "" {
		return fmt.Errorf("WhatsApp recipient (to) is required")
	}

	// Basic phone number validation (WhatsApp uses E.164 format)
	if !isValidPhoneNumber(req.To) {
		return fmt.Errorf("invalid phone number: %s (must be in E.164 format without +, e.g., 1234567890)", req.To)
	}

	// Body validation
	if req.Body == "" && req.Template == "" {
		return fmt.Errorf("WhatsApp body or template is required")
	}

	// Check message length (WhatsApp allows up to 4096 characters)
	if len(req.Body) > 4096 {
		return fmt.Errorf("WhatsApp body too long (%d characters, max 4096)", len(req.Body))
	}

	// Validate attachments
	for i, att := range req.Attachments {
		if att.Filename == "" {
			return fmt.Errorf("attachment %d: filename is required", i)
		}
		// WhatsApp supports various media types
		if !isValidMediaType(att.ContentType) {
			return fmt.Errorf("attachment %d: unsupported media type %s", i, att.ContentType)
		}
	}

	return nil
}

// Send sends a WhatsApp notification
func (c *Channel) Send(ctx context.Context, req *channels.SendRequest) (*channels.SendResponse, error) {
	// Validate request
	if err := c.Validate(ctx, req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Resolve credentials
	var creds *WhatsAppCredentials
	var err error
	if c.credResolver != nil {
		creds, err = c.credResolver.ResolveWhatsApp(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve credentials: %w", err)
		}
	} else {
		// Use default configuration
		if c.config.Cloud == nil {
			return nil, fmt.Errorf("WhatsApp Cloud configuration is required")
		}
		creds = &WhatsAppCredentials{
			PhoneID: c.config.Cloud.PhoneID,
			Token:   c.config.Cloud.Token,
			Source:  "global",
		}
	}

	// Create adapter configuration
	var adapterConfig interface{}
	var adapter adapters.Adapter

	switch c.adapterName {
	case "whatsapp-cloud":
		// Use config.WhatsAppConfig directly for the adapter
		adapterConfig = &config.WhatsAppConfig{
			Token:      creds.Token,
			PhoneID:    creds.PhoneID,
			BaseURL:    c.config.Cloud.BaseURL,
			APIVersion: c.config.Cloud.APIVersion,
		}

		adapter, err = adapters.Get("whatsapp-cloud", adapterConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get WhatsApp Cloud adapter: %w", err)
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
		Msg("Sending WhatsApp message via adapter")

	messageID, err := adapter.Send(ctx, adapterReq)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("adapter", c.adapterName).
			Str("to", maskPhoneNumber(req.To)).
			Msg("Failed to send WhatsApp message")
		return nil, fmt.Errorf("adapter send failed: %w", err)
	}

	c.logger.Info().
		Str("adapter", c.adapterName).
		Str("to", maskPhoneNumber(req.To)).
		Str("message_id", messageID).
		Msg("WhatsApp message sent successfully")

	return &channels.SendResponse{
		MessageID: messageID,
		Channel:   "whatsapp",
		Adapter:   c.adapterName,
		Metadata: map[string]string{
			"source": creds.Source,
		},
	}, nil
}

// GetAdapters returns supported adapters
func (c *Channel) GetAdapters() []string {
	return []string{"whatsapp-cloud"}
}

// SupportsFeature checks if a feature is supported
func (c *Channel) SupportsFeature(feature channels.Feature) bool {
	switch feature {
	case channels.FeatureAttachments:
		return true // WhatsApp supports media messages
	case channels.FeatureRichText:
		return true // WhatsApp supports formatted text (bold, italic, etc.)
	case channels.FeatureTemplates:
		return true // WhatsApp supports message templates
	case channels.FeatureDeliveryReceipt:
		return true // WhatsApp supports read receipts and delivery status
	case channels.FeatureRetry:
		return false // Could be implemented in future
	case channels.FeatureBatching:
		return false // WhatsApp doesn't support batch sending natively
	case channels.FeatureScheduling:
		return false // Not supported yet
	default:
		return false
	}
}

// Helper functions

// isValidPhoneNumber performs basic phone number validation
// WhatsApp uses E.164 format without the + prefix
func isValidPhoneNumber(phone string) bool {
	// E.164 format without +: 1-15 digits
	phoneRegex := regexp.MustCompile(`^[1-9]\d{1,14}$`)
	return phoneRegex.MatchString(phone)
}

// isValidMediaType checks if the media type is supported by WhatsApp
func isValidMediaType(contentType string) bool {
	supportedTypes := []string{
		"image/jpeg", "image/png",
		"video/mp4", "video/3gpp",
		"audio/aac", "audio/mp4", "audio/mpeg", "audio/amr", "audio/ogg",
		"application/pdf",
		"application/vnd.ms-powerpoint",
		"application/msword",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}

	for _, supportedType := range supportedTypes {
		if strings.HasPrefix(contentType, supportedType) {
			return true
		}
	}
	return false
}

// maskPhoneNumber masks phone number for logging
func maskPhoneNumber(phone string) string {
	if len(phone) < 4 {
		return "***"
	}

	// Keep first 2 and last 4 digits
	if len(phone) <= 7 {
		return phone[0:2] + "***" + phone[len(phone)-2:]
	}
	return phone[0:2] + "***" + phone[len(phone)-4:]
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

	// Add attachments if present
	if len(req.Attachments) > 0 {
		adapterReq["attachments"] = req.Attachments
	}

	// Add metadata
	if len(req.Metadata) > 0 {
		adapterReq["metadata"] = req.Metadata
	}

	return adapterReq
}
