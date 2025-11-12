package services

import (
	"context"
	"fmt"

	"github.com/huzaifabhutta/notify-core/internal/channels"
	"github.com/huzaifabhutta/notify-core/internal/channels/email"
	"github.com/huzaifabhutta/notify-core/internal/channels/sms"
	"github.com/huzaifabhutta/notify-core/internal/channels/whatsapp"
	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/models"
	"github.com/huzaifabhutta/notify-core/internal/tenantctx"
	"github.com/rs/zerolog"
)

// NotifyServiceV3 uses the channel registry for extensible notifications
// This is the next-generation service that supports pluggable channels
type NotifyServiceV3 struct {
	config             *config.Config
	credentialResolver *CredentialResolver
	logger             zerolog.Logger
}

// NewNotifyServiceV3 creates a new channel registry-based notification service
func NewNotifyServiceV3(cfg *config.Config, credentialResolver *CredentialResolver, logger zerolog.Logger) *NotifyServiceV3 {
	return &NotifyServiceV3{
		config:             cfg,
		credentialResolver: credentialResolver,
		logger:             logger.With().Str("service", "notify-v3").Logger(),
	}
}

// Send sends a notification through the channel registry
// This method uses dynamic channel lookup instead of hardcoded switch statements
func (s *NotifyServiceV3) Send(ctx context.Context, req *SendRequest) (*SendResponse, error) {
	// Extract tenant from context
	tenant, ok := tenantctx.GetTenant(ctx)
	if !ok {
		s.logger.Warn().Msg("No tenant found in context, using global configuration")
	}

	// Validate request
	if req.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}
	if req.To == "" {
		return nil, fmt.Errorf("recipient (to) is required")
	}

	// Log the send attempt
	s.logger.Info().
		Str("channel", string(req.Channel)).
		Str("tenant", getTenantName(tenant)).
		Msg("Processing notification request via channel registry")

	// Get channel configuration based on channel type
	channelConfig, err := s.getChannelConfig(string(req.Channel), tenant)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("channel", string(req.Channel)).
			Msg("Failed to get channel configuration")
		return nil, fmt.Errorf("failed to get channel configuration: %w", err)
	}

	// Get channel from registry
	channel, err := channels.Get(string(req.Channel), channelConfig)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("channel", string(req.Channel)).
			Msg("Failed to get channel from registry")
		return nil, fmt.Errorf("failed to get channel '%s': %w", req.Channel, err)
	}

	// Convert service request to channel request
	channelReq := &channels.SendRequest{
		To:       req.To,
		From:     req.From,
		Subject:  req.Subject,
		Body:     req.Body,
		Template: req.Template,
		Data:     req.Data,
	}

	// Send via channel
	channelResp, err := channel.Send(ctx, channelReq)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("channel", string(req.Channel)).
			Str("tenant", getTenantName(tenant)).
			Msg("Failed to send via channel")
		return nil, fmt.Errorf("channel send failed: %w", err)
	}

	s.logger.Info().
		Str("channel", string(req.Channel)).
		Str("message_id", channelResp.MessageID).
		Str("adapter", channelResp.Adapter).
		Str("tenant", getTenantName(tenant)).
		Msg("Notification sent successfully via channel registry")

	// Build response
	resp := &SendResponse{
		MessageID: channelResp.MessageID,
		Channel:   string(req.Channel),
	}

	// Add source from metadata if available
	if source, ok := channelResp.Metadata["source"]; ok {
		resp.Source = source
	}

	// Add tenant info if available
	if tenant != nil {
		resp.TenantID = tenant.ID
		resp.TenantName = tenant.Name
	}

	return resp, nil
}

// getChannelConfig returns the configuration for a specific channel
// This method creates the appropriate config structure for each channel type
func (s *NotifyServiceV3) getChannelConfig(channelName string, tenant interface{}) (interface{}, error) {
	switch channelName {
	case "email":
		return s.getEmailChannelConfig(tenant)
	case "sms":
		return s.getSMSChannelConfig(tenant)
	case "whatsapp":
		return s.getWhatsAppChannelConfig(tenant)
	default:
		return nil, fmt.Errorf("unsupported channel: %s", channelName)
	}
}

// getEmailChannelConfig creates email channel configuration
func (s *NotifyServiceV3) getEmailChannelConfig(tenant interface{}) (*email.Config, error) {
	cfg := &email.Config{
		DefaultAdapter: s.config.Adapters.Email.Default,
		SMTP:           &s.config.SMTP,
		Logger:         &s.logger,
	}

	// Add SES configuration if available
	if s.config.Adapters.Email.SES.Region != "" {
		cfg.SES = &email.SESConfig{
			Region:           s.config.Adapters.Email.SES.Region,
			FromEmail:        s.config.Adapters.Email.SES.FromEmail,
			ConfigurationSet: s.config.Adapters.Email.SES.ConfigurationSet,
			AccessKeyID:      s.config.Adapters.Email.SES.AccessKeyID,
			SecretAccessKey:  s.config.Adapters.Email.SES.SecretAccessKey,
			RoleARN:          s.config.Adapters.Email.SES.RoleARN,
		}
	}

	// Add credential resolver if available
	if s.credentialResolver != nil {
		cfg.Credentials = &emailCredentialResolver{
			resolver: s.credentialResolver,
			tenant:   tenant,
		}
	}

	return cfg, nil
}

// getSMSChannelConfig creates SMS channel configuration
func (s *NotifyServiceV3) getSMSChannelConfig(tenant interface{}) (*sms.Config, error) {
	cfg := &sms.Config{
		DefaultAdapter: s.config.Adapters.SMS.Default,
		Logger:         &s.logger,
	}

	// Add SNS configuration if available
	if s.config.Adapters.SMS.SNS.Region != "" {
		cfg.SNS = &sms.SNSConfig{
			Region:          s.config.Adapters.SMS.SNS.Region,
			SenderID:        s.config.Adapters.SMS.SNS.SenderID,
			SMSType:         s.config.Adapters.SMS.SNS.SMSType,
			AccessKeyID:     s.config.Adapters.SMS.SNS.AccessKeyID,
			SecretAccessKey: s.config.Adapters.SMS.SNS.SecretAccessKey,
		}
	}

	// Add credential resolver if available
	if s.credentialResolver != nil {
		cfg.Credentials = &smsCredentialResolver{
			resolver: s.credentialResolver,
			tenant:   tenant,
		}
	}

	return cfg, nil
}

// getWhatsAppChannelConfig creates WhatsApp channel configuration
func (s *NotifyServiceV3) getWhatsAppChannelConfig(tenant interface{}) (*whatsapp.Config, error) {
	cfg := &whatsapp.Config{
		DefaultAdapter: "whatsapp-cloud",
		Cloud:          &s.config.WhatsApp,
		Logger:         &s.logger,
	}

	// Add credential resolver if available
	if s.credentialResolver != nil {
		cfg.Credentials = &whatsappCredentialResolver{
			resolver: s.credentialResolver,
			tenant:   tenant,
		}
	}

	return cfg, nil
}

// Helper functions

func getTenantName(tenant interface{}) string {
	if tenant == nil {
		return "global"
	}
	// Proper type assertion to get tenant name
	if t, ok := tenant.(*models.Tenant); ok && t != nil {
		return t.Name
	}
	return "unknown"
}

// Credential resolver adapters
// These adapt the service's CredentialResolver to the channel's CredentialResolver interface

type emailCredentialResolver struct {
	resolver *CredentialResolver
	tenant   interface{}
}

func (r *emailCredentialResolver) ResolveEmail(ctx context.Context) (*email.EmailCredentials, error) {
	// Type assert tenant to proper type
	var tenant *models.Tenant
	if r.tenant != nil {
		if t, ok := r.tenant.(*models.Tenant); ok {
			tenant = t
		}
	}

	// Resolve credentials using service's credential resolver
	creds, err := r.resolver.ResolveEmailCredentials(tenant)
	if err != nil {
		return nil, err
	}

	// Convert from services.SMTPCredentials to email.EmailCredentials
	return &email.EmailCredentials{
		Host:     creds.Host,
		Port:     creds.Port,
		User:     creds.User,
		Password: creds.Password,
		From:     creds.From,
		Source:   creds.Source,
	}, nil
}

type smsCredentialResolver struct {
	resolver *CredentialResolver
	tenant   interface{}
}

func (r *smsCredentialResolver) ResolveSMS(ctx context.Context) (*sms.SMSCredentials, error) {
	// Type assert tenant to proper type
	var tenant *models.Tenant
	if r.tenant != nil {
		if t, ok := r.tenant.(*models.Tenant); ok {
			tenant = t
		}
	}

	// Resolve credentials using service's credential resolver
	creds, err := r.resolver.ResolveSMSCredentials(tenant)
	if err != nil {
		return nil, err
	}

	// Convert from services.SMSCredentials to sms.SMSCredentials
	// Note: The SMS channel credentials only include Provider and Source
	// The actual API keys and sender IDs come from the channel config
	return &sms.SMSCredentials{
		Provider: creds.Provider,
		Source:   creds.Source,
	}, nil
}

type whatsappCredentialResolver struct {
	resolver *CredentialResolver
	tenant   interface{}
}

func (r *whatsappCredentialResolver) ResolveWhatsApp(ctx context.Context) (*whatsapp.WhatsAppCredentials, error) {
	// Type assert tenant to proper type
	var tenant *models.Tenant
	if r.tenant != nil {
		if t, ok := r.tenant.(*models.Tenant); ok {
			tenant = t
		}
	}

	// Resolve credentials using service's credential resolver
	creds, err := r.resolver.ResolveWhatsAppCredentials(tenant)
	if err != nil {
		return nil, err
	}

	// Convert from services.WhatsAppCredentials to whatsapp.WhatsAppCredentials
	return &whatsapp.WhatsAppCredentials{
		Token:   creds.Token,
		PhoneID: creds.PhoneID,
		Source:  creds.Source,
	}, nil
}
