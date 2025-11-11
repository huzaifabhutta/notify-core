# ⚠️ CRITICAL FIXES REQUIRED - DO NOT DEPLOY WITHOUT THESE

## 🚨 STOP: Read This Before Deploying

This document outlines **mandatory fixes** for critical security vulnerabilities found in the codebase.

**Current Status:** 🔴 **NOT SAFE FOR PRODUCTION**

---

## 1. ADD AUTHENTICATION (CRITICAL)

### Current Problem
```go
// cmd/server/main.go:48 - NO AUTHENTICATION!
app.Post("/send", func(c *fiber.Ctx) error {
    var req notify.SendRequest
    if err := c.BodyParser(&req); err != nil { ... }
    // Anyone can send emails!
```

### Required Fix

**Option A: API Key Authentication (Recommended for MVP)**

```go
// internal/auth/middleware.go
package auth

import (
    "github.com/gofiber/fiber/v2"
)

func RequireAPIKey(validKeys map[string]bool) fiber.Handler {
    return func(c *fiber.Ctx) error {
        apiKey := c.Get("X-API-Key")
        if apiKey == "" {
            return c.Status(401).JSON(fiber.Map{
                "error": "Missing API key",
            })
        }

        if !validKeys[apiKey] {
            return c.Status(403).JSON(fiber.Map{
                "error": "Invalid API key",
            })
        }

        // Store tenant info for multi-tenancy
        c.Locals("api_key", apiKey)
        return c.Next()
    }
}
```

**Update main.go:**
```go
// Load API keys from environment
validAPIKeys := map[string]bool{
    os.Getenv("API_KEY"): true,
}

// Protect /send endpoint
app.Post("/send", auth.RequireAPIKey(validAPIKeys), func(c *fiber.Ctx) error {
    // Now authenticated!
```

**Add to .env.example:**
```env
# Authentication
API_KEY=your-secret-api-key-here
```

---

## 2. FIX PATH TRAVERSAL VULNERABILITY (CRITICAL)

### Current Problem
```go
// internal/email/adapter.go:80
templatePath := filepath.Join(a.templates.Dir, templateName+".html")
// Can read ANY file: templateName = "../../etc/passwd"
```

### Required Fix

```go
// internal/email/adapter.go

import (
    "path/filepath"
    "regexp"
    "errors"
)

var validTemplateNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (a *Adapter) renderTemplate(templateName string, data map[string]interface{}) (string, error) {
    // CRITICAL: Validate template name
    if !validTemplateNameRegex.MatchString(templateName) {
        return "", errors.New("invalid template name: must contain only letters, numbers, hyphens, and underscores")
    }

    templatePath := filepath.Join(a.templates.Dir, templateName+".html")

    // CRITICAL: Verify resolved path is within templates directory
    absTemplatePath, err := filepath.Abs(templatePath)
    if err != nil {
        return "", fmt.Errorf("invalid template path: %w", err)
    }

    absTemplateDir, err := filepath.Abs(a.templates.Dir)
    if err != nil {
        return "", fmt.Errorf("invalid template directory: %w", err)
    }

    if !strings.HasPrefix(absTemplatePath, absTemplateDir) {
        return "", errors.New("invalid template: path traversal detected")
    }

    // Now safe to parse
    tmpl, err := template.ParseFiles(templatePath)
    if err != nil {
        return "", fmt.Errorf("template not found: %s", templateName)
    }

    // Rest of function...
}
```

---

## 3. FIX TYPE SYSTEM (CRITICAL DESIGN FLAW)

### Current Problem
```go
// Adapter interface is broken - email.Adapter doesn't match
type Adapter interface {
    Send(ctx context.Context, req *SendRequest) error  // Says this
}

// But implements this:
func (a *Adapter) Send(ctx context.Context, req interface{}) error
```

### Required Fix: Option A (Recommended)

**Make interface accept interface{}:**
```go
// internal/notify/service.go

type Adapter interface {
    Send(ctx context.Context, req interface{}) error
    Name() string
}

// Remove the fragile anonymous struct conversion
func (s *Service) Send(ctx context.Context, req *SendRequest) error {
    // Validate request
    if err := s.validateRequest(req); err != nil {
        return fmt.Errorf("invalid request: %w", err)
    }

    adapter, ok := s.adapters[req.Channel]
    if !ok {
        return fmt.Errorf("unsupported channel: %s", req.Channel)
    }

    // Pass request directly - let adapter handle conversion
    if err := adapter.Send(ctx, req); err != nil {
        return fmt.Errorf("failed to send via %s: %w", adapter.Name(), err)
    }

    return nil
}
```

**Update email adapter:**
```go
// internal/email/adapter.go

func (a *Adapter) Send(ctx context.Context, req interface{}) error {
    // Type assertion with better error
    sendReq, ok := req.(*notify.SendRequest)
    if !ok {
        return fmt.Errorf("invalid request type: expected *notify.SendRequest, got %T", req)
    }

    // Rest of implementation...
}
```

---

## 4. ADD INPUT VALIDATION (CRITICAL)

### Required Implementation

```go
// internal/notify/validation.go
package notify

import (
    "fmt"
    "net/mail"
    "regexp"
)

const (
    MaxSubjectLength = 998  // RFC 5322
    MaxDataSize      = 1024 * 100 // 100KB
)

var validTemplateNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (s *Service) validateRequest(req *SendRequest) error {
    // Validate recipient
    if req.To == "" {
        return fmt.Errorf("recipient (to) is required")
    }

    // Validate email format for email channel
    if req.Channel == ChannelEmail {
        if _, err := mail.ParseAddress(req.To); err != nil {
            return fmt.Errorf("invalid email address: %w", err)
        }
    }

    // Validate channel
    if req.Channel == "" {
        return fmt.Errorf("channel is required")
    }

    // Validate template name (prevent path traversal)
    if req.Template == "" {
        return fmt.Errorf("template is required")
    }

    if !validTemplateNameRegex.MatchString(req.Template) {
        return fmt.Errorf("invalid template name: must contain only letters, numbers, hyphens, and underscores")
    }

    // Validate subject length
    if len(req.Subject) > MaxSubjectLength {
        return fmt.Errorf("subject too long: maximum %d characters", MaxSubjectLength)
    }

    // Validate data size (prevent DoS)
    dataSize := estimateDataSize(req.Data)
    if dataSize > MaxDataSize {
        return fmt.Errorf("data payload too large: maximum %d bytes", MaxDataSize)
    }

    return nil
}

func estimateDataSize(data map[string]interface{}) int {
    // Rough estimate of JSON size
    size := 0
    for k, v := range data {
        size += len(k)
        size += len(fmt.Sprintf("%v", v))
    }
    return size
}
```

---

## 5. FIX ERROR EXPOSURE (CRITICAL)

### Current Problem
```go
// cmd/server/main.go:56-59
if err := notifyService.Send(c.Context(), &req); err != nil {
    return c.Status(500).JSON(fiber.Map{
        "error": err.Error(), // Leaks credentials, paths, etc.!
    })
}
```

### Required Fix

```go
// internal/errors/errors.go
package errors

type ErrorCode string

const (
    ErrInvalidRequest    ErrorCode = "INVALID_REQUEST"
    ErrAuthFailed       ErrorCode = "AUTH_FAILED"
    ErrUnsupportedChannel ErrorCode = "UNSUPPORTED_CHANNEL"
    ErrTemplateNotFound  ErrorCode = "TEMPLATE_NOT_FOUND"
    ErrSendFailed        ErrorCode = "SEND_FAILED"
    ErrRateLimit         ErrorCode = "RATE_LIMIT_EXCEEDED"
)

type AppError struct {
    Code       ErrorCode `json:"code"`
    Message    string    `json:"message"`
    InternalErr error    `json:"-"` // Never sent to client
}

func (e *AppError) Error() string {
    if e.InternalErr != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.InternalErr)
    }
    return e.Message
}
```

**Update main.go:**
```go
if err := notifyService.Send(c.Context(), &req); err != nil {
    // Log internal error with full details
    log.Printf("Send failed: %v", err)

    // Return safe error to client
    var appErr *errors.AppError
    if errors.As(err, &appErr) {
        return c.Status(500).JSON(fiber.Map{
            "error": appErr.Code,
            "message": appErr.Message,
        })
    }

    // Generic error for unexpected issues
    return c.Status(500).JSON(fiber.Map{
        "error": "INTERNAL_ERROR",
        "message": "Failed to send notification",
    })
}
```

---

## 6. CALL CONFIG VALIDATION (HIGH PRIORITY)

### Simple Fix

```go
// cmd/server/main.go

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // CRITICAL: Validate configuration before starting
    if err := cfg.Validate(); err != nil {
        log.Fatalf("Invalid configuration: %v", err)
    }

    // Continue with server setup...
}
```

**Enhance validation:**
```go
// internal/config/config.go

func (c *Config) Validate() error {
    // SMTP validation
    if c.SMTP.Host == "" {
        return fmt.Errorf("SMTP_HOST is required")
    }
    if c.SMTP.User == "" {
        return fmt.Errorf("SMTP_USER is required")
    }
    if c.SMTP.Password == "" {
        return fmt.Errorf("SMTP_PASS is required")
    }
    if c.SMTP.Port < 1 || c.SMTP.Port > 65535 {
        return fmt.Errorf("invalid SMTP_PORT: must be between 1-65535")
    }

    // Validate email format
    if _, err := mail.ParseAddress(c.SMTP.From); err != nil {
        return fmt.Errorf("invalid SMTP_FROM email address: %w", err)
    }

    // Templates directory validation
    if _, err := os.Stat(c.Templates.Dir); os.IsNotExist(err) {
        return fmt.Errorf("templates directory does not exist: %s", c.Templates.Dir)
    }

    return nil
}
```

---

## 7. ADD RATE LIMITING (HIGH PRIORITY)

### Implementation

```bash
go get github.com/gofiber/fiber/v2/middleware/limiter
```

```go
// cmd/server/main.go

import (
    "github.com/gofiber/fiber/v2/middleware/limiter"
    "time"
)

func main() {
    // ... config setup ...

    app := fiber.New(...)

    // Add rate limiting middleware
    app.Use(limiter.New(limiter.Config{
        Max:        20,                    // 20 requests
        Expiration: 1 * time.Minute,       // per minute
        KeyGenerator: func(c *fiber.Ctx) string {
            // Rate limit by API key (after auth) or IP
            apiKey := c.Locals("api_key")
            if apiKey != nil {
                return apiKey.(string)
            }
            return c.IP() // Fallback to IP
        },
        LimitReached: func(c *fiber.Ctx) error {
            return c.Status(429).JSON(fiber.Map{
                "error": "RATE_LIMIT_EXCEEDED",
                "message": "Too many requests. Please try again later.",
            })
        },
    }))

    // Rest of middleware and routes...
}
```

---

## 8. FIX DOCKER POSTGRES PASSWORD (HIGH PRIORITY)

### Current Problem
```yaml
# docker-compose.yml:55
POSTGRES_PASSWORD=notify  # Hardcoded weak password!
```

### Required Fix

```yaml
# docker-compose.yml
postgres:
  environment:
    - POSTGRES_USER=${DB_USER:-notify}
    - POSTGRES_PASSWORD=${DB_PASSWORD:?Database password must be set}
    # :? means required - will error if not set
```

**Update .env.example:**
```env
# Database (generate strong password)
DB_USER=notify
DB_PASSWORD=your-secure-random-password-here
DB_URL=postgres://notify:your-secure-random-password-here@postgres:5432/notify?sslmode=disable
```

**Documentation:**
```bash
# Generate secure password
openssl rand -base64 32
```

---

## ✅ VERIFICATION CHECKLIST

Before deploying, verify:

- [ ] API key authentication is implemented and tested
- [ ] Template name validation prevents path traversal
- [ ] Type system is fixed (no anonymous struct conversions)
- [ ] Input validation rejects invalid emails, long subjects, large payloads
- [ ] Error messages don't leak internal details
- [ ] Config.Validate() is called on startup
- [ ] Rate limiting is enabled and tested
- [ ] Docker uses strong, env-based passwords
- [ ] All fixes have unit tests
- [ ] Integration tests cover security scenarios
- [ ] go.sum file exists and is committed

---

## 🧪 SECURITY TEST CASES

Add these tests:

```go
// Test path traversal prevention
func TestPathTraversalPrevention(t *testing.T) {
    maliciousTemplates := []string{
        "../../../etc/passwd",
        "..%2F..%2F..%2Fetc%2Fpasswd",
        "....//....//etc/passwd",
    }

    for _, tmpl := range maliciousTemplates {
        req := SendRequest{
            Template: tmpl,
            // ...
        }

        err := service.Send(ctx, &req)
        if err == nil {
            t.Errorf("Expected error for malicious template: %s", tmpl)
        }
    }
}

// Test authentication
func TestUnauthenticatedRequest(t *testing.T) {
    resp := httptest.NewRequest("POST", "/send", body)
    // No X-API-Key header

    if resp.StatusCode != 401 {
        t.Error("Expected 401 Unauthorized")
    }
}

// Test rate limiting
func TestRateLimit(t *testing.T) {
    for i := 0; i < 25; i++ {
        resp := makeRequest(t)
        if i < 20 {
            assert.Equal(t, 200, resp.StatusCode)
        } else {
            assert.Equal(t, 429, resp.StatusCode)
        }
    }
}
```

---

## 📊 IMPLEMENTATION TIMELINE

**Week 1.5 - Critical Fixes (3-5 days)**

| Day | Task | Priority |
|-----|------|----------|
| 1 | Add API key authentication | 🔴 Critical |
| 1 | Fix path traversal vulnerability | 🔴 Critical |
| 2 | Add input validation | 🔴 Critical |
| 2 | Fix error exposure | 🔴 Critical |
| 3 | Fix type system | 🔴 Critical |
| 3 | Add rate limiting | 🟠 High |
| 4 | Call config validation | 🟠 High |
| 4 | Fix Docker passwords | 🟠 High |
| 5 | Write security tests | 🟠 High |
| 5 | Full security review | 🟠 High |

---

## 🎯 AFTER THESE FIXES

The service will be:
- ✅ Protected from unauthorized access
- ✅ Safe from path traversal attacks
- ✅ Properly validated and type-safe
- ✅ Rate-limited against abuse
- ✅ Production-ready for MVP

Then you can proceed with Week 2 (WhatsApp integration).

---

_This is a MANDATORY action list. Do not skip any item._
