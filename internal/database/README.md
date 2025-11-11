# Database Layer

This package provides database connectivity and migration management for the notify-core multi-tenant notification service.

## Overview

The database layer uses PostgreSQL and provides:
- **Connection pooling** with configurable limits
- **Migration system** for schema versioning
- **Multi-tenant data isolation** with tenant-specific tables
- **Transaction support** for data consistency

## Architecture

```
internal/database/
├── database.go      # Connection management
├── migrations.go    # Migration system
└── README.md        # This file

internal/models/
└── tenant.go        # Tenant data model

internal/repository/
└── tenant.go        # Tenant data access layer

internal/services/
└── tenant_service.go # Tenant business logic
```

## Database Schema

### Tables

#### 1. `tenants`
Stores tenant information and channel-specific credentials.

```sql
CREATE TABLE tenants (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    api_key VARCHAR(255) NOT NULL UNIQUE,

    -- Email configuration (per-tenant SMTP)
    smtp_host VARCHAR(255),
    smtp_port INT,
    smtp_user VARCHAR(255),
    smtp_password VARCHAR(255),
    smtp_from VARCHAR(255),

    -- WhatsApp configuration (per-tenant)
    wa_token TEXT,
    wa_phone_id VARCHAR(255),

    -- SMS configuration (per-tenant)
    sms_provider VARCHAR(50),
    sms_api_key TEXT,
    sms_sender_id VARCHAR(255),

    -- Status
    active BOOLEAN NOT NULL DEFAULT true,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Indexes:**
- `idx_tenants_api_key` on `api_key` (for fast authentication)
- `idx_tenants_active` on `active` (for filtering active tenants)

#### 2. `templates`
Stores notification templates per tenant and channel.

```sql
CREATE TABLE templates (
    id SERIAL PRIMARY KEY,
    tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE,
    channel VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    subject VARCHAR(500),
    body TEXT NOT NULL,
    variables JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, channel, name)
);
```

**Indexes:**
- `idx_templates_tenant` on `tenant_id`
- `idx_templates_channel` on `channel`
- `idx_templates_active` on `active`

#### 3. `notifications`
Logs all sent notifications for audit and analytics.

```sql
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE,
    channel VARCHAR(50) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    template_name VARCHAR(255),
    subject VARCHAR(500),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    message_id VARCHAR(255),
    error TEXT,
    data JSONB,
    sent_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Indexes:**
- `idx_notifications_tenant` on `tenant_id`
- `idx_notifications_status` on `status`
- `idx_notifications_channel` on `channel`
- `idx_notifications_created_at` on `created_at`
- `idx_notifications_sent_at` on `sent_at`

#### 4. `schema_migrations`
Tracks applied database migrations (auto-created).

```sql
CREATE TABLE schema_migrations (
    version INT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## Setup

### 1. PostgreSQL Installation

**macOS (Homebrew):**
```bash
brew install postgresql@15
brew services start postgresql@15
```

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
```

**Docker:**
```bash
docker run -d \
  --name notify-postgres \
  -e POSTGRES_USER=notify \
  -e POSTGRES_PASSWORD=your-password \
  -e POSTGRES_DB=notify \
  -p 5432:5432 \
  postgres:15-alpine
```

### 2. Database Creation

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database and user
CREATE DATABASE notify;
CREATE USER notify WITH ENCRYPTED PASSWORD 'your-password';
GRANT ALL PRIVILEGES ON DATABASE notify TO notify;

# Exit
\q
```

### 3. Environment Configuration

Update `.env` file:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=notify
DB_PASSWORD=your-password
DB_NAME=notify
DB_SSL_MODE=disable
DB_MAX_CONNS=25
DB_MAX_IDLE=5
```

### 4. Run Migrations

```bash
# Run all pending migrations
go run cmd/migrate/main.go -action=up

# Check migration status
go run cmd/migrate/main.go -action=status

# Rollback last migration
go run cmd/migrate/main.go -action=down
```

## Usage

### Connect to Database

```go
import (
    "github.com/huzaifabhutta/notify-core/internal/config"
    "github.com/huzaifabhutta/notify-core/internal/database"
)

// Load config
cfg, _ := config.Load()

// Create database config
dbCfg := &database.Config{
    Host:     cfg.Database.Host,
    Port:     cfg.Database.Port,
    User:     cfg.Database.User,
    Password: cfg.Database.Password,
    DBName:   cfg.Database.DBName,
    SSLMode:  cfg.Database.SSLMode,
    MaxConns: cfg.Database.MaxConns,
    MaxIdle:  cfg.Database.MaxIdle,
}

// Connect
db, err := database.New(dbCfg)
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### Run Migrations

```go
import "github.com/huzaifabhutta/notify-core/internal/database"

migrator := database.NewMigrator(db)

// Run all pending migrations
if err := migrator.Up(ctx); err != nil {
    log.Fatal(err)
}
```

### Create a Tenant

```go
import (
    "github.com/huzaifabhutta/notify-core/internal/models"
    "github.com/huzaifabhutta/notify-core/internal/repository"
    "github.com/huzaifabhutta/notify-core/internal/services"
)

// Create repository
tenantRepo := repository.NewTenantRepository(db)

// Create service
tenantService := services.NewTenantService(tenantRepo, logger)

// Create tenant
req := &models.CreateTenantRequest{
    Name: "acme-corp",
    SMTPHost: "smtp.sendgrid.net",
    SMTPPort: 587,
    SMTPUser: "apikey",
    SMTPPassword: "SG.xxx",
    SMTPFrom: "noreply@acme.com",
}

tenant, err := tenantService.CreateTenant(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Tenant created with API key: %s\n", tenant.APIKey)
```

### Authenticate with API Key

```go
// Validate API key from request
tenant, err := tenantService.ValidateAPIKey(ctx, apiKey)
if err != nil {
    // Invalid API key
    return fiber.NewError(fiber.StatusUnauthorized, "Invalid API key")
}

// Use tenant-specific credentials
if tenant.HasEmailConfig() {
    // Use tenant's SMTP config
} else {
    // Fallback to global SMTP config
}
```

## Migration System

### Adding New Migrations

Edit `internal/database/migrations.go` and add a new migration:

```go
func GetMigrations() []Migration {
    return []Migration{
        // ... existing migrations ...
        {
            Version: 4,
            Name:    "add_templates_priority",
            Up: `
                ALTER TABLE templates
                ADD COLUMN priority INT DEFAULT 0;

                CREATE INDEX idx_templates_priority ON templates(priority);
            `,
            Down: `
                DROP INDEX IF EXISTS idx_templates_priority;
                ALTER TABLE templates DROP COLUMN priority;
            `,
        },
    }
}
```

**Rules:**
1. Never modify existing migrations
2. Always increment version number
3. Always provide both `Up` and `Down` SQL
4. Test rollback (`Down`) before committing
5. Use transactions for safety (auto-handled by migrator)

### Migration Workflow

```bash
# 1. Add new migration to migrations.go
# 2. Check status before applying
go run cmd/migrate/main.go -action=status

# 3. Apply migration
go run cmd/migrate/main.go -action=up

# 4. Verify in database
psql -U notify -d notify -c "SELECT * FROM schema_migrations;"

# 5. If issues, rollback
go run cmd/migrate/main.go -action=down
```

## Connection Pooling

The database layer automatically configures connection pooling:

- **MaxOpenConns**: 25 (configurable via `DB_MAX_CONNS`)
- **MaxIdleConns**: 5 (configurable via `DB_MAX_IDLE`)
- **ConnMaxLifetime**: 5 minutes

Adjust based on your application load:

```go
db.SetMaxOpenConns(50)     // Increase for high traffic
db.SetMaxIdleConns(10)     // Increase to reduce connection overhead
db.SetConnMaxLifetime(10 * time.Minute)
```

## Multi-Tenant Architecture

### Credential Hierarchy

1. **Tenant-specific credentials** (stored in `tenants` table)
   - Each tenant can have their own SMTP/WhatsApp/SMS config
   - Used when available

2. **Global fallback credentials** (from environment variables)
   - Used when tenant doesn't have channel-specific config
   - Configured in `.env` file

### Example Flow

```go
func (s *NotifyService) getEmailConfig(tenant *models.Tenant) SMTPConfig {
    if tenant.HasEmailConfig() {
        // Use tenant-specific SMTP
        return SMTPConfig{
            Host:     tenant.SMTPHost,
            Port:     tenant.SMTPPort,
            User:     tenant.SMTPUser,
            Password: tenant.SMTPPassword,
            From:     tenant.SMTPFrom,
        }
    }

    // Fallback to global SMTP
    return s.globalSMTPConfig
}
```

## Best Practices

### Security

1. **Never log passwords** - Use `[REDACTED]` in logs
2. **Use SSL in production** - Set `DB_SSL_MODE=require`
3. **Rotate API keys** - Implement key rotation for tenants
4. **Encrypt sensitive data** - Consider encrypting SMTP passwords at rest

### Performance

1. **Use indexes** - All foreign keys and frequently queried columns have indexes
2. **Connection pooling** - Reuse connections across requests
3. **Prepared statements** - Repository uses parameterized queries
4. **Batch operations** - Use transactions for multiple inserts

### Monitoring

```go
// Check database health
if err := db.Ping(ctx); err != nil {
    log.Error().Err(err).Msg("Database health check failed")
}

// Monitor connection pool stats
stats := db.Stats()
log.Info().
    Int("open_connections", stats.OpenConnections).
    Int("in_use", stats.InUse).
    Int("idle", stats.Idle).
    Msg("Database connection pool stats")
```

## Troubleshooting

### Connection Refused

```
Error: connection refused
```

**Solution:**
- Check PostgreSQL is running: `pg_isready -h localhost -p 5432`
- Verify port: `lsof -i :5432`
- Check credentials in `.env`

### Migration Failed

```
Error: migration 2 failed
```

**Solution:**
1. Check database logs: `tail -f /var/log/postgresql/postgresql.log`
2. Manually inspect: `psql -U notify -d notify`
3. Rollback: `go run cmd/migrate/main.go -action=down`
4. Fix migration SQL
5. Retry: `go run cmd/migrate/main.go -action=up`

### Too Many Connections

```
Error: pq: sorry, too many clients already
```

**Solution:**
- Reduce `DB_MAX_CONNS` in `.env`
- Increase PostgreSQL `max_connections`: Edit `postgresql.conf`
- Check for connection leaks: Always `defer db.Close()`

## Future Enhancements

- [ ] Add connection retry logic with exponential backoff
- [ ] Implement read replicas for scaling
- [ ] Add database metrics collection
- [ ] Support for multi-database sharding per tenant
- [ ] Automated backup and restore scripts
- [ ] Migration rollback with data preservation
- [ ] Database encryption at rest

## References

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Go database/sql](https://pkg.go.dev/database/sql)
- [pq Driver](https://github.com/lib/pq)
- [Migration Best Practices](https://www.enterprisedb.com/postgres-tutorials/postgresql-database-migration-best-practices)
