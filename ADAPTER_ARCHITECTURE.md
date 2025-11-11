# Adapter Architecture Proposal

## Overview

This document outlines the recommended adapter architecture for notify-core to support long-term maintainability, open-source contributions, and pluggable provider integrations.

## Design Principles

1. **Default to AWS**: AWS SES for email, SNS for SMS (cost-effective, reliable)
2. **Pluggable**: Easy to add/remove adapters without core changes
3. **Vendor-Agnostic Core**: Core service doesn't know about specific providers
4. **Registry Pattern**: Adapters register themselves at startup
5. **Configuration-Driven**: Which adapter to use is in config, not code

## Architecture

### Directory Structure

```
internal/adapters/
├── registry.go           # Central adapter registry
├── interface.go          # Core interfaces
├── factory.go            # Adapter factory
├── email/
│   ├── ses/             # AWS SES adapter (DEFAULT)
│   │   ├── adapter.go
│   │   ├── adapter_test.go
│   │   └── README.md
│   ├── smtp/            # Generic SMTP adapter
│   │   ├── adapter.go
│   │   ├── adapter_test.go
│   │   └── README.md
│   ├── sendgrid/        # SendGrid native API (optional)
│   └── mailgun/         # Mailgun native API (optional)
├── whatsapp/
│   ├── cloud/           # Facebook Cloud API (DEFAULT)
│   ├── twilio/          # Twilio WhatsApp
│   └── generic/         # Generic HTTP API
└── sms/
    ├── sns/             # AWS SNS (DEFAULT)
    └── twilio/          # Twilio SMS
```

### Core Interfaces

```go
// internal/adapters/interface.go

package adapters

import "context"

// Adapter is the base interface all channel adapters must implement
type Adapter interface {
    Send(ctx context.Context, req interface{}) (messageID string, err error)
    Name() string
    Type() AdapterType
}

// AdapterType defines the adapter category
type AdapterType string

const (
    AdapterTypeEmail    AdapterType = "email"
    AdapterTypeWhatsApp AdapterType = "whatsapp"
    AdapterTypeSMS      AdapterType = "sms"
)

// EmailAdapter extends Adapter with email-specific methods
type EmailAdapter interface {
    Adapter
    SendEmail(ctx context.Context, req *EmailRequest) (messageID string, err error)
    ValidateEmail(email string) error
}

// WhatsAppAdapter extends Adapter with WhatsApp-specific methods
type WhatsAppAdapter interface {
    Adapter
    SendTemplate(ctx context.Context, req *WhatsAppRequest) (messageID string, err error)
    ValidatePhoneNumber(phone string) error
}

// SMSAdapter extends Adapter with SMS-specific methods
type SMSAdapter interface {
    Adapter
    SendSMS(ctx context.Context, req *SMSRequest) (messageID string, err error)
    ValidatePhoneNumber(phone string) error
}

// EmailRequest is the standard email request structure
type EmailRequest struct {
    To       string
    From     string
    Subject  string
    Body     string
    Template string
    Data     map[string]interface{}
}

// WhatsAppRequest is the standard WhatsApp request structure
type WhatsAppRequest struct {
    To       string
    Template string
    Data     map[string]interface{}
}

// SMSRequest is the standard SMS request structure
type SMSRequest struct {
    To   string
    From string
    Body string
}
```

### Registry Pattern

```go
// internal/adapters/registry.go

package adapters

import (
    "fmt"
    "sync"
)

// Registry manages adapter registration and creation
type Registry struct {
    mu       sync.RWMutex
    adapters map[string]AdapterFactory
}

// AdapterFactory creates an adapter instance
type AdapterFactory func(config interface{}) (Adapter, error)

var globalRegistry = &Registry{
    adapters: make(map[string]AdapterFactory),
}

// Register registers an adapter factory
func Register(name string, factory AdapterFactory) {
    globalRegistry.mu.Lock()
    defer globalRegistry.mu.Unlock()
    globalRegistry.adapters[name] = factory
}

// Get retrieves an adapter by name
func Get(name string, config interface{}) (Adapter, error) {
    globalRegistry.mu.RLock()
    factory, ok := globalRegistry.adapters[name]
    globalRegistry.mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("adapter %s not found", name)
    }

    return factory(config)
}

// List returns all registered adapter names
func List() []string {
    globalRegistry.mu.RLock()
    defer globalRegistry.mu.RUnlock()

    names := make([]string, 0, len(globalRegistry.adapters))
    for name := range globalRegistry.adapters {
        names = append(names, name)
    }
    return names
}
```

### Adapter Implementation Example

```go
// internal/adapters/email/ses/adapter.go

package ses

import (
    "context"

    "github.com/aws/aws-sdk-go-v2/service/sesv2"
    "github.com/huzaifabhutta/notify-core/internal/adapters"
)

func init() {
    // Auto-register on import
    adapters.Register("email-ses", NewAdapter)
}

type Adapter struct {
    client *sesv2.Client
    config *Config
}

type Config struct {
    Region          string
    FromEmail       string
    ConfigurationSet string
}

func NewAdapter(cfg interface{}) (adapters.Adapter, error) {
    sesConfig := cfg.(*Config)

    // Initialize AWS SES client
    client := sesv2.NewFromConfig(/* AWS config */)

    return &Adapter{
        client: client,
        config: sesConfig,
    }, nil
}

func (a *Adapter) Name() string {
    return "email-ses"
}

func (a *Adapter) Type() adapters.AdapterType {
    return adapters.AdapterTypeEmail
}

func (a *Adapter) Send(ctx context.Context, req interface{}) (string, error) {
    emailReq := req.(*adapters.EmailRequest)

    // Use AWS SES SDK
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

    if a.config.ConfigurationSet != "" {
        input.ConfigurationSetName = &a.config.ConfigurationSet
    }

    output, err := a.client.SendEmail(ctx, input)
    if err != nil {
        return "", err
    }

    return *output.MessageId, nil
}
```

### Configuration

```yaml
# config.yaml

adapters:
  email:
    default: "email-ses"  # Default email adapter

    # SES configuration
    ses:
      region: "us-east-1"
      from: "noreply@example.com"
      configuration_set: "my-config-set"

    # SMTP fallback (for self-hosted or other providers)
    smtp:
      host: "smtp.gmail.com"
      port: 587
      user: "user@gmail.com"
      password: "${SMTP_PASSWORD}"
      from: "noreply@example.com"

  whatsapp:
    default: "whatsapp-cloud"  # Facebook Cloud API

    cloud:
      token: "${WHATSAPP_TOKEN}"
      phone_id: "${WHATSAPP_PHONE_ID}"

    # Alternative: Twilio WhatsApp
    twilio:
      account_sid: "${TWILIO_ACCOUNT_SID}"
      auth_token: "${TWILIO_AUTH_TOKEN}"
      from: "whatsapp:+14155238886"

  sms:
    default: "sms-sns"  # AWS SNS

    sns:
      region: "us-east-1"
      sender_id: "MyApp"
```

### Service Integration

```go
// internal/services/notify_service.go

func (s *NotifyService) sendEmail(ctx context.Context, tenant *models.Tenant, req *SendRequest) (messageID, source string, err error) {
    // Resolve credentials
    creds, err := s.credentialResolver.ResolveEmailCredentials(tenant)
    if err != nil {
        return "", "", err
    }

    // Get adapter name from config (defaults to "email-ses")
    adapterName := s.config.Adapters.Email.Default
    if adapterName == "" {
        adapterName = "email-ses"  // Default
    }

    // Create adapter from registry
    adapter, err := adapters.Get(adapterName, creds)
    if err != nil {
        return "", "", fmt.Errorf("failed to create email adapter: %w", err)
    }

    // Build request
    emailReq := &adapters.EmailRequest{
        To:       req.To,
        From:     creds.From,
        Subject:  req.Subject,
        Body:     req.Body,
        Template: req.Template,
        Data:     req.Data,
    }

    // Send
    messageID, err = adapter.Send(ctx, emailReq)
    return messageID, creds.Source, err
}
```

## Migration Strategy

### Phase 1: Extract Current Adapters (Week 1)
- Move `internal/email` → `internal/adapters/email/smtp`
- Move `internal/whatsapp` → `internal/adapters/whatsapp/generic`
- Create registry infrastructure
- No breaking changes to API

### Phase 2: Add AWS Adapters (Week 2)
- Implement `internal/adapters/email/ses`
- Implement `internal/adapters/sms/sns`
- Make SES the default in config
- SMTP remains as fallback

### Phase 3: Documentation (Week 3)
- Write adapter development guide
- Document configuration options
- Create example custom adapter
- Update deployment docs

### Phase 4: Community (Ongoing)
- Accept community adapter contributions
- Maintain adapter compatibility matrix
- Version adapter interfaces carefully

## Benefits

### For Users
✅ **Easy Setup**: Default AWS adapters work out-of-box
✅ **Flexibility**: Can switch providers via config
✅ **Cost-Effective**: AWS SES is cheap ($0.10/1000 emails)
✅ **Self-Hosted**: Can use SMTP adapter for any provider

### For Contributors
✅ **Clear Interface**: Adapter interface is well-defined
✅ **Independent**: Each adapter is self-contained
✅ **Testable**: Mock adapters for testing
✅ **Documented**: Each adapter has README

### For Maintainers
✅ **Modular**: Changes don't affect core
✅ **Scalable**: Easy to add new adapters
✅ **Deprecatable**: Can remove old adapters cleanly
✅ **Versioned**: Adapters can evolve independently

## Comparison: Current vs Proposed

| Aspect | Current | Proposed |
|--------|---------|----------|
| **Email** | Generic SMTP only | SES (default) + SMTP (fallback) + community |
| **Extensibility** | Hard to add providers | Registry pattern, easy to add |
| **AWS Integration** | Via SMTP only | Native SDK with full features |
| **Configuration** | Hardcoded in code | Config-driven adapter selection |
| **Community** | Hard to contribute | Clear adapter development guide |
| **Testing** | Coupled to providers | Mock adapters easily |
| **Default** | No default | AWS SES/SNS (cost-effective) |

## Recommended Default Stack

For a self-hosted open-source project, I recommend:

**Defaults (AWS-based):**
- Email: AWS SES ($0.10/1000 emails)
- SMS: AWS SNS ($0.00645/SMS in US)
- WhatsApp: Facebook Cloud API (free + message costs)

**Fallbacks (Self-hosted):**
- Email: SMTP adapter (works with ANY SMTP server)
- SMS: Twilio adapter
- WhatsApp: Generic HTTP adapter (current implementation)

**Why AWS as Default?**
1. ✅ Cost-effective ($100/month covers ~1M emails)
2. ✅ Reliable (99.99% SLA)
3. ✅ Scalable (millions of messages/day)
4. ✅ Free tier (62,000 emails/month free first 12 months)
5. ✅ Well-documented SDKs
6. ✅ Popular in open-source projects

## Action Items

To implement this architecture:

- [ ] Create adapter interface and registry
- [ ] Refactor existing email adapter to SMTP adapter
- [ ] Implement AWS SES adapter (with tests)
- [ ] Implement AWS SNS adapter (for SMS)
- [ ] Update configuration system
- [ ] Write adapter development guide
- [ ] Update documentation
- [ ] Add adapter examples

## Timeline Estimate

- **Week 1-2**: Registry pattern + refactor existing
- **Week 3-4**: AWS SES/SNS adapters
- **Week 5**: Documentation + examples
- **Week 6**: Testing + polish

Total: ~6 weeks for full implementation

## Questions to Decide

1. Should we include adapter versioning from day 1?
2. Should adapters be plugins (separate repos) or monorepo?
3. Should we support multiple adapters simultaneously (routing)?
4. What's the deprecation policy for old adapters?

## Conclusion

The plugin-based adapter architecture provides:
- ✅ AWS as sensible default
- ✅ Easy provider switching
- ✅ Community contribution path
- ✅ Long-term maintainability
- ✅ No vendor lock-in

This balances "batteries included" (AWS defaults) with "bring your own" (custom adapters).
