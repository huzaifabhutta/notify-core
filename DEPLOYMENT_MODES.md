# Deployment Modes - notify-core

notify-core supports **multiple deployment modes** to suit different use cases, from simple single-tenant setups to complex multi-tenant architectures.

---

## 📊 Overview

| Mode | Binary | Database Required | Multi-Tenant | Best For |
|------|--------|-------------------|--------------|----------|
| **Simple** | `main.go` | ❌ No | ❌ No | Small projects, MVPs, single-tenant |
| **Multi-Tenant** | `main_tenant.go` | ✅ PostgreSQL | ✅ Yes | SaaS, enterprises, multiple customers |

---

## 🚀 Mode 1: Simple (No Database)

**Perfect for**: Small projects, MVPs, personal use, single-tenant applications

### Features
- ✅ No database required
- ✅ Configuration via environment variables
- ✅ Multiple API keys (comma-separated)
- ✅ Global credentials only (SMTP, WhatsApp, SMS)
- ✅ Fast setup (< 5 minutes)
- ✅ Vendor-agnostic (any SMTP, WhatsApp provider)

### What's NOT Included
- ❌ Per-tenant credentials
- ❌ Database-backed authentication
- ❌ Tenant management API
- ❌ Advanced rate limiting per tenant
- ❌ Usage tracking/analytics

### Quick Start

#### 1. Build the simple server
```bash
go build -o notify-simple ./cmd/server/main.go
```

#### 2. Configure via environment variables
```bash
# API Keys (comma-separated: key:tenant_name)
export API_KEYS="abc123:tenant1,def456:tenant2"

# SMTP Configuration (global)
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT=587
export SMTP_USER="your-email@gmail.com"
export SMTP_PASS="your-app-password"
export SMTP_FROM="noreply@yourapp.com"

# WhatsApp Configuration (global, optional)
export WA_TOKEN="your-whatsapp-token"
export WA_PHONE_ID="your-phone-id"
# Optional: Override provider (defaults to Facebook)
# export WA_BASE_URL="https://waba.360dialog.io"
# export WA_API_VERSION="v21.0"

# Server Configuration
export PORT=8080
export ENV=production
```

#### 3. Run the server
```bash
./notify-simple
```

### API Usage

**Send Email** (using any API key from API_KEYS):
```bash
curl -X POST http://localhost:8080/send \
  -H "X-API-Key: abc123" \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "email",
    "to": "user@example.com",
    "subject": "Welcome!",
    "template": "welcome",
    "data": {
      "name": "John Doe"
    }
  }'
```

**Send WhatsApp**:
```bash
curl -X POST http://localhost:8080/send \
  -H "X-API-Key: abc123" \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "whatsapp",
    "to": "+1234567890",
    "template": "order_confirmation",
    "data": {
      "order_id": "12345"
    }
  }'
```

### Rate Limits
- **Global**: 100 requests/minute per IP
- **Per API Key**: 20 requests/minute

### Use Cases
- ✅ Personal projects
- ✅ Internal tools
- ✅ MVPs and prototypes
- ✅ Single-tenant SaaS
- ✅ Microservices (one service = one tenant)
- ✅ Development and testing

### Docker Deployment
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o notify-simple ./cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/notify-simple .
COPY --from=builder /app/templates ./templates

# Environment variables provided at runtime
CMD ["./notify-simple"]
```

```bash
docker build -t notify-simple .
docker run -p 8080:8080 \
  -e API_KEYS="abc123:tenant1" \
  -e SMTP_HOST="smtp.gmail.com" \
  -e SMTP_PORT=587 \
  -e SMTP_USER="your-email@gmail.com" \
  -e SMTP_PASS="your-app-password" \
  notify-simple
```

---

## 🏢 Mode 2: Multi-Tenant (Database-Backed)

**Perfect for**: SaaS applications, enterprises, multiple customers, advanced features

### Features
- ✅ PostgreSQL database required
- ✅ Unlimited tenants
- ✅ Per-tenant credentials (SMTP, WhatsApp, SMS)
- ✅ Global credentials as fallback
- ✅ Tenant management API (CRUD operations)
- ✅ Database-backed authentication
- ✅ Per-tenant rate limiting
- ✅ Encrypted credentials (AES-256-GCM)
- ✅ Hashed API keys (bcrypt)
- ✅ Vendor-agnostic (any SMTP, WhatsApp provider)
- ✅ Automatic database migrations

### What's Included
- ✅ Tenant CRUD API (`/v2/tenants`)
- ✅ Per-tenant configuration
- ✅ Credential encryption at rest
- ✅ API key hashing
- ✅ Per-tenant and per-IP rate limiting
- ✅ Automatic credential fallback (tenant → global)
- ✅ Database migrations
- ✅ Audit trail (created_at, updated_at)

### Quick Start

#### 1. Setup PostgreSQL
```bash
# Using Docker
docker run --name notify-postgres \
  -e POSTGRES_USER=notify \
  -e POSTGRES_PASSWORD=securepassword \
  -e POSTGRES_DB=notify \
  -p 5432:5432 \
  -d postgres:15

# Or use managed PostgreSQL (AWS RDS, Google Cloud SQL, etc.)
```

#### 2. Build the multi-tenant server
```bash
go build -o notify-multitenant ./cmd/server/main_tenant.go
```

#### 3. Configure via environment variables
```bash
# Database Configuration (REQUIRED)
export DB_HOST="localhost"
export DB_PORT=5432
export DB_USER="notify"
export DB_PASSWORD="securepassword"
export DB_NAME="notify"
export DB_SSL_MODE="disable"
export DB_MAX_CONNS=25
export DB_MAX_IDLE=5

# Security Configuration (REQUIRED)
# Generate with: openssl rand -base64 32
export ENCRYPTION_KEY="your-32-character-encryption-key-here-change-this"

# Global SMTP (fallback for tenants without SMTP config)
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT=587
export SMTP_USER="your-email@gmail.com"
export SMTP_PASS="your-app-password"
export SMTP_FROM="noreply@yourapp.com"

# Global WhatsApp (fallback, optional)
export WA_TOKEN="your-whatsapp-token"
export WA_PHONE_ID="your-phone-id"
# Optional: Override provider
# export WA_BASE_URL="https://waba.360dialog.io"
# export WA_API_VERSION="v21.0"

# Server Configuration
export PORT=8080
export ENV=production
```

#### 4. Run migrations (automatic on first start)
```bash
./notify-multitenant
# Migrations run automatically on startup
```

### API Usage

#### Create a Tenant
```bash
curl -X POST http://localhost:8080/v2/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "name": "acme-corp",
    "smtp_host": "smtp.sendgrid.net",
    "smtp_port": 587,
    "smtp_user": "apikey",
    "smtp_password": "SG.xxx",
    "smtp_from": "noreply@acme.com",
    "wa_token": "EAAG...",
    "wa_phone_id": "123456789",
    "wa_base_url": "https://waba.360dialog.io",
    "wa_api_version": "v21.0"
  }'
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "tenant": {
      "id": 1,
      "name": "acme-corp",
      "active": true,
      "created_at": "2025-11-11T10:00:00Z"
    },
    "api_key": "550e8400-e29b-41d4-a716-446655440000"
  },
  "message": "Tenant created successfully. Save the API key - it won't be shown again!"
}
```

#### Get Tenant Info (authenticated)
```bash
curl -X GET http://localhost:8080/v2/tenants/me \
  -H "X-API-Key: 550e8400-e29b-41d4-a716-446655440000"
```

#### Update Tenant (authenticated)
```bash
curl -X PUT http://localhost:8080/v2/tenants/me \
  -H "X-API-Key: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "smtp_host": "smtp.mailgun.org",
    "smtp_user": "postmaster@mg.acme.com"
  }'
```

#### Send Notification (authenticated)
```bash
curl -X POST http://localhost:8080/v2/send \
  -H "X-API-Key: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "email",
    "to": "user@example.com",
    "subject": "Welcome to Acme Corp!",
    "template": "welcome",
    "data": {
      "name": "John Doe",
      "company": "Acme Corp"
    }
  }'
```

**Response**:
```json
{
  "status": "success",
  "data": {
    "message_id": "abc-123-def-456",
    "tenant_id": 1,
    "source": "tenant",
    "channel": "email"
  }
}
```

### Credential Resolution

The system automatically resolves credentials with the following priority:

1. **Tenant-specific credentials** (from database)
2. **Global credentials** (from environment variables - fallback)

**Example**: Tenant A uses SendGrid, Tenant B uses Gmail, Tenant C has no SMTP config (uses global).

### Rate Limits
- **Global**: 100 requests/minute per IP
- **Per Tenant**: 50 requests/minute
- **Per API Key Validation**: 10 requests/second per IP (brute-force protection)

### Use Cases
- ✅ Multi-tenant SaaS applications
- ✅ Enterprise applications with multiple departments
- ✅ White-label notification services
- ✅ Notification-as-a-Service platforms
- ✅ Applications with per-customer credentials
- ✅ Advanced security requirements

### Docker Compose Deployment
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: notify
      POSTGRES_PASSWORD: securepassword
      POSTGRES_DB: notify
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U notify"]
      interval: 10s
      timeout: 5s
      retries: 5

  notify:
    build:
      context: .
      dockerfile: Dockerfile.multitenant
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=notify
      - DB_PASSWORD=securepassword
      - DB_NAME=notify
      - ENCRYPTION_KEY=your-32-character-encryption-key-here
      - SMTP_HOST=smtp.gmail.com
      - SMTP_PORT=587
      - SMTP_USER=your-email@gmail.com
      - SMTP_PASS=your-app-password
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  postgres_data:
```

**Dockerfile.multitenant**:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o notify-multitenant ./cmd/server/main_tenant.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/notify-multitenant .
COPY --from=builder /app/templates ./templates

CMD ["./notify-multitenant"]
```

---

## 🔄 Migration Path

### From Simple → Multi-Tenant

If you start with Simple mode and later need multi-tenancy:

1. **Setup PostgreSQL** database
2. **Switch to multi-tenant binary**:
   ```bash
   go build -o notify-multitenant ./cmd/server/main_tenant.go
   ```
3. **Migrate API keys** to database:
   ```bash
   # For each API key in API_KEYS, create a tenant
   curl -X POST http://localhost:8080/v2/tenants \
     -d '{"name": "tenant1", ...}'
   ```
4. **Update clients** with new API keys from database
5. **Keep global credentials** as fallback

### From Multi-Tenant → Simple

If you need to simplify:

1. **Export tenant credentials** to environment variables
2. **Switch to simple binary**:
   ```bash
   go build -o notify-simple ./cmd/server/main.go
   ```
3. **Set API_KEYS** environment variable
4. **Remove database** dependency

---

## 📋 Feature Comparison

| Feature | Simple Mode | Multi-Tenant Mode |
|---------|-------------|-------------------|
| Database Required | ❌ No | ✅ PostgreSQL |
| Setup Time | < 5 minutes | 15-30 minutes |
| API Keys | Environment variable | Database + Auto-generated |
| Per-Tenant Credentials | ❌ No | ✅ Yes |
| Global Credentials | ✅ Yes | ✅ Yes (fallback) |
| Tenant Management API | ❌ No | ✅ Yes |
| Credential Encryption | ❌ No | ✅ AES-256-GCM |
| API Key Hashing | ❌ No | ✅ bcrypt |
| Rate Limiting | IP + API Key | IP + Tenant + API Key |
| Vendor-Agnostic | ✅ Yes | ✅ Yes |
| Self-Hosted | ✅ Yes | ✅ Yes |
| Scalability | Low-Medium | High |
| Memory Footprint | Low | Medium |
| Complexity | Low | Medium |

---

## 🎯 Decision Guide

### Choose **Simple Mode** if:
- ✅ You have a single tenant (one app/customer)
- ✅ You want minimal setup
- ✅ You don't need per-customer credentials
- ✅ You're building an MVP or prototype
- ✅ You want to avoid database complexity
- ✅ You have < 10,000 requests/day
- ✅ All notifications use the same SMTP/WhatsApp account

### Choose **Multi-Tenant Mode** if:
- ✅ You have multiple tenants (SaaS, enterprise)
- ✅ Each tenant needs their own credentials
- ✅ You need tenant management API
- ✅ You need advanced security (encryption, hashing)
- ✅ You want per-tenant rate limiting
- ✅ You're building a notification-as-a-service
- ✅ You need audit trails and analytics
- ✅ You have > 10,000 requests/day

---

## 🔒 Security Considerations

### Simple Mode
- API keys in environment variables (not hashed)
- No credential encryption
- Global rate limiting only
- **Best for**: Internal tools, trusted environments

### Multi-Tenant Mode
- API keys hashed with bcrypt (cost=10)
- Credentials encrypted with AES-256-GCM
- Per-tenant and per-IP rate limiting
- Brute-force protection (10 req/sec per IP)
- **Best for**: Public-facing SaaS, untrusted environments

---

## 📚 Additional Resources

- **Architecture**: See `VENDOR_AGNOSTIC_AUDIT.md` for vendor-agnostic design
- **Day 16 Implementation**: See `DAY16_IMPLEMENTATION.md` for multi-tenant details
- **API Reference**: See API docs for complete endpoint reference
- **Security**: See `DAY15_CODE_REVIEW.md` for security implementation

---

## 🤝 Contributing

When adding features, ensure compatibility with **both modes**:

1. **Simple mode** should work without database
2. **Multi-tenant mode** should gracefully fall back to global config
3. Both modes should be vendor-agnostic
4. Document which mode supports which features

---

## 📞 Support

- **Issues**: https://github.com/huzaifabhutta/notify-core/issues
- **Discussions**: https://github.com/huzaifabhutta/notify-core/discussions
- **Documentation**: https://docs.notify-core.dev (coming soon)

---

**Built with ❤️ for the open-source community**
