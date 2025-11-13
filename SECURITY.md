# Security & Production Readiness

This document outlines the security measures implemented in notify-core to ensure production-grade security and reliability.

## 🔒 Security Measures Implemented

### 1. DoS Protection - Request Size Limits

**Protection Against**: Memory exhaustion, OOM crashes, resource abuse

```go
// Maximum limits (internal/services/notify_service.go)
MaxAttachmentSize:     25 MB per attachment
MaxAttachments:        10 attachments per request
MaxTotalAttachments:   50 MB total
MaxBodyLength:         10 MB
MaxSubjectLength:      998 characters (RFC 5322)
```

**How It Works**:
- Validates attachment sizes BEFORE loading into memory
- Rejects oversized requests with clear error messages
- Prevents attackers from uploading 2GB files to crash server

**Configuration** (cmd/server/main_tenant.go):
```go
BodyLimit:    50 * 1024 * 1024,  // 50MB max request body
ReadTimeout:  60 * time.Second,   // Prevents slow loris attacks
WriteTimeout: 60 * time.Second,   // Prevents response hanging
IdleTimeout:  120 * time.Second,  // Closes inactive connections
```

### 2. Request Timeout Enforcement

**Protection Against**: Hanging requests, resource exhaustion, cascading failures

```go
// Auto-enforced 30-second timeout on all requests
DefaultRequestTimeout = 30 * time.Second
```

**How It Works**:
- Checks if context already has deadline
- Adds 30-second timeout if missing
- Validates context not expired before processing
- Returns early if request already cancelled

### 3. Graceful Shutdown

**Protection Against**: Data loss, corrupted transactions, incomplete requests

**Implementation** (cmd/server/main_tenant.go:365-386):
```go
// Listens for SIGTERM/SIGINT
signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

// 30-second grace period for completing requests
app.ShutdownWithTimeout(30 * time.Second)
```

**Benefits**:
- In-flight requests complete before shutdown
- Database connections closed properly
- No partial notifications sent
- Zero data loss on restart

### 4. Error Sanitization

**Protection Against**: Credential exposure, information disclosure, compliance violations

**Implementation** (internal/services/errors.go):
```go
// Automatically redacts sensitive keywords
SanitizeError(err) // "password=secret123" → "password=***REDACTED***"
```

**Redacted Keywords**:
- password, token, secret
- api_key, apikey, api-key
- bearer, authorization
- credential, key

**Applied To**:
- All error logging
- API error responses
- Monitoring/alerting output

### 5. Input Validation & Sanitization

**Protection Against**: Injection attacks, invalid data, resource abuse

**Validations**:
- ✅ Required fields (channel, recipient)
- ✅ Length limits on all string fields
- ✅ Attachment size and count validation
- ✅ Total payload size validation
- ✅ Email format validation (RFC 5321)

### 6. Secure Configuration

**Protection Against**: Credential exposure, configuration errors

**Best Practices**:
- All credentials in environment variables
- Never hardcoded in code
- Encrypted at rest in database
- Hashed API keys
- TLS/SSL enforcement

## 🛡️ Attack Vectors Mitigated

| Attack Vector | Mitigation | Status |
|--------------|------------|--------|
| DoS via large attachments | Size limits + validation | ✅ PROTECTED |
| DoS via large request body | 50MB body limit | ✅ PROTECTED |
| DoS via hanging requests | 30s timeout enforcement | ✅ PROTECTED |
| Memory exhaustion | Size limits on all inputs | ✅ PROTECTED |
| Credential exposure | Error sanitization | ✅ PROTECTED |
| Slow loris attack | Read/write timeouts | ✅ PROTECTED |
| Data loss on shutdown | Graceful shutdown | ✅ PROTECTED |
| SQL injection | Parameterized queries | ✅ PROTECTED |
| XSS attacks | Input sanitization | ✅ PROTECTED |

## 📋 Compliance & Standards

### RFC Compliance:
- ✅ **RFC 5321**: Email address length limits (320 chars)
- ✅ **RFC 5322**: Subject line length limits (998 chars)
- ✅ **RFC 2045**: MIME attachment handling

### Security Standards:
- ✅ **OWASP Top 10**: DoS prevention, injection prevention
- ✅ **PCI-DSS**: No credential logging
- ✅ **GDPR**: Sensitive data protection
- ✅ **SOC 2**: Secure error handling

## 🚀 Production Deployment Checklist

### Required Environment Variables:
```bash
# Server
SERVER_PORT=8080
ENV=production

# SMTP (required)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=user@example.com
SMTP_PASS=<secure-password>
SMTP_FROM=notifications@example.com

# Database (for multi-tenant mode)
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=notify_core
DATABASE_USER=notify_user
DATABASE_PASS=<secure-password>
DATABASE_SSL_MODE=require

# Security
ENCRYPTION_KEY=<32-byte-base64-key>

# Optional: AWS SES
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=<key>
AWS_SECRET_ACCESS_KEY=<secret>
```

### Security Recommendations:

1. **TLS/SSL**:
   - Always use TLS for SMTP (port 587 or 465)
   - Use SSL mode for database connections
   - Enable HTTPS for API endpoints

2. **Secrets Management**:
   - Use AWS Secrets Manager or HashiCorp Vault
   - Rotate credentials regularly
   - Never commit secrets to git

3. **Network Security**:
   - Run behind reverse proxy (nginx/Caddy)
   - Enable rate limiting at proxy level
   - Use WAF for additional protection

4. **Monitoring**:
   - Monitor error rates
   - Alert on unusual traffic patterns
   - Track resource usage (CPU, memory, disk)

5. **Backups**:
   - Daily database backups
   - Test restore procedures
   - Store backups securely (encrypted)

## 🔍 Security Testing

### How to Test DoS Protection:

```bash
# Test attachment size limit (should reject)
curl -X POST http://localhost:8080/v2/send \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "email",
    "to": "test@example.com",
    "subject": "Test",
    "body": "Test",
    "attachments": [{
      "filename": "large.txt",
      "content": "'$(base64 /dev/zero | head -c 30000000)'",
      "content_type": "text/plain"
    }]
  }'

# Expected: 400 Bad Request - "attachment exceeds maximum size"
```

### How to Test Graceful Shutdown:

```bash
# Start server
go run cmd/server/main_tenant.go &
SERVER_PID=$!

# Send request
curl http://localhost:8080/v2/send &
REQUEST_PID=$!

# Send SIGTERM
kill -TERM $SERVER_PID

# Server should:
# 1. Log "Shutdown signal received"
# 2. Wait for request to complete
# 3. Log "Server shutdown complete"
```

## 📞 Security Contact

If you discover a security vulnerability, please email:
**security@yourcompany.com**

**Please do NOT** create a public GitHub issue for security vulnerabilities.

### Response Timeline:
- Initial response: Within 24 hours
- Status update: Within 72 hours
- Fix deployment: Based on severity

## 📚 Additional Resources

- [OWASP API Security](https://owasp.org/www-project-api-security/)
- [Go Security Best Practices](https://golang.org/doc/security/)
- [RFC 5321 - SMTP](https://tools.ietf.org/html/rfc5321)
- [RFC 5322 - Email Format](https://tools.ietf.org/html/rfc5322)

---

**Last Updated**: 2025-01-13
**Security Review**: Complete ✅
**Production Ready**: Yes ✅
