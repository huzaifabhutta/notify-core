# Notify-Core Go SDK

Official Go SDK for notify-core - a self-hosted notification service.

## Installation

```bash
go get github.com/huzaifabhutta/notify-core/pkg/sdk
```

## Quick Start

```go
package main

import (
    "log"

    "github.com/huzaifabhutta/notify-core/pkg/sdk"
)

func main() {
    // Create client
    client := sdk.NewClient("http://notify-core:8080", "your-api-key")

    // Send email
    resp, err := client.SendEmail(
        "user@example.com",
        "Welcome to our platform!",
        "welcome",
        map[string]interface{}{
            "CustomerName": "John Doe",
            "ActionURL":    "https://app.com/get-started",
        },
    )

    if err != nil {
        log.Fatalf("Failed to send email: %v", err)
    }

    log.Printf("Email sent successfully: %s", resp.Message)
}
```

## Usage

### Initialize Client

```go
// Basic initialization
client := sdk.NewClient("http://notify-core:8080", "your-api-key")

// With custom timeout
client := sdk.NewClient("http://notify-core:8080", "your-api-key").
    WithTimeout(10 * time.Second)
```

### Send Email

```go
resp, err := client.SendEmail(
    "recipient@example.com",
    "Your Order Confirmation",
    "order-confirmation",
    map[string]interface{}{
        "OrderID":    "12345",
        "Total":      "$99.99",
        "CustomerName": "Jane Smith",
    },
)

if err != nil {
    log.Printf("Error: %v", err)
    return
}

log.Printf("Success: %s (ID: %s)", resp.Message, resp.MessageID)
```

### Send WhatsApp Message

```go
resp, err := client.SendWhatsApp(
    "+1234567890",
    "order_update",
    map[string]interface{}{
        "order_id": "12345",
        "status":   "shipped",
    },
)
```

### Send SMS

```go
resp, err := client.SendSMS(
    "+1234567890",
    "otp",
    map[string]interface{}{
        "code": "123456",
    },
)
```

### Send Generic Notification

```go
notification := sdk.Notification{
    To:       "user@example.com",
    Channel:  "email",
    Template: "password-reset",
    Subject:  "Reset Your Password",
    Data: map[string]interface{}{
        "CustomerName": "John Doe",
        "ResetURL":     "https://app.com/reset?token=abc123",
        "ExpiresIn":    "24 hours",
    },
}

resp, err := client.Send(notification)
```

### Using Context

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.SendWithContext(ctx, notification)
```

### Health Check

```go
// Check if service is reachable
if err := client.Ping(); err != nil {
    log.Printf("Service is down: %v", err)
}
```

## Integration Examples

### In Your MRQZ Service

```go
package main

import (
    "github.com/huzaifabhutta/notify-core/pkg/sdk"
)

var notifier *sdk.Client

func init() {
    notifier = sdk.NewClient(
        os.Getenv("NOTIFY_URL"),    // http://notify-core:8080
        os.Getenv("NOTIFY_API_KEY"), // mrqz-key-abc123
    )
}

func SendOrderConfirmation(order *Order) error {
    _, err := notifier.SendEmail(
        order.CustomerEmail,
        "Your Order Confirmation",
        "order-confirmation",
        map[string]interface{}{
            "CustomerName": order.CustomerName,
            "OrderID":      order.ID,
            "Items":        order.Items,
            "Total":        order.Total,
        },
    )
    return err
}

func SendWhatsAppNotification(phone, message string) error {
    _, err := notifier.SendWhatsApp(phone, "notification", map[string]interface{}{
        "message": message,
    })
    return err
}
```

### In Your Kasbb Service

```go
package notifications

import (
    "sync"

    "github.com/huzaifabhutta/notify-core/pkg/sdk"
)

var (
    client *sdk.Client
    once   sync.Once
)

// GetNotifier returns a singleton instance
func GetNotifier() *sdk.Client {
    once.Do(func() {
        client = sdk.NewClient(
            "http://notify-core:8080",
            "kasbb-key-xyz789",
        )
    })
    return client
}

// SendWelcomeEmail sends welcome email to new users
func SendWelcomeEmail(email, name string) error {
    _, err := GetNotifier().SendEmail(
        email,
        "Welcome to Kasbb!",
        "welcome",
        map[string]interface{}{
            "CustomerName": name,
            "ActionURL":    "https://kasbb.com/onboarding",
        },
    )
    return err
}
```

## Error Handling

```go
resp, err := client.SendEmail(...)
if err != nil {
    // Check error type
    switch {
    case strings.Contains(err.Error(), "INVALID_REQUEST"):
        log.Printf("Invalid request: %v", err)
    case strings.Contains(err.Error(), "RATE_LIMIT_EXCEEDED"):
        log.Printf("Rate limited: %v", err)
        time.Sleep(time.Minute)
        // Retry
    case strings.Contains(err.Error(), "AUTH_REQUIRED"):
        log.Printf("Authentication failed: %v", err)
    default:
        log.Printf("Unexpected error: %v", err)
    }
    return err
}

log.Printf("Notification sent: %s", resp.MessageID)
```

## Configuration

### Environment Variables (Recommended)

```bash
export NOTIFY_URL=http://notify-core:8080
export NOTIFY_API_KEY=your-api-key-here
```

```go
client := sdk.NewClient(
    os.Getenv("NOTIFY_URL"),
    os.Getenv("NOTIFY_API_KEY"),
)
```

### Docker Compose

```yaml
# In your service's docker-compose.yml
services:
  your-service:
    environment:
      - NOTIFY_URL=http://notify-core:8080
      - NOTIFY_API_KEY=your-service-key:your-tenant
    networks:
      - internal-network
```

## API Reference

### Client Methods

- `NewClient(baseURL, apiKey string) *Client` - Create new client
- `WithTimeout(timeout time.Duration) *Client` - Set custom timeout
- `Send(notification Notification) (*Response, error)` - Send notification
- `SendWithContext(ctx context.Context, notification Notification) (*Response, error)` - Send with context
- `SendEmail(to, subject, template string, data map[string]interface{}) (*Response, error)` - Send email
- `SendWhatsApp(to, template string, data map[string]interface{}) (*Response, error)` - Send WhatsApp
- `SendSMS(to, template string, data map[string]interface{}) (*Response, error)` - Send SMS
- `Ping() error` - Health check
- `PingWithContext(ctx context.Context) error` - Health check with context

### Types

```go
type Notification struct {
    To       string                 // Recipient (email, phone)
    Channel  string                 // "email", "whatsapp", or "sms"
    Template string                 // Template name
    Subject  string                 // Email subject (optional)
    Data     map[string]interface{} // Template variables
    From     string                 // Override sender (optional)
}

type Response struct {
    Success   bool   // Operation success status
    Message   string // Human-readable message
    MessageID string // Unique message ID (if available)
}
```

## Testing

```go
// Use httptest for testing
func TestNotifications(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Mock response
        json.NewEncoder(w).Encode(sdk.Response{
            Success: true,
            Message: "Sent",
        })
    }))
    defer server.Close()

    client := sdk.NewClient(server.URL, "test-key")

    resp, err := client.SendEmail("test@example.com", "Test", "test", nil)
    assert.NoError(t, err)
    assert.True(t, resp.Success)
}
```

## Best Practices

1. **Singleton Pattern**: Create one client instance and reuse it
2. **Use Context**: Always use `SendWithContext` for timeout control
3. **Error Handling**: Handle rate limits and retries gracefully
4. **Environment Config**: Don't hardcode URLs or API keys
5. **Health Checks**: Implement health checks in your service
6. **Logging**: Log notification failures for debugging
7. **Templates**: Pre-create templates in notify-core before using

## Examples

See `/examples/go-sdk` directory for complete examples:
- Basic usage
- Error handling
- Retry logic
- Integration with web services
- Testing

## License

MIT
