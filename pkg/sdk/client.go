package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the notify-core SDK client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new SDK client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithTimeout sets a custom timeout for the HTTP client
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.httpClient.Timeout = timeout
	return c
}

// Notification represents a notification to be sent
type Notification struct {
	To       string                 `json:"to"`
	Channel  string                 `json:"channel"`
	Template string                 `json:"template"`
	Subject  string                 `json:"subject,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	From     string                 `json:"from,omitempty"`
}

// Response represents the API response
type Response struct {
	Status    string `json:"status"`            // "success" or "error"
	Message   string `json:"message"`
	MessageID string `json:"message_id,omitempty"`
	Channel   string `json:"channel,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Status    string `json:"status"`    // "error"
	Error     string `json:"error"`     // Error code
	Message   string `json:"message"`   // Error message
	Timestamp string `json:"timestamp,omitempty"`
}

// IsSuccess returns true if the response status is "success"
func (r *Response) IsSuccess() bool {
	return r.Status == "success"
}

// Send sends a notification
func (c *Client) Send(notification Notification) (*Response, error) {
	return c.SendWithContext(context.Background(), notification)
}

// SendWithContext sends a notification with context
func (c *Client) SendWithContext(ctx context.Context, notification Notification) (*Response, error) {
	// Validate notification
	if err := notification.Validate(); err != nil {
		return nil, fmt.Errorf("invalid notification: %w", err)
	}

	// Marshal request body
	body, err := json.Marshal(notification)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/send", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return nil, fmt.Errorf("API error (%s): %s", errResp.Error, errResp.Message)
	}

	// Parse success response
	var apiResp Response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &apiResp, nil
}

// SendEmail is a convenience method for sending emails
func (c *Client) SendEmail(to, subject, template string, data map[string]interface{}) (*Response, error) {
	return c.Send(Notification{
		To:       to,
		Channel:  "email",
		Subject:  subject,
		Template: template,
		Data:     data,
	})
}

// SendWhatsApp is a convenience method for sending WhatsApp messages
func (c *Client) SendWhatsApp(to, template string, data map[string]interface{}) (*Response, error) {
	return c.Send(Notification{
		To:       to,
		Channel:  "whatsapp",
		Template: template,
		Data:     data,
	})
}

// SendSMS is a convenience method for sending SMS messages
func (c *Client) SendSMS(to, template string, data map[string]interface{}) (*Response, error) {
	return c.Send(Notification{
		To:       to,
		Channel:  "sms",
		Template: template,
		Data:     data,
	})
}

// Validate validates the notification
func (n *Notification) Validate() error {
	if n.To == "" {
		return fmt.Errorf("recipient (to) is required")
	}

	if n.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if n.Template == "" {
		return fmt.Errorf("template is required")
	}

	// Channel-specific validation
	switch n.Channel {
	case "email", "whatsapp", "sms":
		// Valid channels
	default:
		return fmt.Errorf("unsupported channel: %s", n.Channel)
	}

	return nil
}

// Ping checks if the notify-core service is reachable
func (c *Client) Ping() error {
	return c.PingWithContext(context.Background())
}

// PingWithContext checks if the notify-core service is reachable with context
func (c *Client) PingWithContext(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service unhealthy (status %d)", resp.StatusCode)
	}

	return nil
}
