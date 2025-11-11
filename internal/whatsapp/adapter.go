package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/huzaifabhutta/notify-core/internal/config"
)

const (
	// WhatsApp Cloud API version
	APIVersion = "v21.0"
	// Base URL for WhatsApp Cloud API
	BaseURL = "https://graph.facebook.com"
)

// Adapter handles WhatsApp notifications via Meta Cloud API
type Adapter struct {
	config     *config.WhatsAppConfig
	httpClient *http.Client
}

// NewAdapter creates a new WhatsApp adapter
func NewAdapter(cfg *config.WhatsAppConfig) *Adapter {
	return &Adapter{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the adapter name
func (a *Adapter) Name() string {
	return "whatsapp"
}

// SendRequest represents a WhatsApp send request
type SendRequest struct {
	To       string
	Template string
	Data     map[string]interface{}
}

// WhatsAppMessageRequest represents the WhatsApp API request structure
type WhatsAppMessageRequest struct {
	MessagingProduct string      `json:"messaging_product"`
	RecipientType    string      `json:"recipient_type"`
	To               string      `json:"to"`
	Type             string      `json:"type"`
	Template         *Template   `json:"template,omitempty"`
	Text             *TextObject `json:"text,omitempty"`
}

// Template represents a WhatsApp template message
type Template struct {
	Name       string      `json:"name"`
	Language   Language    `json:"language"`
	Components []Component `json:"components,omitempty"`
}

// Language represents the template language
type Language struct {
	Code string `json:"code"`
}

// Component represents a template component with parameters
type Component struct {
	Type       string      `json:"type"`
	Parameters []Parameter `json:"parameters"`
}

// Parameter represents a template parameter
type Parameter struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// TextObject represents a simple text message
type TextObject struct {
	PreviewURL bool   `json:"preview_url"`
	Body       string `json:"body"`
}

// WhatsAppResponse represents the WhatsApp API response
type WhatsAppResponse struct {
	MessagingProduct string    `json:"messaging_product"`
	Contacts         []Contact `json:"contacts"`
	Messages         []Message `json:"messages"`
}

// Contact represents a contact in the response
type Contact struct {
	Input string `json:"input"`
	WaID  string `json:"wa_id"`
}

// Message represents a message in the response
type Message struct {
	ID string `json:"id"`
}

// WhatsAppError represents an error response from WhatsApp API
type WhatsAppError struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Message      string `json:"message"`
	Type         string `json:"type"`
	Code         int    `json:"code"`
	ErrorSubcode int    `json:"error_subcode"`
	FBTraceID    string `json:"fbtrace_id"`
}

// Send sends a WhatsApp notification
func (a *Adapter) Send(ctx context.Context, req interface{}) error {
	// Extract send request using reflection (similar to email adapter)
	sendReq, err := extractSendRequest(req)
	if err != nil {
		return err
	}

	// Validate phone number format (should be E.164: +1234567890)
	if sendReq.To == "" {
		return fmt.Errorf("recipient phone number is required")
	}

	// Build WhatsApp API request
	waReq := a.buildTemplateMessage(sendReq)

	// Send via WhatsApp Cloud API
	messageID, err := a.sendMessage(ctx, waReq)
	if err != nil {
		return fmt.Errorf("failed to send WhatsApp message: %w", err)
	}

	// Log success (will be enhanced in Day 13)
	_ = messageID

	return nil
}

// buildTemplateMessage builds a template message request
func (a *Adapter) buildTemplateMessage(req SendRequest) *WhatsAppMessageRequest {
	// Build template components from data
	var components []Component
	if len(req.Data) > 0 {
		params := make([]Parameter, 0, len(req.Data))
		for _, value := range req.Data {
			params = append(params, Parameter{
				Type: "text",
				Text: fmt.Sprintf("%v", value),
			})
		}

		if len(params) > 0 {
			components = append(components, Component{
				Type:       "body",
				Parameters: params,
			})
		}
	}

	return &WhatsAppMessageRequest{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               req.To,
		Type:             "template",
		Template: &Template{
			Name: req.Template,
			Language: Language{
				Code: "en", // Default to English, could be configurable
			},
			Components: components,
		},
	}
}

// sendMessage sends the message via WhatsApp Cloud API
func (a *Adapter) sendMessage(ctx context.Context, msg *WhatsAppMessageRequest) (string, error) {
	// Build API URL
	url := fmt.Sprintf("%s/%s/%s/messages", BaseURL, APIVersion, a.config.PhoneID)

	// Marshal request body
	body, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.config.Token)

	// Send request
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var waErr WhatsAppError
		if err := json.Unmarshal(respBody, &waErr); err != nil {
			return "", fmt.Errorf("WhatsApp API error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return "", fmt.Errorf("WhatsApp API error: %s (code: %d)", waErr.Error.Message, waErr.Error.Code)
	}

	// Parse success response
	var waResp WhatsAppResponse
	if err := json.Unmarshal(respBody, &waResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(waResp.Messages) == 0 {
		return "", fmt.Errorf("no message ID in response")
	}

	return waResp.Messages[0].ID, nil
}

// extractSendRequest extracts fields from the notify.SendRequest
func extractSendRequest(req interface{}) (SendRequest, error) {
	// Try direct cast first
	if r, ok := req.(*SendRequest); ok {
		return *r, nil
	}
	if r, ok := req.(SendRequest); ok {
		return r, nil
	}

	// Extract from notify.SendRequest using type assertion
	type RequestLike interface {
		To, Template string
		Data         map[string]interface{}
	}

	if r, ok := req.(RequestLike); ok {
		return SendRequest{
			To:       r.To,
			Template: r.Template,
			Data:     r.Data,
		}, nil
	}

	// Try pointer version
	if ptr, ok := req.(interface {
		To, Template string
		Data         map[string]interface{}
	}); ok {
		return SendRequest{
			To:       ptr.To,
			Template: ptr.Template,
			Data:     ptr.Data,
		}, nil
	}

	return SendRequest{}, fmt.Errorf("invalid request type for WhatsApp adapter: got %T", req)
}
