package services

import (
	"context"
	"fmt"

	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/email"
	"github.com/huzaifabhutta/notify-core/internal/middleware"
	"github.com/huzaifabhutta/notify-core/internal/models"
	"github.com/huzaifabhutta/notify-core/internal/whatsapp"
	"github.com/rs/zerolog"
)

// Channel represents a notification channel type
type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelWhatsApp Channel = "whatsapp"
	ChannelSMS      Channel = "sms"
)

// SendRequest represents a notification send request
type SendRequest struct {
	To       string                 `json:"to" validate:"required"`       // Recipient (email, phone, etc.)
	Channel  Channel                `json:"channel" validate:"required"`  // Notification channel
	Template string                 `json:"template"`                     // Template name (optional)
	Subject  string                 `json:"subject"`                      // Subject (for email)
	Body     string                 `json:"body"`                         // Direct body content (if no template)
	Data     map[string]interface{} `json:"data"`                         // Template data
	From     string                 `json:"from"`                         // Optional: override sender
}

// SendResponse represents the response from sending a notification
type SendResponse struct {
	MessageID  string `json:"message_id,omitempty"`
	Channel    string `json:"channel"`
	TenantID   int    `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	Source     string `json:"source"` // "tenant" or "global" - which credentials were used
}

// Adapter defines the interface for notification channels
// This interface is vendor-agnostic - works with any SMTP, WhatsApp, SMS provider
type Adapter interface {
	Send(ctx context.Context, req interface{}) (messageID string, err error)
	Name() string
}

// NotifyService is the tenant-aware notification service
// It resolves credentials per tenant and uses vendor-agnostic adapters
type NotifyService struct {
	config             *config.Config
	credentialResolver *CredentialResolver
	logger             zerolog.Logger
}

// NewNotifyService creates a new tenant-aware notification service
func NewNotifyService(cfg *config.Config, credentialResolver *CredentialResolver, logger zerolog.Logger) *NotifyService {
	return &NotifyService{
		config:             cfg,
		credentialResolver: credentialResolver,
		logger:             logger.With().Str("service", "notify").Logger(),
	}
}

// Send sends a notification through the specified channel using tenant-specific or global credentials
func (s *NotifyService) Send(ctx context.Context, req *SendRequest) (*SendResponse, error) {
	// Extract tenant from context
	tenant, ok := middleware.GetTenantFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant not found in context - ensure authentication middleware is applied")
	}

	s.logger.Info().
		Int("tenant_id", tenant.ID).
		Str("tenant_name", tenant.Name).
		Str("channel", string(req.Channel)).
		Str("to", maskRecipient(req.To)).
		Msg("Sending notification")

	// Validate request
	if err := s.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Send based on channel
	var messageID, source string
	var err error

	switch req.Channel {
	case ChannelEmail:
		messageID, source, err = s.sendEmail(ctx, tenant, req)
	case ChannelWhatsApp:
		messageID, source, err = s.sendWhatsApp(ctx, tenant, req)
	case ChannelSMS:
		messageID, source, err = s.sendSMS(ctx, tenant, req)
	default:
		return nil, fmt.Errorf("unsupported channel: %s", req.Channel)
	}

	if err != nil {
		s.logger.Error().
			Err(err).
			Int("tenant_id", tenant.ID).
			Str("channel", string(req.Channel)).
			Msg("Failed to send notification")
		return nil, err
	}

	s.logger.Info().
		Int("tenant_id", tenant.ID).
		Str("channel", string(req.Channel)).
		Str("message_id", messageID).
		Str("source", source).
		Msg("Notification sent successfully")

	return &SendResponse{
		MessageID:  messageID,
		Channel:    string(req.Channel),
		TenantID:   tenant.ID,
		TenantName: tenant.Name,
		Source:     source,
	}, nil
}

// sendEmail sends an email using tenant-specific or global SMTP credentials
// Vendor-agnostic: works with any SMTP server (Gmail, SendGrid, Mailgun, SES, self-hosted, etc.)
func (s *NotifyService) sendEmail(ctx context.Context, tenant *models.Tenant, req *SendRequest) (messageID, source string, err error) {
	// Resolve SMTP credentials (tenant-specific or global fallback)
	creds, err := s.credentialResolver.ResolveEmailCredentials(tenant)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve email credentials: %w", err)
	}

	s.logger.Debug().
		Int("tenant_id", tenant.ID).
		Str("source", creds.Source).
		Str("smtp_host", creds.Host).
		Msg("Using email credentials")

	// Create SMTP config for this specific request
	smtpConfig := &config.SMTPConfig{
		Host:     creds.Host,
		Port:     creds.Port,
		User:     creds.User,
		Password: creds.Password,
		From:     creds.From,
	}

	// Override From if specified in request
	if req.From != "" {
		smtpConfig.From = req.From
	}

	// Create email adapter with resolved credentials
	// This is vendor-agnostic - works with any SMTP provider
	emailAdapter := email.NewAdapter(smtpConfig, &s.config.Templates)

	// Convert SendRequest to email-specific request
	emailReq := map[string]interface{}{
		"to":       req.To,
		"subject":  req.Subject,
		"template": req.Template,
		"body":     req.Body,
		"data":     req.Data,
	}

	// Send email
	messageID, err = emailAdapter.Send(ctx, emailReq)
	if err != nil {
		return "", creds.Source, fmt.Errorf("failed to send email: %w", err)
	}

	return messageID, creds.Source, nil
}

// sendWhatsApp sends a WhatsApp message using tenant-specific or global credentials
// Vendor-agnostic: works with any WhatsApp Business API provider
func (s *NotifyService) sendWhatsApp(ctx context.Context, tenant *models.Tenant, req *SendRequest) (messageID, source string, err error) {
	// Resolve WhatsApp credentials (tenant-specific or global fallback)
	creds, err := s.credentialResolver.ResolveWhatsAppCredentials(tenant)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve WhatsApp credentials: %w", err)
	}

	s.logger.Debug().
		Int("tenant_id", tenant.ID).
		Str("source", creds.Source).
		Msg("Using WhatsApp credentials")

	// Create WhatsApp config for this specific request
	waConfig := &config.WhatsAppConfig{
		Token:   creds.Token,
		PhoneID: creds.PhoneID,
	}

	// Create WhatsApp adapter with resolved credentials
	// Vendor-agnostic - works with any WhatsApp Business API provider
	whatsappAdapter := whatsapp.NewAdapter(waConfig)

	// Convert SendRequest to WhatsApp-specific request
	waReq := map[string]interface{}{
		"to":       req.To,
		"template": req.Template,
		"body":     req.Body,
		"data":     req.Data,
	}

	// Send WhatsApp message
	messageID, err = whatsappAdapter.Send(ctx, waReq)
	if err != nil {
		return "", creds.Source, fmt.Errorf("failed to send WhatsApp message: %w", err)
	}

	return messageID, creds.Source, nil
}

// sendSMS sends an SMS using tenant-specific or global credentials
// Vendor-agnostic: works with any SMS provider (Twilio, Plivo, Vonage, etc.)
func (s *NotifyService) sendSMS(ctx context.Context, tenant *models.Tenant, req *SendRequest) (messageID, source string, err error) {
	// Resolve SMS credentials (tenant-specific or global fallback)
	creds, err := s.credentialResolver.ResolveSMSCredentials(tenant)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve SMS credentials: %w", err)
	}

	s.logger.Debug().
		Int("tenant_id", tenant.ID).
		Str("source", creds.Source).
		Str("provider", creds.Provider).
		Msg("Using SMS credentials")

	// TODO: Implement SMS adapter (vendor-agnostic)
	// Will work with any SMS provider: Twilio, Plivo, Vonage, AWS SNS, etc.
	return "", creds.Source, fmt.Errorf("SMS channel not yet implemented")
}

// validateRequest validates a send request
func (s *NotifyService) validateRequest(req *SendRequest) error {
	if req.To == "" {
		return fmt.Errorf("recipient (to) is required")
	}

	if req.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	// Either template or body must be provided
	if req.Template == "" && req.Body == "" {
		return fmt.Errorf("either template or body is required")
	}

	// Email-specific validation
	if req.Channel == ChannelEmail && req.Subject == "" && req.Template == "" {
		return fmt.Errorf("subject is required for email (unless using template)")
	}

	return nil
}

// maskRecipient masks sensitive recipient information in logs (PII protection)
func maskRecipient(recipient string) string {
	if len(recipient) <= 4 {
		return "***"
	}
	// Show first 2 and last 2 characters, mask the rest
	return recipient[:2] + "***" + recipient[len(recipient)-2:]
}
