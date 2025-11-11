package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/huzaifabhutta/notify-core/internal/adapters"
	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/logger"
)

const (
	// Default WhatsApp Cloud API version (used if not configured)
	DefaultAPIVersion = "v21.0"
	// Default Base URL for WhatsApp Cloud API (used if not configured)
	DefaultBaseURL = "https://graph.facebook.com"
)

// Adapter handles WhatsApp notifications via any WhatsApp Business API provider
// Vendor-agnostic: works with Facebook Cloud API, 360dialog, Twilio, on-premises, etc.
type Adapter struct {
	config     *config.WhatsAppConfig
	httpClient *http.Client
	baseURL    string // Configurable base URL (defaults to Facebook Cloud API)
	apiVersion string // Configurable API version (defaults to v21.0)
}

// NewAdapter creates a new WhatsApp adapter
// Vendor-agnostic: uses configured BaseURL or defaults to Facebook Cloud API
func NewAdapter(cfg *config.WhatsAppConfig) *Adapter {
	// Use configured BaseURL or fallback to Facebook Cloud API
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	// Use configured API version or fallback to default
	apiVersion := cfg.APIVersion
	if apiVersion == "" {
		apiVersion = DefaultAPIVersion
	}

	return &Adapter{
		config:     cfg,
		baseURL:    baseURL,
		apiVersion: apiVersion,
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
// Returns the message ID from WhatsApp API and error
func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
	start := time.Now()
	log := logger.FromContext(ctx)

	// Extract fields using common adapter logic
	baseReq, err := adapters.ExtractBaseRequest(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to extract WhatsApp request")
		return "", fmt.Errorf("whatsapp adapter: %w", err)
	}

	maskedTo := logger.MaskPhone(baseReq.To)
	log.Debug().
		Str("channel", "whatsapp").
		Str("to", maskedTo).
		Str("template", baseReq.Template).
		Msg("Processing WhatsApp notification")

	// Build WhatsApp API request
	waReq := a.buildTemplateMessage(baseReq)

	// Send via WhatsApp Cloud API
	messageID, err := a.sendMessage(ctx, waReq)
	if err != nil {
		log.Error().
			Err(err).
			Str("to", maskedTo).
			Str("template", baseReq.Template).
			Dur("duration", time.Since(start)).
			Msg("Failed to send WhatsApp message")
		return "", fmt.Errorf("failed to send WhatsApp message: %w", err)
	}

	log.Info().
		Str("channel", "whatsapp").
		Str("to", maskedTo).
		Str("template", baseReq.Template).
		Str("message_id", messageID).
		Dur("duration", time.Since(start)).
		Msg("WhatsApp message sent successfully")

	return messageID, nil
}

// buildTemplateMessage builds a template message request
func (a *Adapter) buildTemplateMessage(req adapters.BaseRequest) *WhatsAppMessageRequest {
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

// sendMessage sends the message via WhatsApp Business API
// Vendor-agnostic: uses configured base URL and API version
func (a *Adapter) sendMessage(ctx context.Context, msg *WhatsAppMessageRequest) (string, error) {
	log := logger.FromContext(ctx)

	// Build API URL using configured base URL and version
	url := fmt.Sprintf("%s/%s/%s/messages", a.baseURL, a.apiVersion, a.config.PhoneID)

	// Marshal request body
	body, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	log.Debug().
		Str("api_url", url).
		Str("base_url", a.baseURL).
		Str("api_version", a.apiVersion).
		Msg("Sending request to WhatsApp API")

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
		log.Error().Err(err).Msg("HTTP request to WhatsApp API failed")
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
		log.Warn().
			Int("status_code", resp.StatusCode).
			Str("response", string(respBody)).
			Msg("WhatsApp API returned non-200 status")

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

	log.Debug().
		Str("message_id", waResp.Messages[0].ID).
		Msg("WhatsApp API request successful")

	return waResp.Messages[0].ID, nil
}
