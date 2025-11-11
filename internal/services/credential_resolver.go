package services

import (
	"fmt"

	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/models"
)

// CredentialResolver resolves credentials for a tenant
// Falls back to global config if tenant doesn't have channel-specific credentials
// This keeps the service vendor-agnostic - works with any SMTP/WhatsApp/SMS provider
type CredentialResolver struct {
	globalConfig *config.Config
}

// NewCredentialResolver creates a new credential resolver
func NewCredentialResolver(globalConfig *config.Config) *CredentialResolver {
	return &CredentialResolver{
		globalConfig: globalConfig,
	}
}

// SMTPCredentials holds SMTP configuration for sending emails
// Vendor-agnostic: works with any SMTP server (Gmail, SendGrid, Mailgun, SES, etc.)
type SMTPCredentials struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	Source   string // "tenant" or "global" for debugging
}

// WhatsAppCredentials holds WhatsApp API configuration
// Vendor-agnostic: works with any WhatsApp Business API provider
type WhatsAppCredentials struct {
	Token    string
	PhoneID  string
	Source   string // "tenant" or "global"
}

// SMSCredentials holds SMS API configuration
// Vendor-agnostic: works with any SMS provider (Twilio, Plivo, Vonage, etc.)
type SMSCredentials struct {
	Provider string // Provider name (for adapter selection)
	APIKey   string
	From     string
	Source   string // "tenant" or "global"
}

// ResolveEmailCredentials returns SMTP credentials for a tenant
// Priority: 1) Tenant-specific, 2) Global fallback
func (r *CredentialResolver) ResolveEmailCredentials(tenant *models.Tenant) (*SMTPCredentials, error) {
	// Try tenant-specific credentials first
	if tenant != nil && tenant.HasEmailConfig() {
		return &SMTPCredentials{
			Host:     tenant.SMTPHost,
			Port:     tenant.SMTPPort,
			User:     tenant.SMTPUser,
			Password: tenant.SMTPPassword,
			From:     tenant.SMTPFrom,
			Source:   "tenant",
		}, nil
	}

	// Fallback to global credentials
	if r.globalConfig.SMTP.Host == "" {
		return nil, fmt.Errorf("no email credentials available: tenant has no SMTP config and no global SMTP configured")
	}

	return &SMTPCredentials{
		Host:     r.globalConfig.SMTP.Host,
		Port:     r.globalConfig.SMTP.Port,
		User:     r.globalConfig.SMTP.User,
		Password: r.globalConfig.SMTP.Password,
		From:     r.globalConfig.SMTP.From,
		Source:   "global",
	}, nil
}

// ResolveWhatsAppCredentials returns WhatsApp credentials for a tenant
// Priority: 1) Tenant-specific, 2) Global fallback
func (r *CredentialResolver) ResolveWhatsAppCredentials(tenant *models.Tenant) (*WhatsAppCredentials, error) {
	// Try tenant-specific credentials first
	if tenant != nil && tenant.HasWhatsAppConfig() {
		return &WhatsAppCredentials{
			Token:   tenant.WAToken,
			PhoneID: tenant.WAPhoneID,
			Source:  "tenant",
		}, nil
	}

	// Fallback to global credentials
	if r.globalConfig.WhatsApp.Token == "" {
		return nil, fmt.Errorf("no WhatsApp credentials available: tenant has no WhatsApp config and no global WhatsApp configured")
	}

	return &WhatsAppCredentials{
		Token:   r.globalConfig.WhatsApp.Token,
		PhoneID: r.globalConfig.WhatsApp.PhoneID,
		Source:  "global",
	}, nil
}

// ResolveSMSCredentials returns SMS credentials for a tenant
// Priority: 1) Tenant-specific, 2) Global fallback
func (r *CredentialResolver) ResolveSMSCredentials(tenant *models.Tenant) (*SMSCredentials, error) {
	// Try tenant-specific credentials first
	if tenant != nil && tenant.HasSMSConfig() {
		return &SMSCredentials{
			Provider: tenant.SMSProvider,
			APIKey:   tenant.SMSAPIKey,
			From:     tenant.SMSSenderID,
			Source:   "tenant",
		}, nil
	}

	// Fallback to global credentials
	if r.globalConfig.SMS.APIKey == "" {
		return nil, fmt.Errorf("no SMS credentials available: tenant has no SMS config and no global SMS configured")
	}

	return &SMSCredentials{
		Provider: r.globalConfig.SMS.Provider,
		APIKey:   r.globalConfig.SMS.APIKey,
		From:     r.globalConfig.SMS.From,
		Source:   "global",
	}, nil
}

// HasEmailCredentials checks if email credentials are available for tenant
func (r *CredentialResolver) HasEmailCredentials(tenant *models.Tenant) bool {
	if tenant != nil && tenant.HasEmailConfig() {
		return true
	}
	return r.globalConfig.SMTP.Host != ""
}

// HasWhatsAppCredentials checks if WhatsApp credentials are available for tenant
func (r *CredentialResolver) HasWhatsAppCredentials(tenant *models.Tenant) bool {
	if tenant != nil && tenant.HasWhatsAppConfig() {
		return true
	}
	return r.globalConfig.WhatsApp.Token != ""
}

// HasSMSCredentials checks if SMS credentials are available for tenant
func (r *CredentialResolver) HasSMSCredentials(tenant *models.Tenant) bool {
	if tenant != nil && tenant.HasSMSConfig() {
		return true
	}
	return r.globalConfig.SMS.APIKey != ""
}
