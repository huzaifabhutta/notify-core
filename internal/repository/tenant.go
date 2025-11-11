// Package repository provides data access layer for the notify-core database.
//
// SECURITY: All queries MUST use parameterized statements ($1, $2, etc.) to prevent SQL injection.
// NEVER concatenate user input into SQL strings.
//
// ✅ SAFE:   query := "SELECT * FROM users WHERE id = $1"
// ❌ UNSAFE: query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userInput)
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"strings"

	"github.com/google/uuid"
	"github.com/huzaifabhutta/notify-core/internal/database"
	"github.com/huzaifabhutta/notify-core/internal/models"
	"github.com/huzaifabhutta/notify-core/internal/security"
)

// TenantRepository handles tenant data access
type TenantRepository struct {
	db            *database.DB
	encryptionKey string
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *database.DB, encryptionKey string) *TenantRepository {
	return &TenantRepository{
		db:            db,
		encryptionKey: encryptionKey,
	}
}

// Create creates a new tenant with hashed API key and encrypted credentials
func (r *TenantRepository) Create(ctx context.Context, req *models.CreateTenantRequest) (*models.Tenant, string, error) {
	const maxRetries = 3

	query := `
		INSERT INTO tenants (
			name, api_key,
			smtp_host, smtp_port, smtp_user, smtp_password, smtp_from,
			wa_token, wa_phone_id,
			sms_provider, sms_api_key, sms_sender_id,
			active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	var tenant *models.Tenant
	var plainAPIKey string
	var err error

	// Retry loop for potential UUID collisions
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Generate plain API key
		plainAPIKey = uuid.New().String()

		// Hash API key for storage
		hashedAPIKey, err := security.HashAPIKey(plainAPIKey)
		if err != nil {
			return nil, "", fmt.Errorf("failed to hash API key: %w", err)
		}

		// Encrypt sensitive credentials
		encryptedSMTPPassword, err := security.EncryptIfNotEmpty(req.SMTPPassword, r.encryptionKey)
		if err != nil {
			return nil, "", fmt.Errorf("failed to encrypt SMTP password: %w", err)
		}

		encryptedWAToken, err := security.EncryptIfNotEmpty(req.WAToken, r.encryptionKey)
		if err != nil {
			return nil, "", fmt.Errorf("failed to encrypt WhatsApp token: %w", err)
		}

		encryptedSMSAPIKey, err := security.EncryptIfNotEmpty(req.SMSAPIKey, r.encryptionKey)
		if err != nil {
			return nil, "", fmt.Errorf("failed to encrypt SMS API key: %w", err)
		}

		tenant = &models.Tenant{
			Name:   req.Name,
			APIKey: hashedAPIKey, // Store hashed version

			SMTPHost:     req.SMTPHost,
			SMTPPort:     req.SMTPPort,
			SMTPUser:     req.SMTPUser,
			SMTPPassword: encryptedSMTPPassword, // Store encrypted
			SMTPFrom:     req.SMTPFrom,

			WAToken:   encryptedWAToken, // Store encrypted
			WAPhoneID: req.WAPhoneID,

			SMSProvider: req.SMSProvider,
			SMSAPIKey:   encryptedSMSAPIKey, // Store encrypted
			SMSSenderID: req.SMSSenderID,

			Active: true,
		}

		err = r.db.QueryRowContext(ctx, query,
			tenant.Name, tenant.APIKey,
			tenant.SMTPHost, tenant.SMTPPort, tenant.SMTPUser, tenant.SMTPPassword, tenant.SMTPFrom,
			tenant.WAToken, tenant.WAPhoneID,
			tenant.SMSProvider, tenant.SMSAPIKey, tenant.SMSSenderID,
			tenant.Active, now, now,
		).Scan(&tenant.ID, &tenant.CreatedAt, &tenant.UpdatedAt)

		if err == nil {
			break // Success
		}

		// Check if it's a unique constraint error on api_key
		if strings.Contains(err.Error(), "unique constraint") && strings.Contains(err.Error(), "api_key") {
			// UUID collision - retry with new UUID
			if attempt < maxRetries-1 {
				continue
			}
		}

		// Different error or max retries reached
		return nil, "", fmt.Errorf("failed to create tenant after %d attempts: %w", attempt+1, err)
	}

	// Return plain API key to user (only time they see it)
	return tenant, plainAPIKey, nil
}

// GetByID retrieves a tenant by ID and decrypts sensitive credentials
func (r *TenantRepository) GetByID(ctx context.Context, id int) (*models.Tenant, error) {
	query := `
		SELECT
			id, name, api_key,
			smtp_host, smtp_port, smtp_user, smtp_password, smtp_from,
			wa_token, wa_phone_id,
			sms_provider, sms_api_key, sms_sender_id,
			active, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID, &tenant.Name, &tenant.APIKey,
		&tenant.SMTPHost, &tenant.SMTPPort, &tenant.SMTPUser, &tenant.SMTPPassword, &tenant.SMTPFrom,
		&tenant.WAToken, &tenant.WAPhoneID,
		&tenant.SMSProvider, &tenant.SMSAPIKey, &tenant.SMSSenderID,
		&tenant.Active, &tenant.CreatedAt, &tenant.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Decrypt sensitive credentials before returning
	if err := r.decryptTenantCredentials(tenant); err != nil {
		return nil, fmt.Errorf("failed to decrypt tenant credentials: %w", err)
	}

	return tenant, nil
}

// GetByAPIKey retrieves a tenant by comparing plain API key with hashed keys
// Note: This iterates through active tenants to compare hashes. For large tenant counts,
// consider using HMAC instead of bcrypt for O(1) lookups, or implement caching.
func (r *TenantRepository) GetByAPIKey(ctx context.Context, plainAPIKey string) (*models.Tenant, error) {
	query := `
		SELECT
			id, name, api_key,
			smtp_host, smtp_port, smtp_user, smtp_password, smtp_from,
			wa_token, wa_phone_id,
			sms_provider, sms_api_key, sms_sender_id,
			active, created_at, updated_at
		FROM tenants
		WHERE active = true
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenants: %w", err)
	}
	defer rows.Close()

	// Iterate through active tenants and compare API key hashes
	for rows.Next() {
		tenant := &models.Tenant{}
		err := rows.Scan(
			&tenant.ID, &tenant.Name, &tenant.APIKey,
			&tenant.SMTPHost, &tenant.SMTPPort, &tenant.SMTPUser, &tenant.SMTPPassword, &tenant.SMTPFrom,
			&tenant.WAToken, &tenant.WAPhoneID,
			&tenant.SMSProvider, &tenant.SMSAPIKey, &tenant.SMSSenderID,
			&tenant.Active, &tenant.CreatedAt, &tenant.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}

		// Compare plain API key with hashed key
		if err := security.CompareAPIKey(tenant.APIKey, plainAPIKey); err == nil {
			// Match found - decrypt sensitive credentials before returning
			if err := r.decryptTenantCredentials(tenant); err != nil {
				return nil, fmt.Errorf("failed to decrypt tenant credentials: %w", err)
			}
			return tenant, nil
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenants: %w", err)
	}

	// No match found
	return nil, fmt.Errorf("tenant not found or inactive")
}

// List retrieves all tenants with optional filters
func (r *TenantRepository) List(ctx context.Context, activeOnly bool) ([]*models.Tenant, error) {
	query := `
		SELECT
			id, name, api_key,
			smtp_host, smtp_port, smtp_user, smtp_password, smtp_from,
			wa_token, wa_phone_id,
			sms_provider, sms_api_key, sms_sender_id,
			active, created_at, updated_at
		FROM tenants
	`

	if activeOnly {
		query += " WHERE active = true"
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		tenant := &models.Tenant{}
		err := rows.Scan(
			&tenant.ID, &tenant.Name, &tenant.APIKey,
			&tenant.SMTPHost, &tenant.SMTPPort, &tenant.SMTPUser, &tenant.SMTPPassword, &tenant.SMTPFrom,
			&tenant.WAToken, &tenant.WAPhoneID,
			&tenant.SMSProvider, &tenant.SMSAPIKey, &tenant.SMSSenderID,
			&tenant.Active, &tenant.CreatedAt, &tenant.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenants = append(tenants, tenant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenants: %w", err)
	}

	return tenants, nil
}

// Update updates a tenant with encryption for sensitive fields
func (r *TenantRepository) Update(ctx context.Context, id int, req *models.UpdateTenantRequest) (*models.Tenant, error) {
	// Build dynamic query based on provided fields
	query := "UPDATE tenants SET updated_at = $1"
	args := []interface{}{time.Now()}
	argCount := 2

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argCount)
		args = append(args, *req.Name)
		argCount++
	}

	if req.SMTPHost != nil {
		query += fmt.Sprintf(", smtp_host = $%d", argCount)
		args = append(args, *req.SMTPHost)
		argCount++
	}

	if req.SMTPPort != nil {
		query += fmt.Sprintf(", smtp_port = $%d", argCount)
		args = append(args, *req.SMTPPort)
		argCount++
	}

	if req.SMTPUser != nil {
		query += fmt.Sprintf(", smtp_user = $%d", argCount)
		args = append(args, *req.SMTPUser)
		argCount++
	}

	if req.SMTPPassword != nil {
		// Encrypt SMTP password before storing
		encryptedPassword, err := security.EncryptString(*req.SMTPPassword, r.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt SMTP password: %w", err)
		}
		query += fmt.Sprintf(", smtp_password = $%d", argCount)
		args = append(args, encryptedPassword)
		argCount++
	}

	if req.SMTPFrom != nil {
		query += fmt.Sprintf(", smtp_from = $%d", argCount)
		args = append(args, *req.SMTPFrom)
		argCount++
	}

	if req.WAToken != nil {
		// Encrypt WhatsApp token before storing
		encryptedToken, err := security.EncryptString(*req.WAToken, r.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt WhatsApp token: %w", err)
		}
		query += fmt.Sprintf(", wa_token = $%d", argCount)
		args = append(args, encryptedToken)
		argCount++
	}

	if req.WAPhoneID != nil {
		query += fmt.Sprintf(", wa_phone_id = $%d", argCount)
		args = append(args, *req.WAPhoneID)
		argCount++
	}

	if req.SMSProvider != nil {
		query += fmt.Sprintf(", sms_provider = $%d", argCount)
		args = append(args, *req.SMSProvider)
		argCount++
	}

	if req.SMSAPIKey != nil {
		// Encrypt SMS API key before storing
		encryptedAPIKey, err := security.EncryptString(*req.SMSAPIKey, r.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt SMS API key: %w", err)
		}
		query += fmt.Sprintf(", sms_api_key = $%d", argCount)
		args = append(args, encryptedAPIKey)
		argCount++
	}

	if req.SMSSenderID != nil {
		query += fmt.Sprintf(", sms_sender_id = $%d", argCount)
		args = append(args, *req.SMSSenderID)
		argCount++
	}

	if req.Active != nil {
		query += fmt.Sprintf(", active = $%d", argCount)
		args = append(args, *req.Active)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	// Fetch updated tenant (will be decrypted by GetByID)
	return r.GetByID(ctx, id)
}

// Delete soft deletes a tenant (sets active = false)
func (r *TenantRepository) Delete(ctx context.Context, id int) error {
	query := "UPDATE tenants SET active = false, updated_at = $1 WHERE id = $2"
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}
	return nil
}

// HardDelete permanently deletes a tenant and all associated data
func (r *TenantRepository) HardDelete(ctx context.Context, id int) error {
	query := "DELETE FROM tenants WHERE id = $1"
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to hard delete tenant: %w", err)
	}
	return nil
}

// decryptTenantCredentials decrypts sensitive credentials in a tenant object
func (r *TenantRepository) decryptTenantCredentials(tenant *models.Tenant) error {
	var err error

	// Decrypt SMTP password
	if tenant.SMTPPassword != "" {
		tenant.SMTPPassword, err = security.DecryptString(tenant.SMTPPassword, r.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt SMTP password: %w", err)
		}
	}

	// Decrypt WhatsApp token
	if tenant.WAToken != "" {
		tenant.WAToken, err = security.DecryptString(tenant.WAToken, r.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt WhatsApp token: %w", err)
		}
	}

	// Decrypt SMS API key
	if tenant.SMSAPIKey != "" {
		tenant.SMSAPIKey, err = security.DecryptString(tenant.SMSAPIKey, r.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt SMS API key: %w", err)
		}
	}

	return nil
}
