package notify

import (
	"context"
	"fmt"

	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/email"
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
	To       string                 `json:"to"`       // Recipient (email, phone, etc.)
	Channel  Channel                `json:"channel"`  // Notification channel
	Template string                 `json:"template"` // Template name
	Subject  string                 `json:"subject"`  // Subject (for email)
	Data     map[string]interface{} `json:"data"`     // Template data
	From     string                 `json:"from"`     // Optional: override sender
}

// Adapter defines the interface for notification channels
type Adapter interface {
	Send(ctx context.Context, req *SendRequest) error
	Name() string
}

// Service is the main notification service
type Service struct {
	config   *config.Config
	adapters map[Channel]Adapter
}

// NewService creates a new notification service
func NewService(cfg *config.Config) *Service {
	s := &Service{
		config:   cfg,
		adapters: make(map[Channel]Adapter),
	}

	// Register email adapter
	emailAdapter := email.NewAdapter(&cfg.SMTP, &cfg.Templates)
	s.RegisterAdapter(ChannelEmail, emailAdapter)

	// TODO: Register WhatsApp adapter
	// TODO: Register SMS adapter

	return s
}

// RegisterAdapter registers a notification channel adapter
func (s *Service) RegisterAdapter(channel Channel, adapter Adapter) {
	s.adapters[channel] = adapter
}

// Send sends a notification through the specified channel
func (s *Service) Send(ctx context.Context, req *SendRequest) error {
	// Validate request
	if err := s.validateRequest(req); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	// Get adapter for the channel
	adapter, ok := s.adapters[req.Channel]
	if !ok {
		return fmt.Errorf("unsupported channel: %s", req.Channel)
	}

	// Send notification - convert request based on channel type
	var adapterReq interface{}
	switch req.Channel {
	case ChannelEmail:
		// Import email package type locally to avoid import cycle
		adapterReq = &struct {
			To       string
			Channel  interface{}
			Template string
			Subject  string
			Data     map[string]interface{}
			From     string
		}{
			To:       req.To,
			Channel:  req.Channel,
			Template: req.Template,
			Subject:  req.Subject,
			Data:     req.Data,
			From:     req.From,
		}
	default:
		adapterReq = req
	}

	if err := adapter.Send(ctx, adapterReq); err != nil {
		return fmt.Errorf("failed to send via %s: %w", adapter.Name(), err)
	}

	return nil
}

// validateRequest validates the send request
func (s *Service) validateRequest(req *SendRequest) error {
	if req.To == "" {
		return fmt.Errorf("recipient (to) is required")
	}

	if req.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if req.Template == "" {
		return fmt.Errorf("template is required")
	}

	return nil
}

// GetSupportedChannels returns a list of supported channels
func (s *Service) GetSupportedChannels() []string {
	channels := make([]string, 0, len(s.adapters))
	for channel := range s.adapters {
		channels = append(channels, string(channel))
	}
	return channels
}
