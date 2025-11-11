# Day 15: Aggressive Code Review

## Executive Summary

**Review Date**: 2025-11-11
**Scope**: Database layer, migrations, tenant models, repositories, services
**Severity Levels**: 🔴 Critical | 🟠 High | 🟡 Medium | 🟢 Low | ℹ️ Info

---

## Critical Issues Found: 0 🔴

No critical breaking issues found.

---

## High Priority Issues: 3 🟠

### 🟠 Issue #1: Missing Database Connection Config Validation

**File**: `internal/config/config.go`
**Location**: `Validate()` function (line 115-159)

**Problem**:
The `Validate()` function checks SMTP config but **does NOT validate database configuration**. This means the application could start with invalid DB credentials and fail at runtime.

**Current Code**:
```go
func (c *Config) Validate() error {
    // SMTP validation
    if c.SMTP.Host == "" {
        return fmt.Errorf("SMTP_HOST is required")
    }
    // ... SMTP checks ...

    // NO DATABASE VALIDATION! ❌

    return nil
}
```

**Impact**:
- Application starts successfully but fails on first DB operation
- Poor user experience - errors happen at runtime, not startup
- Difficult to debug in production

**Recommendation**:
```go
// Add database validation
if c.Database.Host == "" {
    return fmt.Errorf("DB_HOST is required")
}
if c.Database.Port < 1 || c.Database.Port > 65535 {
    return fmt.Errorf("invalid DB_PORT: must be between 1-65535")
}
if c.Database.User == "" {
    return fmt.Errorf("DB_USER is required")
}
if c.Database.DBName == "" {
    return fmt.Errorf("DB_NAME is required")
}
if c.Database.MaxConns < 1 {
    return fmt.Errorf("DB_MAX_CONNS must be at least 1")
}
if c.Database.MaxIdle < 0 || c.Database.MaxIdle > c.Database.MaxConns {
    return fmt.Errorf("DB_MAX_IDLE must be between 0 and DB_MAX_CONNS")
}
```

---

### 🟠 Issue #2: API Keys Stored in Plain Text

**File**: `internal/database/migrations.go`
**Location**: Migration 1 - tenants table (line 164-201)

**Problem**:
Tenant API keys are stored in **plain text** in the database. This is a **security vulnerability**.

**Current Schema**:
```sql
api_key VARCHAR(255) NOT NULL UNIQUE,  -- ❌ Plain text!
```

**Impact**:
- If database is compromised, all tenant API keys are exposed
- No defense-in-depth
- Violates security best practices (OWASP Top 10)

**Recommendation**:
1. **Hash API keys** before storing (use bcrypt or argon2)
2. Return plain-text key only once during creation
3. Store only the hash for validation

```go
// In repository.Create():
import "golang.org/x/crypto/bcrypt"

// Generate plain-text key
plainKey := uuid.New().String()

// Hash it
hashedKey, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
if err != nil {
    return nil, err
}

// Store hash
tenant.APIKey = string(hashedKey)

// Return plain key to user (only time they see it)
tenant.PlainAPIKey = plainKey // Add this field to response
```

```go
// In repository.GetByAPIKey():
// Compare provided key with hash
err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(providedKey))
```

**Alternative**: Use HMAC with a secret key (faster than bcrypt for API keys)

---

### 🟠 Issue #3: SMTP Passwords Stored in Plain Text

**File**: `internal/database/migrations.go`
**Location**: Migration 1 - tenants table (line 177)

**Problem**:
SMTP passwords, WhatsApp tokens, and SMS API keys are stored in **plain text**.

**Current Schema**:
```sql
smtp_password VARCHAR(255),  -- ❌ Plain text!
wa_token TEXT,               -- ❌ Plain text!
sms_api_key TEXT,            -- ❌ Plain text!
```

**Impact**:
- Database breach exposes all tenant credentials
- Regulatory compliance issues (GDPR, PCI-DSS)
- Cannot pass security audit

**Recommendation**:
1. **Encrypt sensitive credentials at rest** using AES-256
2. Use application-level encryption with a master key
3. Store master key in environment variable or key management service (AWS KMS, HashiCorp Vault)

```go
// Example encryption layer
func EncryptSensitive(plaintext, masterKey string) (string, error) {
    // Use crypto/aes with GCM mode
    // Return base64-encoded ciphertext
}

func DecryptSensitive(ciphertext, masterKey string) (string, error) {
    // Decrypt and return plaintext
}
```

**Schema remains same but values are encrypted**:
```sql
smtp_password VARCHAR(255),  -- Stores: "encrypted:AES256:base64data"
```

---

## Medium Priority Issues: 4 🟡

### 🟡 Issue #4: Missing Context Timeout in Database Layer

**File**: `internal/database/database.go`
**Location**: `New()` function (line 48-53)

**Problem**:
The `Ping()` call uses the passed context but doesn't set a timeout. If DB is unreachable, could hang indefinitely.

**Current Code**:
```go
if err := db.PingContext(ctx); err != nil {
    return nil, fmt.Errorf("failed to ping database: %w", err)
}
```

**Recommendation**:
```go
pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

if err := db.PingContext(pingCtx); err != nil {
    return nil, fmt.Errorf("failed to ping database: %w", err)
}
```

---

### 🟡 Issue #5: No SQL Injection Protection Documentation

**File**: `internal/repository/tenant.go`
**Location**: All query functions

**Problem**:
While the code correctly uses parameterized queries (`$1`, `$2`), there's **no documentation** warning future developers about SQL injection risks.

**Current Code**:
```go
query := `SELECT ... FROM tenants WHERE id = $1`  // ✅ Safe
err := r.db.QueryRowContext(ctx, query, id).Scan(...)
```

**Recommendation**:
Add package-level documentation:

```go
// Package repository provides data access layer for the notify-core database.
//
// SECURITY: All queries MUST use parameterized statements ($1, $2, etc.) to prevent SQL injection.
// NEVER concatenate user input into SQL strings.
//
// ✅ SAFE:   query := "SELECT * FROM users WHERE id = $1"
// ❌ UNSAFE: query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userInput)
package repository
```

---

### 🟡 Issue #6: Missing Index on `notifications.tenant_id + created_at`

**File**: `internal/database/migrations.go`
**Location**: Migration 3 - notifications table (line 259-263)

**Problem**:
The query pattern for analytics will be: "Get notifications for tenant X in date range Y".
Current indexes:
- `idx_notifications_tenant` on `tenant_id` ✅
- `idx_notifications_created_at` on `created_at` ✅

But PostgreSQL may not efficiently use both indexes for a combined query.

**Recommendation**:
Add composite index:

```sql
CREATE INDEX idx_notifications_tenant_created ON notifications(tenant_id, created_at DESC);
```

This optimizes queries like:
```sql
SELECT * FROM notifications
WHERE tenant_id = 1
  AND created_at >= '2025-11-01'
  AND created_at < '2025-12-01'
ORDER BY created_at DESC;
```

---

### 🟡 Issue #7: No Connection Leak Protection

**File**: `internal/database/database.go`
**Location**: Connection pooling (line 55-57)

**Problem**:
`ConnMaxLifetime` is set to 5 minutes, but there's no `ConnMaxIdleTime`. Long-lived idle connections could accumulate.

**Current Code**:
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
// Missing: SetConnMaxIdleTime ❌
```

**Recommendation**:
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
db.SetConnMaxIdleTime(30 * time.Second) // Close idle connections after 30s
```

---

## Low Priority Issues: 5 🟢

### 🟢 Issue #8: Inconsistent Error Messages

**File**: `internal/repository/tenant.go`

**Problem**:
Error messages are inconsistent in format:
- Line 84: `"tenant not found: %d"`
- Line 102: `"tenant not found or inactive"` (no ID)

**Recommendation**:
Standardize error messages:
```go
return nil, fmt.Errorf("tenant not found or inactive: api_key=%s", apiKey)
```

---

### 🟢 Issue #9: Missing Struct Tags for Logging

**File**: `internal/models/tenant.go`

**Problem**:
Tenant struct has JSON tags but no custom string representation. Logging a tenant might accidentally log sensitive data.

**Current Code**:
```go
type Tenant struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    APIKey string `json:"api_key,omitempty"`  // ❌ Logged as-is!
    SMTPPassword string `json:"smtp_password,omitempty"`  // ❌ Sensitive!
}
```

**Recommendation**:
Add custom String() method:

```go
func (t *Tenant) String() string {
    return fmt.Sprintf("Tenant{ID:%d, Name:%s, APIKey:[REDACTED], Active:%t}",
        t.ID, t.Name, t.Active)
}
```

Or use structured logging with explicit fields:
```go
logger.Info().
    Int("tenant_id", tenant.ID).
    Str("tenant_name", tenant.Name).
    // Don't log APIKey or passwords!
    Msg("Tenant created")
```

---

### 🟢 Issue #10: No Rate Limiting on API Key Validation

**File**: `internal/services/tenant_service.go`
**Location**: `ValidateAPIKey()` (line 77-90)

**Problem**:
No protection against API key brute-force attacks. An attacker could try thousands of API keys.

**Recommendation**:
- Add rate limiting per IP address
- Add exponential backoff after N failed attempts
- Log failed validation attempts for monitoring

```go
// Use a rate limiter library like golang.org/x/time/rate
limiter := rate.NewLimiter(rate.Every(time.Second), 10) // 10 req/sec

if !limiter.Allow() {
    return nil, fmt.Errorf("rate limit exceeded")
}
```

---

### 🟢 Issue #11: Missing Tenant Name Validation

**File**: `internal/services/tenant_service.go`
**Location**: `CreateTenant()` (line 23-33)

**Problem**:
Only checks if name is empty, but doesn't validate format. Could allow problematic names.

**Current Validation**:
```go
if req.Name == "" {
    return nil, fmt.Errorf("tenant name is required")
}
```

**Recommendation**:
```go
if req.Name == "" {
    return nil, fmt.Errorf("tenant name is required")
}

// Validate format
if len(req.Name) > 255 {
    return nil, fmt.Errorf("tenant name too long (max 255 characters)")
}

// Only allow alphanumeric, hyphens, underscores
validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
if !validName.MatchString(req.Name) {
    return nil, fmt.Errorf("tenant name must contain only letters, numbers, hyphens, and underscores")
}
```

---

### 🟢 Issue #12: No Unique API Key Retry Logic

**File**: `internal/repository/tenant.go`
**Location**: `Create()` (line 21-24)

**Problem**:
API key is generated using `uuid.New()`, which is statistically unique but theoretically could collide (extremely rare).

**Current Code**:
```go
apiKey := uuid.New().String()
// If collision happens, INSERT will fail with unique constraint error
```

**Recommendation**:
Add retry logic (defensive programming):

```go
const maxRetries = 3
var apiKey string
var err error

for i := 0; i < maxRetries; i++ {
    apiKey = uuid.New().String()

    // Try to insert
    err = r.db.QueryRowContext(ctx, query, ...).Scan(...)

    if err == nil {
        break // Success
    }

    // Check if it's a unique constraint error
    if strings.Contains(err.Error(), "unique constraint") && strings.Contains(err.Error(), "api_key") {
        continue // Retry with new UUID
    }

    return nil, err // Different error, don't retry
}
```

---

## Informational Notes: 3 ℹ️

### ℹ️ Note #1: Migration Rollback Considerations

**File**: `internal/database/migrations.go`

**Observation**:
Down migrations use `DROP TABLE ... CASCADE`, which will delete all dependent data.

**Consideration**:
- In production, you may want to export data before rolling back
- Consider adding a "safe rollback" that renames tables instead of dropping
- Document rollback procedure in runbook

---

### ℹ️ Note #2: No Database Transaction Isolation Level Set

**File**: `internal/database/database.go`

**Observation**:
PostgreSQL defaults to `READ COMMITTED` isolation level. This is generally fine, but be aware:
- Concurrent updates to same tenant could have race conditions
- Consider using `SELECT ... FOR UPDATE` in high-concurrency scenarios

**Example Race Condition**:
```
User A: SELECT api_key FROM tenants WHERE id = 1
User B: UPDATE tenants SET api_key = 'new' WHERE id = 1
User A: Uses old api_key (stale read)
```

---

### ℹ️ Note #3: No Database Backup Strategy

**Observation**:
Code is production-ready but there's no mention of backup strategy.

**Recommendation**:
Document backup procedures:
- PostgreSQL `pg_dump` for logical backups
- Filesystem snapshots for physical backups
- Retention policy (7 days, 30 days, 1 year)
- Disaster recovery plan

---

## Security Checklist

| Item | Status | Notes |
|------|--------|-------|
| SQL Injection Protection | ✅ | Parameterized queries used everywhere |
| Sensitive Data Encryption | ❌ | Passwords/tokens stored in plain text |
| API Key Hashing | ❌ | API keys stored in plain text |
| Input Validation | ⚠️ | Basic validation, needs improvement |
| Error Messages | ✅ | Don't leak internal details |
| Connection Pooling | ✅ | Properly configured |
| Transaction Safety | ✅ | Used in migrations |
| Logging Security | ⚠️ | Could accidentally log secrets |
| Rate Limiting | ❌ | Not implemented |
| CSRF Protection | N/A | API-only (no forms) |

---

## Performance Checklist

| Item | Status | Notes |
|------|--------|-------|
| Indexes on Foreign Keys | ✅ | All FKs indexed |
| Indexes on Query Fields | ⚠️ | Missing composite index |
| Connection Pooling | ✅ | 25 max, 5 idle |
| Query Parameterization | ✅ | All queries use $1, $2 |
| Batch Operations | N/A | Not yet needed |
| N+1 Query Prevention | ✅ | Single queries, no loops |
| JSONB Indexing | ⏳ | Consider GIN index on JSONB fields later |

---

## Code Quality Checklist

| Item | Status | Notes |
|------|--------|-------|
| Error Handling | ✅ | All errors wrapped with context |
| Logging | ✅ | Structured logging with zerolog |
| Testing | ✅ | Comprehensive unit tests added |
| Documentation | ✅ | README and inline comments |
| Type Safety | ✅ | No interface{} abuse |
| Context Propagation | ✅ | ctx passed everywhere |
| Null Safety | ✅ | Pointers used for optional updates |
| Code Duplication | ✅ | Minimal duplication |

---

## Recommendations Priority

### Must Fix Before Production (Priority 1)
1. 🟠 Issue #2: Hash API keys before storing
2. 🟠 Issue #3: Encrypt sensitive credentials (SMTP passwords, tokens)
3. 🟠 Issue #1: Add database config validation

### Should Fix Soon (Priority 2)
4. 🟡 Issue #4: Add context timeout to Ping()
5. 🟡 Issue #6: Add composite index for tenant+date queries
6. 🟡 Issue #7: Set ConnMaxIdleTime
7. 🟢 Issue #11: Add tenant name validation

### Nice to Have (Priority 3)
8. 🟢 Issue #10: Add rate limiting
9. 🟢 Issue #9: Add String() method to redact secrets
10. 🟡 Issue #5: Add SQL injection warning in package docs
11. 🟢 Issue #8: Standardize error messages
12. 🟢 Issue #12: Add UUID collision retry logic

---

## Test Coverage Analysis

### What Was Tested ✅
- ✅ Repository CRUD operations
- ✅ Service business logic
- ✅ Model helper methods
- ✅ Migration structure validation
- ✅ Edge cases (empty values, invalid IDs, database errors)
- ✅ Mock database interactions

### What Needs Testing ⏳
- ⏳ Integration tests with real PostgreSQL
- ⏳ Migration Up/Down with actual database
- ⏳ Concurrent access to same tenant
- ⏳ Connection pool exhaustion scenarios
- ⏳ Large dataset performance (1000+ tenants)
- ⏳ Transaction rollback scenarios

---

## Conclusion

**Overall Assessment**: 🟢 **GOOD** - Code is solid with some security improvements needed

### Strengths
1. ✅ **Excellent SQL injection protection** - all queries parameterized
2. ✅ **Good separation of concerns** - layers are well-defined
3. ✅ **Comprehensive testing** - unit tests cover major scenarios
4. ✅ **Type safety** - proper use of Go types
5. ✅ **Documentation** - well-documented with README

### Weaknesses
1. ❌ **Security**: Sensitive data stored in plain text
2. ❌ **Security**: API keys not hashed
3. ⚠️ **Validation**: Config validation incomplete
4. ⚠️ **Performance**: Missing composite index for analytics

### Verdict
The code is **production-ready** after addressing the 3 high-priority security issues:
- Hash API keys (Issue #2)
- Encrypt sensitive credentials (Issue #3)
- Add database config validation (Issue #1)

Everything else can be addressed in follow-up PRs.

---

## Files Reviewed

- ✅ `internal/database/database.go`
- ✅ `internal/database/migrations.go`
- ✅ `internal/models/tenant.go`
- ✅ `internal/repository/tenant.go`
- ✅ `internal/services/tenant_service.go`
- ✅ `internal/config/config.go`
- ✅ `cmd/migrate/main.go`
- ✅ `.env.example`

---

## Action Items

- [ ] Fix Issue #1: Add DB config validation
- [ ] Fix Issue #2: Implement API key hashing
- [ ] Fix Issue #3: Implement credential encryption
- [ ] Fix Issue #4: Add Ping() timeout
- [ ] Fix Issue #6: Add composite index
- [ ] Fix Issue #7: Set ConnMaxIdleTime
- [ ] Document backup/recovery strategy
- [ ] Run integration tests with real PostgreSQL
- [ ] Security audit before production deployment
