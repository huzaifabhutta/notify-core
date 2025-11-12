package channels

import (
	"context"
)

// Type represents the category of channel
type Type string

const (
	TypeMessaging Type = "messaging" // Email, SMS, WhatsApp
	TypePush      Type = "push"      // Push notifications (FCM, APNs)
	TypeVoice     Type = "voice"     // Voice calls (Twilio Voice)
	TypeWebhook   Type = "webhook"   // Webhook notifications
	TypeChatApp   Type = "chat"      // Slack, Discord, Telegram
)

// Feature represents a channel capability
type Feature string

const (
	FeatureAttachments     Feature = "attachments"      // Supports file attachments
	FeatureRichText        Feature = "rich_text"        // Supports HTML/Markdown
	FeatureTemplates       Feature = "templates"        // Supports templates
	FeatureScheduling      Feature = "scheduling"       // Supports scheduled delivery
	FeatureDeliveryReceipt Feature = "delivery_receipt" // Supports delivery receipts
	FeatureBatching        Feature = "batching"         // Supports batch sending
	FeatureRetry           Feature = "retry"            // Supports automatic retry
)

// Channel represents a notification channel (email, sms, whatsapp, push, etc.)
// Each channel knows how to validate requests and send notifications through adapters
type Channel interface {
	// Name returns the channel identifier (e.g., "email", "sms", "push")
	Name() string

	// Type returns the channel type for categorization
	Type() Type

	// Validate validates a send request for this channel
	Validate(ctx context.Context, req *SendRequest) error

	// Send sends a notification through this channel
	// Returns the message ID from the underlying adapter
	Send(ctx context.Context, req *SendRequest) (*SendResponse, error)

	// GetAdapters returns the list of adapters this channel can use
	GetAdapters() []string

	// SupportsFeature checks if the channel supports a specific feature
	SupportsFeature(feature Feature) bool
}

// SendRequest represents a channel-agnostic send request
type SendRequest struct {
	To          string                 `json:"to"`                    // Recipient (email, phone, device token, etc.)
	From        string                 `json:"from,omitempty"`        // Sender (optional, channel-specific)
	Subject     string                 `json:"subject,omitempty"`     // Subject/Title (for email, push, etc.)
	Body        string                 `json:"body"`                  // Message body (required)
	Template    string                 `json:"template,omitempty"`    // Template name (optional)
	Data        map[string]interface{} `json:"data,omitempty"`        // Template data
	Attachments []Attachment           `json:"attachments,omitempty"` // Attachments (if supported)
	Metadata    map[string]string      `json:"metadata,omitempty"`    // Channel-specific metadata
}

// SendResponse represents a send response
type SendResponse struct {
	MessageID string            `json:"message_id"` // Message ID from adapter
	Channel   string            `json:"channel"`    // Channel name
	Adapter   string            `json:"adapter"`    // Adapter used
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
	Filename    string `json:"filename"`     // File name
	ContentType string `json:"content_type"` // MIME type
	Content     []byte `json:"-"`            // File content (omit from JSON)
	URL         string `json:"url,omitempty"` // Alternative: URL to attachment
}

// Metadata holds channel metadata
type Metadata struct {
	Name        string    `json:"name"`        // Channel name (e.g., "email", "push")
	Type        Type      `json:"type"`        // Channel type
	Description string    `json:"description"` // Human-readable description
	Version     string    `json:"version"`     // Channel implementation version
	Adapters    []string  `json:"adapters"`    // Supported adapters
	Features    []Feature `json:"features"`    // Supported features
}
