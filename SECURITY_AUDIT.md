# 🔴 CRITICAL SECURITY AUDIT - notify-core

**Date:** 2025-11-11
**Reviewer:** Code Audit
**Status:** ⚠️ **MULTIPLE CRITICAL VULNERABILITIES FOUND**

---

## 🚨 CRITICAL SEVERITY (Must Fix Immediately)

### 1. **NO AUTHENTICATION ON /send ENDPOINT**
**File:** `cmd/server/main.go:48`
**Severity:** 🔴 **CRITICAL**

```go
app.Post("/send", func(c *fiber.Ctx) error {
    // NO AUTH CHECK - Anyone can use this!
```

**Impact:**
- Public endpoint with NO authentication
- Anyone on the internet can send emails using your SMTP credentials
- Can be abused for spam, phishing, or harassment
- Could result in SMTP account suspension
- Potential legal liability

**Attack Scenario:**
```bash
# Attacker can send unlimited emails
curl -X POST http://your-server:8080/send -d '{
  "to": "victim@example.com",
  "channel": "email",
  "template": "welcome",
  "data": {"malicious": "content"}
}'
```

**Fix Required:**
- Add API key authentication
- Add JWT/bearer token validation
- Add per-tenant authentication

---

### 2. **PATH TRAVERSAL VULNERABILITY**
**File:** `internal/email/adapter.go:80`
**Severity:** 🔴 **CRITICAL**

```go
templatePath := filepath.Join(a.templates.Dir, templateName+".html")
tmpl, err := template.ParseFiles(templatePath)
```

**Impact:**
- Attacker can read arbitrary files from the server
- No validation that `templateName` doesn't contain "../"
- Can expose secrets, source code, `/etc/passwd`, etc.

**Attack Scenario:**
```bash
curl -X POST http://your-server:8080/send -d '{
  "template": "../../../etc/passwd",
  ...
}'
```

**Fix Required:**
- Validate template name contains only alphanumeric + hyphen/underscore
- Use allowlist of valid template names
- Check that resolved path is within templates directory

---

### 3. **ARCHITECTURE: BROKEN TYPE SYSTEM**
**Files:** `internal/notify/service.go:30-34`, `internal/email/adapter.go:45`
**Severity:** 🔴 **CRITICAL** (Design Flaw)

```go
// Adapter interface says:
type Adapter interface {
    Send(ctx context.Context, req *SendRequest) error
}

// But email.Adapter implements:
func (a *Adapter) Send(ctx context.Context, req interface{}) error
```

**Impact:**
- Adapter interface contract is broken
- Type safety is completely lost
- Runtime type assertion failures
- Anonymous struct workaround in service.go:82-96 is fragile
- Makes code unmaintainable and error-prone

**Current Workaround (Bad):**
```go
// internal/notify/service.go:82-96
adapterReq = &struct {
    To       string
    Channel  interface{}
    Template string
    Subject  string
    Data     map[string]interface{}
    From     string
}{...}
```

**Fix Required:**
- Redesign adapter interface to accept `interface{}`
- OR use generic request type
- OR create channel-specific request types
- Remove fragile type conversions

---

## 🟠 HIGH SEVERITY (Fix Before Production)

### 4. **ERROR MESSAGE EXPOSURE**
**File:** `cmd/server/main.go:56-59`
**Severity:** 🟠 **HIGH**

```go
if err := notifyService.Send(c.Context(), &req); err != nil {
    return c.Status(500).JSON(fiber.Map{
        "error": err.Error(), // Exposes internal errors!
    })
}
```

**Impact:**
- Leaks SMTP credentials in error messages
- Exposes file paths and internal structure
- Reveals template rendering errors with user data
- Information disclosure vulnerability

**Example Leaked Error:**
```
"error": "failed to send via email: smtp error: 535 Authentication failed for user@example.com with password abc123..."
```

**Fix Required:**
- Return generic error messages to client
- Log detailed errors server-side only
- Use error codes instead of messages

---

### 5. **CONFIGURATION VALIDATION NEVER CALLED**
**File:** `cmd/server/main.go:16-19`, `config/config.go:115-120`
**Severity:** 🟠 **HIGH**

```go
cfg, err := config.Load()
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}
// cfg.Validate() is NEVER called!
```

**Impact:**
- Service starts with invalid SMTP configuration
- Fails only at runtime when trying to send
- Poor user experience
- Debugging nightmare

**Fix Required:**
```go
cfg, err := config.Load()
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}
if err := cfg.Validate(); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

---

### 6. **NO INPUT VALIDATION**
**File:** `cmd/server/main.go:48-54`
**Severity:** 🟠 **HIGH**

```go
var req notify.SendRequest
if err := c.BodyParser(&req); err != nil {
    return c.Status(400).JSON(fiber.Map{
        "error": "Invalid request body",
    })
}
// NO VALIDATION of email, template name, data size, etc.
```

**Impact:**
- Invalid email addresses accepted
- Template names not sanitized (enables path traversal)
- No limits on subject/data size (DoS risk)
- Can cause panics or undefined behavior

**Missing Validations:**
- Email format validation (regex)
- Template name whitelist
- Subject length limit (< 998 chars per RFC)
- Data payload size limit
- Required fields check

**Fix Required:**
- Add comprehensive input validation
- Use validation library (e.g., go-playground/validator)
- Sanitize all user inputs

---

### 7. **NO RATE LIMITING**
**File:** `cmd/server/main.go`
**Severity:** 🟠 **HIGH**

**Impact:**
- Endpoint can be hammered with requests
- Abuse for spam campaigns
- SMTP quota exhaustion
- High costs (if using paid SMTP service)
- Potential blacklisting of SMTP server

**Fix Required:**
- Add rate limiting middleware
- Implement per-IP limits
- Add per-API-key limits (after adding auth)
- Consider queue-based processing

---

### 8. **DOCKER: HARDCODED WEAK POSTGRES PASSWORD**
**File:** `docker-compose.yml:55`
**Severity:** 🟠 **HIGH**

```yaml
environment:
  - POSTGRES_USER=notify
  - POSTGRES_PASSWORD=notify  # Weak password!
```

**Impact:**
- Anyone can access database
- Default credentials are publicly visible
- Violates security best practices

**Fix Required:**
- Use strong passwords from environment variables
- Use Docker secrets
- Never commit credentials to git

---

## 🟡 MEDIUM SEVERITY (Should Fix Soon)

### 9. **CONTEXT NOT RESPECTED**
**File:** `internal/email/adapter.go:45`
**Severity:** 🟡 **MEDIUM**

```go
func (a *Adapter) Send(ctx context.Context, req interface{}) error {
    // ctx is accepted but never used!
    // No cancellation checks
    // SMTP operations can't be cancelled
```

**Impact:**
- Cannot cancel long-running SMTP operations
- Context deadline/timeout ignored
- Resource leaks on cancellation

**Fix Required:**
- Use context-aware SMTP client
- Check ctx.Done() before operations
- Pass context through to SMTP layer

---

### 10. **NO TIMEOUT ON SMTP**
**File:** `internal/email/adapter.go:121`
**Severity:** 🟡 **MEDIUM**

```go
err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
```

**Impact:**
- SMTP connection can hang indefinitely
- No timeout on TCP connection
- Can exhaust goroutines

**Fix Required:**
- Use context with timeout
- Set TCP dialer timeout
- Add overall operation timeout

---

### 11. **SILENT TEMPLATE ERRORS**
**File:** `internal/email/adapter.go:83-85`
**Severity:** 🟡 **MEDIUM**

```go
tmpl, err := template.ParseFiles(templatePath)
if err != nil {
    // If template file doesn't exist, use a simple default
    return a.renderSimpleTemplate(data)
}
```

**Impact:**
- Template parsing errors silently swallowed
- Users don't know template is broken
- Sends generic email instead of intended one
- No logging of failure

**Fix Required:**
- Log template parsing failures
- Return error if template expected but missing
- Don't silently fallback

---

### 12. **CORS TOO PERMISSIVE**
**File:** `cmd/server/main.go:28`
**Severity:** 🟡 **MEDIUM**

```go
app.Use(cors.New()) // Allows all origins!
```

**Impact:**
- Any website can make requests
- CSRF potential (though POST needs auth first)

**Fix Required:**
```go
app.Use(cors.New(cors.Config{
    AllowOrigins: "https://yourdomain.com",
    AllowMethods: "POST",
}))
```

---

### 13. **NO GRACEFUL SHUTDOWN**
**File:** `cmd/server/main.go:75`
**Severity:** 🟡 **MEDIUM**

```go
if err := app.Listen(":" + port); err != nil {
    log.Fatalf("Failed to start server: %v", err)
}
```

**Impact:**
- In-flight requests dropped on SIGTERM
- Docker stop takes 10s (forced kill)
- Emails may be partially sent

**Fix Required:**
- Handle SIGTERM/SIGINT
- Wait for in-flight requests
- Close SMTP connections gracefully

---

### 14. **TEMPLATE EXECUTION ERROR LEAKS DATA**
**File:** `internal/email/adapter.go:89-91`
**Severity:** 🟡 **MEDIUM**

```go
if err := tmpl.Execute(&buf, data); err != nil {
    return "", err // Error may contain user data!
}
```

**Impact:**
- Template execution errors may include user-provided data
- Information disclosure in error messages

**Fix Required:**
- Wrap error without exposing data
- Log full error server-side only

---

### 15. **NO RETRY LOGIC**
**File:** `internal/email/adapter.go:121`
**Severity:** 🟡 **MEDIUM**

**Impact:**
- Transient SMTP errors cause complete failure
- No retry for network hiccups
- Poor reliability

**Fix Required:**
- Implement exponential backoff
- Retry transient SMTP errors (4xx codes)
- Don't retry permanent errors (5xx)

---

### 16. **NO STRUCTURED LOGGING**
**File:** All files
**Severity:** 🟡 **MEDIUM**

**Impact:**
- Can't audit who sent what
- Compliance issues (GDPR, etc.)
- Debugging is difficult
- No request tracing

**Fix Required:**
- Add structured logging (zap, zerolog)
- Log: request ID, recipient, template, timestamp
- Don't log email content (PII)

---

## 🟢 LOW SEVERITY (Nice to Have)

### 17. **TEMPLATE CACHING**
**File:** `internal/email/adapter.go:82`
**Severity:** 🟢 **LOW**

**Impact:**
- Templates re-parsed on every send
- Performance issue for high volume

**Fix Required:**
- Cache parsed templates
- Watch for template file changes
- Reload on modification

---

### 18. **HEALTHCHECK DOESN'T CHECK SMTP**
**File:** `cmd/server/main.go:42`, `docker-compose.yml:45`
**Severity:** 🟢 **LOW**

```go
app.Get("/health", func(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{
        "status": "healthy", // Always healthy!
    })
})
```

**Impact:**
- Health check doesn't verify SMTP connectivity
- Service may be "healthy" but unable to send emails

**Fix Required:**
- Add SMTP connection test
- Check template directory exists
- Return detailed health status

---

### 19. **DOCKER: UNNECESSARY POSTGRES DEPENDENCY**
**File:** `docker-compose.yml:42-43`
**Severity:** 🟢 **LOW**

```yaml
depends_on:
  - postgres  # But code doesn't use DB!
```

**Impact:**
- Unnecessary coupling
- Slower startup
- Resource waste

**Fix Required:**
- Remove depends_on until DB is actually used
- Make postgres optional

---

### 20. **MISSING go.sum FILE**
**File:** Project root
**Severity:** 🟢 **LOW**

**Impact:**
- No dependency verification
- Reproducible builds not guaranteed

**Fix Required:**
```bash
go mod tidy
git add go.sum
```

---

## 📊 SUMMARY

| Severity | Count | Must Fix Before |
|----------|-------|-----------------|
| 🔴 Critical | 3 | **DO NOT DEPLOY** |
| 🟠 High | 5 | Production |
| 🟡 Medium | 9 | Public beta |
| 🟢 Low | 3 | v1.0 |

---

## ✅ WHAT'S GOOD

1. **Clean Architecture** - Good separation of concerns
2. **Adapter Pattern** - Extensible design for new channels
3. **Docker Support** - Easy deployment
4. **Multi-stage Build** - Small image size
5. **Tests Exist** - 505 lines of test code
6. **Configuration** - Centralized config management
7. **Templates** - Professional HTML email templates

---

## 🎯 IMMEDIATE ACTION ITEMS (Before ANY deployment)

1. **Add authentication** to /send endpoint
2. **Fix path traversal** vulnerability
3. **Fix broken type system** (adapter interface)
4. **Add input validation** (email, template name, limits)
5. **Hide internal errors** from API responses
6. **Call config.Validate()** on startup
7. **Add rate limiting**
8. **Fix Docker postgres password**

---

## 🔧 RECOMMENDED FIXES (Priority Order)

### Week 1.5 - Critical Security Fixes
1. Add API key authentication
2. Fix path traversal vulnerability
3. Add input validation
4. Fix error exposure
5. Call config validation
6. Add rate limiting

### Week 2 - Quality & Reliability
7. Fix type system (adapter interface)
8. Add context support to SMTP
9. Add SMTP timeout
10. Add graceful shutdown
11. Add structured logging
12. Add retry logic

### Week 3 - Production Hardening
13. Implement template caching
14. Fix CORS configuration
15. Improve healthcheck
16. Remove unnecessary dependencies
17. Add monitoring/metrics

---

## 📝 TESTING GAPS

Current tests do NOT cover:
- Path traversal attacks
- Authentication bypass
- Rate limiting
- Input validation
- Error message exposure
- SMTP timeout scenarios
- Graceful shutdown
- Context cancellation

**Required:** Add security-focused integration tests

---

**VERDICT:** ⚠️ **NOT PRODUCTION READY**

The foundation is solid, but **critical security vulnerabilities** must be fixed before any public deployment.

---

_Generated: 2025-11-11_
