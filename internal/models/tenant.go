package models

import (
	"time"
)

// Tenant represents a tenant in the multi-tenant system
type Tenant struct {
	ID   int    `json:"id"`
	Name string `json:"name"`

	// API Key for authentication
	APIKey string `json:"api_key,omitempty"`

	// Email configuration (SMTP)
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	SMTPUser     string `json:"smtp_user,omitempty"`
	SMTPPassword string `json:"smtp_password,omitempty"`
	SMTPFrom     string `json:"smtp_from,omitempty"`

	// WhatsApp configuration
	WAToken   string `json:"wa_token,omitempty"`
	WAPhoneID string `json:"wa_phone_id,omitempty"`

	// SMS configuration
	SMSProvider  string `json:"sms_provider,omitempty"`
	SMSAPIKey    string `json:"sms_api_key,omitempty"`
	SMSSenderID  string `json:"sms_sender_id,omitempty"`

	// Status
	Active bool `json:"active"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateTenantRequest represents the request to create a new tenant
type CreateTenantRequest struct {
	Name string `json:"name" validate:"required"`

	// Email configuration (optional - will use global if not provided)
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	SMTPUser     string `json:"smtp_user,omitempty"`
	SMTPPassword string `json:"smtp_password,omitempty"`
	SMTPFrom     string `json:"smtp_from,omitempty"`

	// WhatsApp configuration (optional)
	WAToken   string `json:"wa_token,omitempty"`
	WAPhoneID string `json:"wa_phone_id,omitempty"`

	// SMS configuration (optional)
	SMSProvider  string `json:"sms_provider,omitempty"`
	SMSAPIKey    string `json:"sms_api_key,omitempty"`
	SMSSenderID  string `json:"sms_sender_id,omitempty"`
}

// UpdateTenantRequest represents the request to update a tenant
type UpdateTenantRequest struct {
	Name *string `json:"name,omitempty"`

	// Email configuration
	SMTPHost     *string `json:"smtp_host,omitempty"`
	SMTPPort     *int    `json:"smtp_port,omitempty"`
	SMTPUser     *string `json:"smtp_user,omitempty"`
	SMTPPassword *string `json:"smtp_password,omitempty"`
	SMTPFrom     *string `json:"smtp_from,omitempty"`

	// WhatsApp configuration
	WAToken   *string `json:"wa_token,omitempty"`
	WAPhoneID *string `json:"wa_phone_id,omitempty"`

	// SMS configuration
	SMSProvider  *string `json:"sms_provider,omitempty"`
	SMSAPIKey    *string `json:"sms_api_key,omitempty"`
	SMSSenderID  *string `json:"sms_sender_id,omitempty"`

	// Status
	Active *bool `json:"active,omitempty"`
}

// HasEmailConfig checks if tenant has email configuration
func (t *Tenant) HasEmailConfig() bool {
	return t.SMTPHost != "" && t.SMTPUser != "" && t.SMTPPassword != ""
}

// HasWhatsAppConfig checks if tenant has WhatsApp configuration
func (t *Tenant) HasWhatsAppConfig() bool {
	return t.WAToken != "" && t.WAPhoneID != ""
}

// HasSMSConfig checks if tenant has SMS configuration
func (t *Tenant) HasSMSConfig() bool {
	return t.SMSProvider != "" && t.SMSAPIKey != ""
}
