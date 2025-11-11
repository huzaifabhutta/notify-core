package services

import (
	"context"
	"fmt"

	"github.com/huzaifabhutta/notify-core/internal/adapters"
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/email/ses"   // Register SES adapter
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/email/smtp"  // Register SMTP adapter
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/sms/sns"     // Register SNS adapter
	_ "github.com/huzaifabhutta/notify-core/internal/adapters/whatsapp/cloud" // Register WhatsApp adapter
	sesAdapter "github.com/huzaifabhutta/notify-core/internal/adapters/email/ses"
	smtpAdapter "github.com/huzaifabhutta/notify-core/internal/adapters/email/smtp"
	snsAdapter "github.com/huzaifabhutta/notify-core/internal/adapters/sms/sns"
	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/models"
	"github.com/huzaifabhutta/notify-core/internal/tenantctx"
	"github.com/rs/zerolog"
)

// NotifyServiceV2 is the registry-based notification service
// It uses the adapter registry for flexible adapter selection
type NotifyServiceV2 struct {
	config             *config.Config
	credentialResolver *CredentialResolver
	logger             zerolog.Logger
}

// NewNotifyServiceV2 creates a new registry-based notification service
func NewNotifyServiceV2(cfg *config.Config, credentialResolver *CredentialResolver, logger zerolog.Logger) *NotifyServiceV2 {
	return &NotifyServiceV2{
		config:             cfg,
		credentialResolver: credentialResolver,
		logger:             logger.With().Str("service", "notify-v2").Logger(),
	}
}

// Send sends a notification through the specified channel using registry-based adapters
func (s *NotifyServiceV2) Send(ctx context.Context, req *SendRequest) (*SendResponse, error) {
	// Extract tenant from context
	tenant, ok := tenantctx.GetTenant(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant not found in context - ensure authentication middleware is applied")
	}

	s.logger.Info().
		Int("tenant_id", tenant.ID).
		Str("tenant_name", tenant.Name).
		Str("channel", string(req.Channel)).
		Str("to", maskRecipient(req.To)).
		Msg("Sending notification via registry")

	// Validate request
	if err := s.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Send based on channel
	var messageID, source string
	var err error

	switch req.Channel {
	case ChannelEmail:
		messageID, source, err = s.sendEmailV2(ctx, tenant, req)
	case ChannelWhatsApp:
		messageID, source, err = s.sendWhatsAppV2(ctx, tenant, req)
	case ChannelSMS:
		messageID, source, err = s.sendSMSV2(ctx, tenant, req)
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

// sendEmailV2 sends an email using registry-based adapter selection
func (s *NotifyServiceV2) sendEmailV2(ctx context.Context, tenant *models.Tenant, req *SendRequest) (messageID, source string, err error) {
	// Resolve email credentials
	creds, err := s.credentialResolver.ResolveEmailCredentials(tenant)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve email credentials: %w", err)
	}

	// Determine which adapter to use (from config or default to SMTP)
	adapterName := s.config.Adapters.Email.Default
	if adapterName == "" {
		adapterName = "smtp" // Default to SMTP for backward compatibility
	}

	s.logger.Debug().
		Int("tenant_id", tenant.ID).
		Str("adapter", adapterName).
		Str("source", creds.Source).
		Msg("Using email adapter")

	// Create adapter config based on adapter type
	var adapterConfig interface{}

	switch adapterName {
	case "smtp":
		// SMTP adapter configuration
		adapterConfig = &smtpAdapter.Config{
			SMTPConfig: &config.SMTPConfig{
				Host:     creds.Host,
				Port:     creds.Port,
				User:     creds.User,
				Password: creds.Password,
				From:     creds.From,
			},
			TemplateConfig: &s.config.Templates,
		}

	case "ses":
		// AWS SES adapter configuration
		adapterConfig = &sesAdapter.Config{
			Region:           s.config.Adapters.Email.SES.Region,
			FromEmail:        creds.From,
			ConfigurationSet: s.config.Adapters.Email.SES.ConfigurationSet,
			AccessKeyID:      s.config.Adapters.Email.SES.AccessKeyID,
			SecretAccessKey:  s.config.Adapters.Email.SES.SecretAccessKey,
			RoleARN:          s.config.Adapters.Email.SES.RoleARN,
		}

	default:
		return "", "", fmt.Errorf("unsupported email adapter: %s", adapterName)
	}

	// Get adapter from registry
	adapter, err := adapters.Get(adapterName, adapterConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to get email adapter: %w", err)
	}

	// Build request
	emailReq := map[string]interface{}{
		"to":       req.To,
		"from":     creds.From,
		"subject":  req.Subject,
		"template": req.Template,
		"body":     req.Body,
		"data":     req.Data,
	}

	// Override from if specified in request
	if req.From != "" {
		emailReq["from"] = req.From
	}

	// Send email
	messageID, err = adapter.Send(ctx, emailReq)
	if err != nil {
		return "", creds.Source, fmt.Errorf("failed to send email: %w", err)
	}

	return messageID, creds.Source, nil
}

// sendWhatsAppV2 sends a WhatsApp message using registry-based adapter selection
func (s *NotifyServiceV2) sendWhatsAppV2(ctx context.Context, tenant *models.Tenant, req *SendRequest) (messageID, source string, err error) {
	// Resolve WhatsApp credentials
	creds, err := s.credentialResolver.ResolveWhatsAppCredentials(tenant)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve WhatsApp credentials: %w", err)
	}

	// Determine which adapter to use
	adapterName := s.config.Adapters.WhatsApp.Default
	if adapterName == "" {
		adapterName = "whatsapp-cloud" // Default
	}

	s.logger.Debug().
		Int("tenant_id", tenant.ID).
		Str("adapter", adapterName).
		Str("source", creds.Source).
		Msg("Using WhatsApp adapter")

	// Create adapter config
	adapterConfig := &config.WhatsAppConfig{
		Token:      creds.Token,
		PhoneID:    creds.PhoneID,
		BaseURL:    creds.BaseURL,
		APIVersion: creds.APIVersion,
	}

	// Get adapter from registry
	adapter, err := adapters.Get(adapterName, adapterConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to get WhatsApp adapter: %w", err)
	}

	// Build request
	waReq := map[string]interface{}{
		"to":       req.To,
		"template": req.Template,
		"data":     req.Data,
	}

	// Send WhatsApp message
	messageID, err = adapter.Send(ctx, waReq)
	if err != nil {
		return "", creds.Source, fmt.Errorf("failed to send WhatsApp message: %w", err)
	}

	return messageID, creds.Source, nil
}

// sendSMSV2 sends an SMS using registry-based adapter selection
func (s *NotifyServiceV2) sendSMSV2(ctx context.Context, tenant *models.Tenant, req *SendRequest) (messageID, source string, err error) {
	// Determine which adapter to use
	adapterName := s.config.Adapters.SMS.Default
	if adapterName == "" {
		adapterName = "sns" // Default to AWS SNS
	}

	s.logger.Debug().
		Int("tenant_id", tenant.ID).
		Str("adapter", adapterName).
		Msg("Using SMS adapter")

	// Create adapter config based on adapter type
	var adapterConfig interface{}

	switch adapterName {
	case "sns":
		// AWS SNS adapter configuration
		adapterConfig = &snsAdapter.Config{
			Region:          s.config.Adapters.SMS.SNS.Region,
			SenderID:        s.config.Adapters.SMS.SNS.SenderID,
			SMSType:         s.config.Adapters.SMS.SNS.SMSType,
			AccessKeyID:     s.config.Adapters.SMS.SNS.AccessKeyID,
			SecretAccessKey: s.config.Adapters.SMS.SNS.SecretAccessKey,
		}

	default:
		return "", "", fmt.Errorf("unsupported SMS adapter: %s", adapterName)
	}

	// Get adapter from registry
	adapter, err := adapters.Get(adapterName, adapterConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to get SMS adapter: %w", err)
	}

	// Build request
	smsReq := map[string]interface{}{
		"to":   req.To,
		"body": req.Body,
	}

	// If body is empty, try to get it from Data
	if req.Body == "" && req.Data != nil {
		if body, ok := req.Data["body"]; ok {
			smsReq["body"] = body
		}
	}

	// Send SMS
	messageID, err = adapter.Send(ctx, smsReq)
	if err != nil {
		return "", "global", fmt.Errorf("failed to send SMS: %w", err)
	}

	return messageID, "global", nil
}

// validateRequest validates the send request
func (s *NotifyServiceV2) validateRequest(req *SendRequest) error {
	if req.To == "" {
		return fmt.Errorf("recipient is required")
	}

	if req.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	// Channel-specific validation
	switch req.Channel {
	case ChannelEmail:
		if req.Subject == "" && req.Template == "" {
			return fmt.Errorf("either subject or template is required for email")
		}
	case ChannelWhatsApp:
		if req.Template == "" {
			return fmt.Errorf("template is required for WhatsApp")
		}
	case ChannelSMS:
		if req.Body == "" && (req.Data == nil || req.Data["body"] == nil) {
			return fmt.Errorf("body is required for SMS")
		}
	}

	return nil
}
