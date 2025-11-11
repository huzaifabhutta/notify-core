# Adapter Development Guide

This guide explains how to create custom adapters for notify-core, following Go best practices.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Creating an Adapter](#creating-an-adapter)
- [Registering an Adapter](#registering-an-adapter)
- [Testing Your Adapter](#testing-your-adapter)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Architecture Overview

notify-core uses a **registry pattern** for adapters:

```
┌─────────────────────────────────────────┐
│ Adapter Registry (Thread-Safe)         │
│ - Global singleton                      │
│ - Factory pattern                       │
│ - Type-based discovery                  │
└─────────────────────────────────────────┘
            ↓
┌─────────────────────────────────────────┐
│ Adapters (Self-Registering)            │
│ - SMTP (email)                          │
│ - WhatsApp Cloud API                    │
│ - Your Custom Adapter                   │
└─────────────────────────────────────────┘
```

### Key Components

1. **Adapter Interface**: Minimal interface all adapters must implement
2. **Registry**: Thread-safe registry for adapter discovery
3. **Factory**: Function that creates adapter instances
4. **Metadata**: Information about the adapter (name, type, description, version)

## Creating an Adapter

### Step 1: Implement the Adapter Interface

Every adapter must implement the `adapters.Adapter` interface:

```go
type Adapter interface {
    Send(ctx context.Context, req interface{}) (messageID string, err error)
    Name() string
}
```

**Example:**

```go
package myemail

import (
    "context"
    "github.com/huzaifabhutta/notify-core/internal/adapters"
)

type Adapter struct {
    config *Config
}

type Config struct {
    APIKey string
    From   string
}

// Send implements adapters.Adapter
func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
    // Extract request
    emailReq, err := extractEmailRequest(req)
    if err != nil {
        return "", err
    }

    // Send via your provider
    messageID, err := a.sendViaProvider(emailReq)
    if err != nil {
        return "", err
    }

    return messageID, nil
}

// Name implements adapters.Adapter
func (a *Adapter) Name() string {
    return "my-email"
}
```

### Step 2: Create a Factory Function

The factory function creates instances of your adapter:

```go
// NewAdapter is the factory function
func NewAdapter(cfg interface{}) (adapters.Adapter, error) {
    // Type assert to your config type
    config, ok := cfg.(*Config)
    if !ok {
        return nil, adapters.ErrInvalidAdapter
    }

    // Validate configuration
    if config.APIKey == "" {
        return nil, fmt.Errorf("API key is required")
    }

    // Create and return adapter
    return &Adapter{
        config: config,
    }, nil
}
```

## Registering an Adapter

### Auto-Registration (Recommended)

Use `init()` for automatic registration when the package is imported:

```go
package myemail

import "github.com/huzaifabhutta/notify-core/internal/adapters"

func init() {
    adapters.MustRegister(
        adapters.Metadata{
            Name:        "my-email",
            Type:        adapters.TypeEmail,
            Description: "My custom email adapter",
            Version:     "1.0.0",
        },
        NewAdapter,
    )
}
```

### Manual Registration

For dynamic registration at runtime:

```go
err := adapters.Register(
    adapters.Metadata{
        Name:        "my-email",
        Type:        adapters.TypeEmail,
        Description: "My custom email adapter",
        Version:     "1.0.0",
    },
    NewAdapter,
)
if err != nil {
    // Handle error (e.g., adapter already registered)
}
```

### Metadata Fields

- **Name**: Unique identifier (e.g., "smtp", "ses", "sendgrid")
- **Type**: Category (`adapters.TypeEmail`, `adapters.TypeWhatsApp`, `adapters.TypeSMS`)
- **Description**: Human-readable description
- **Version**: Semantic version (e.g., "1.0.0")

## Testing Your Adapter

### Unit Tests

```go
package myemail_test

import (
    "context"
    "testing"
    "github.com/huzaifabhutta/notify-core/internal/adapters/email/myemail"
)

func TestAdapter_Send(t *testing.T) {
    config := &myemail.Config{
        APIKey: "test-key",
        From:   "test@example.com",
    }

    adapter, err := myemail.NewAdapter(config)
    if err != nil {
        t.Fatalf("NewAdapter() failed: %v", err)
    }

    ctx := context.Background()
    req := map[string]interface{}{
        "to":      "recipient@example.com",
        "subject": "Test",
        "body":    "Test message",
    }

    messageID, err := adapter.Send(ctx, req)
    if err != nil {
        t.Fatalf("Send() failed: %v", err)
    }

    if messageID == "" {
        t.Error("Expected non-empty message ID")
    }
}
```

### Integration Tests

```go
func TestAdapter_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Use real configuration
    config := &myemail.Config{
        APIKey: os.Getenv("MY_EMAIL_API_KEY"),
        From:   "test@example.com",
    }

    adapter, _ := myemail.NewAdapter(config)

    // Test actual send
    messageID, err := adapter.Send(context.Background(), testRequest)
    if err != nil {
        t.Fatalf("Integration test failed: %v", err)
    }

    t.Logf("Message sent: %s", messageID)
}
```

### Testing Registration

```go
package myemail_test

import (
    "testing"
    "github.com/huzaifabhutta/notify-core/internal/adapters"
    _ "github.com/huzaifabhutta/notify-core/internal/adapters/email/myemail" // Trigger init()
)

func TestRegistration(t *testing.T) {
    if !adapters.Has("my-email") {
        t.Error("Adapter not registered")
    }

    meta := adapters.GetMetadata("my-email")
    if meta == nil {
        t.Fatal("Metadata not found")
    }

    if meta.Type != adapters.TypeEmail {
        t.Errorf("Expected type %s, got %s", adapters.TypeEmail, meta.Type)
    }
}
```

## Best Practices

### 1. Error Handling

✅ **Do:** Return wrapped errors with context

```go
func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
    result, err := a.provider.Send(emailReq)
    if err != nil {
        return "", fmt.Errorf("failed to send via provider: %w", err)
    }
    return result.ID, nil
}
```

❌ **Don't:** Swallow errors or panic

```go
// Bad
func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
    result, err := a.provider.Send(emailReq)
    if err != nil {
        panic(err) // Never panic in library code!
    }
    return result.ID, nil
}
```

### 2. Configuration Validation

✅ **Do:** Validate in the factory function

```go
func NewAdapter(cfg interface{}) (adapters.Adapter, error) {
    config, ok := cfg.(*Config)
    if !ok {
        return nil, adapters.ErrInvalidAdapter
    }

    // Validate required fields
    if config.APIKey == "" {
        return nil, fmt.Errorf("API key is required")
    }

    if config.From == "" {
        return nil, fmt.Errorf("sender email is required")
    }

    return &Adapter{config: config}, nil
}
```

### 3. Context Handling

✅ **Do:** Respect context cancellation

```go
func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
    // Check context before expensive operations
    select {
    case <-ctx.Done():
        return "", ctx.Err()
    default:
    }

    // Pass context to underlying HTTP calls
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, body)
    resp, err := a.httpClient.Do(httpReq)
    // ...
}
```

### 4. Logging

✅ **Do:** Use structured logging from context

```go
import "github.com/huzaifabhutta/notify-core/internal/logger"

func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
    log := logger.FromContext(ctx)

    log.Debug().
        Str("adapter", a.Name()).
        Str("to", req.To).
        Msg("Sending notification")

    // ...

    log.Info().
        Str("message_id", messageID).
        Msg("Notification sent successfully")

    return messageID, nil
}
```

### 5. Thread Safety

✅ **Do:** Make adapters thread-safe if they hold state

```go
type Adapter struct {
    mu     sync.RWMutex
    config *Config
    client *http.Client // Clients are typically thread-safe
}
```

### 6. Request Type Flexibility

✅ **Do:** Accept multiple request types

```go
func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
    // Support direct struct
    if emailReq, ok := req.(*EmailRequest); ok {
        return a.sendEmail(ctx, emailReq)
    }

    // Support map (for flexibility)
    if mapReq, ok := req.(map[string]interface{}); ok {
        return a.sendEmailFromMap(ctx, mapReq)
    }

    // Use common adapter helper
    baseReq, err := adapters.ExtractBaseRequest(req)
    if err != nil {
        return "", err
    }

    return a.sendEmailFromBase(ctx, &baseReq)
}
```

### 7. Idempotency

✅ **Do:** Generate deterministic message IDs when possible

```go
func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
    // If provider returns an ID, use it
    providerID, err := a.provider.Send(req)
    if err != nil {
        return "", err
    }

    return providerID, nil
}
```

### 8. Retry Logic

✅ **Do:** Implement retry for transient failures

```go
func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
    var lastErr error

    for attempt := 1; attempt <= 3; attempt++ {
        messageID, err := a.attemptSend(ctx, req)
        if err == nil {
            return messageID, nil
        }

        // Don't retry non-retryable errors
        if !isRetryable(err) {
            return "", err
        }

        lastErr = err
        time.Sleep(time.Duration(attempt) * time.Second)
    }

    return "", fmt.Errorf("failed after 3 attempts: %w", lastErr)
}
```

## Examples

### Example 1: AWS SES Adapter

```go
package ses

import (
    "context"
    "github.com/aws/aws-sdk-go-v2/service/sesv2"
    "github.com/huzaifabhutta/notify-core/internal/adapters"
)

func init() {
    adapters.MustRegister(
        adapters.Metadata{
            Name:        "ses",
            Type:        adapters.TypeEmail,
            Description: "AWS SES native adapter with full SES features",
            Version:     "1.0.0",
        },
        NewAdapter,
    )
}

type Config struct {
    Region          string
    FromEmail       string
    ConfigurationSet string
}

type Adapter struct {
    client *sesv2.Client
    config *Config
}

func NewAdapter(cfg interface{}) (adapters.Adapter, error) {
    config, ok := cfg.(*Config)
    if !ok {
        return nil, adapters.ErrInvalidAdapter
    }

    // Create AWS SES client
    client := sesv2.NewFromConfig(/* AWS config */)

    return &Adapter{
        client: client,
        config: config,
    }, nil
}

func (a *Adapter) Name() string {
    return "ses"
}

func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
    // Extract email request
    emailReq := extractEmailRequest(req)

    // Build SES request
    input := &sesv2.SendEmailInput{
        FromEmailAddress: &a.config.FromEmail,
        Destination: &types.Destination{
            ToAddresses: []string{emailReq.To},
        },
        Content: &types.EmailContent{
            Simple: &types.Message{
                Subject: &types.Content{Data: &emailReq.Subject},
                Body: &types.Body{
                    Text: &types.Content{Data: &emailReq.Body},
                },
            },
        },
    }

    // Send via SES
    output, err := a.client.SendEmail(ctx, input)
    if err != nil {
        return "", fmt.Errorf("SES send failed: %w", err)
    }

    return *output.MessageId, nil
}
```

### Example 2: Twilio SMS Adapter

```go
package twilio

import (
    "context"
    "github.com/huzaifabhutta/notify-core/internal/adapters"
    "github.com/twilio/twilio-go"
    twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

func init() {
    adapters.MustRegister(
        adapters.Metadata{
            Name:        "twilio-sms",
            Type:        adapters.TypeSMS,
            Description: "Twilio SMS adapter",
            Version:     "1.0.0",
        },
        NewAdapter,
    )
}

type Config struct {
    AccountSID string
    AuthToken  string
    FromNumber string
}

type Adapter struct {
    client *twilio.RestClient
    config *Config
}

func NewAdapter(cfg interface{}) (adapters.Adapter, error) {
    config, ok := cfg.(*Config)
    if !ok {
        return nil, adapters.ErrInvalidAdapter
    }

    client := twilio.NewRestClientWithParams(twilio.ClientParams{
        Username: config.AccountSID,
        Password: config.AuthToken,
    })

    return &Adapter{
        client: client,
        config: config,
    }, nil
}

func (a *Adapter) Name() string {
    return "twilio-sms"
}

func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
    smsReq := extractSMSRequest(req)

    params := &twilioApi.CreateMessageParams{}
    params.SetTo(smsReq.To)
    params.SetFrom(a.config.FromNumber)
    params.SetBody(smsReq.Body)

    resp, err := a.client.Api.CreateMessage(params)
    if err != nil {
        return "", fmt.Errorf("Twilio send failed: %w", err)
    }

    return *resp.Sid, nil
}
```

## Package Structure

Recommended package structure for adapters:

```
internal/adapters/
├── adapter.go              # Core types (Type, Metadata)
├── common.go               # Common types (Adapter interface, BaseRequest)
├── registry.go             # Registry implementation
├── registry_test.go        # Registry tests
├── registration_test.go    # Test all adapters are registered
├── email/
│   ├── smtp/
│   │   ├── register.go    # SMTP adapter registration
│   │   └── README.md      # SMTP-specific documentation
│   └── ses/               # Your AWS SES adapter
│       ├── adapter.go
│       ├── adapter_test.go
│       ├── register.go
│       └── README.md
├── whatsapp/
│   └── cloud/
│       ├── register.go
│       └── README.md
└── sms/
    └── twilio/
        ├── adapter.go
        ├── adapter_test.go
        ├── register.go
        └── README.md
```

## Contributing Adapters

To contribute a new adapter to notify-core:

1. **Create adapter** following this guide
2. **Write comprehensive tests** (unit + integration)
3. **Document configuration** in README.md
4. **Add examples** to documentation
5. **Submit pull request** with:
   - Adapter code
   - Tests
   - Documentation
   - Example usage

## FAQ

**Q: Should I use the global registry or create my own?**
A: Use the global registry for most cases. Create a custom registry only if you need isolated adapter management.

**Q: Can one adapter be registered under multiple names?**
A: No, each name must be unique. But you can create wrapper factories that register the same underlying adapter with different configurations.

**Q: How do I handle adapter-specific features?**
A: Define adapter-specific interfaces and use type assertions:

```go
type BounceHandler interface {
    HandleBounce(ctx context.Context, notification interface{}) error
}

if bounceHandler, ok := adapter.(BounceHandler); ok {
    bounceHandler.HandleBounce(ctx, notification)
}
```

**Q: Should adapters be stateless?**
A: Prefer stateless adapters when possible. If you need state (e.g., connection pools), make it thread-safe.

**Q: How do I version my adapter?**
A: Use semantic versioning in the Metadata.Version field. Breaking changes should bump the major version.

## Support

- **Documentation**: See [ADAPTER_ARCHITECTURE.md](./ADAPTER_ARCHITECTURE.md)
- **Examples**: Check existing adapters in `internal/adapters/`
- **Issues**: Report bugs or request features via GitHub Issues

Happy adapter development! 🚀
