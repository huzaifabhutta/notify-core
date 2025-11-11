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

	"github.com/google/uuid"
	"github.com/huzaifabhutta/notify-core/internal/database"
	"github.com/huzaifabhutta/notify-core/internal/models"
)

// TenantRepository handles tenant data access
type TenantRepository struct {
	db *database.DB
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *database.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// Create creates a new tenant
func (r *TenantRepository) Create(ctx context.Context, req *models.CreateTenantRequest) (*models.Tenant, error) {
	// Generate API key
	apiKey := uuid.New().String()

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
	tenant := &models.Tenant{
		Name:   req.Name,
		APIKey: apiKey,

		SMTPHost:     req.SMTPHost,
		SMTPPort:     req.SMTPPort,
		SMTPUser:     req.SMTPUser,
		SMTPPassword: req.SMTPPassword,
		SMTPFrom:     req.SMTPFrom,

		WAToken:   req.WAToken,
		WAPhoneID: req.WAPhoneID,

		SMSProvider: req.SMSProvider,
		SMSAPIKey:   req.SMSAPIKey,
		SMSSenderID: req.SMSSenderID,

		Active: true,
	}

	err := r.db.QueryRowContext(ctx, query,
		tenant.Name, tenant.APIKey,
		tenant.SMTPHost, tenant.SMTPPort, tenant.SMTPUser, tenant.SMTPPassword, tenant.SMTPFrom,
		tenant.WAToken, tenant.WAPhoneID,
		tenant.SMSProvider, tenant.SMSAPIKey, tenant.SMSSenderID,
		tenant.Active, now, now,
	).Scan(&tenant.ID, &tenant.CreatedAt, &tenant.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	return tenant, nil
}

// GetByID retrieves a tenant by ID
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

	return tenant, nil
}

// GetByAPIKey retrieves a tenant by API key
func (r *TenantRepository) GetByAPIKey(ctx context.Context, apiKey string) (*models.Tenant, error) {
	query := `
		SELECT
			id, name, api_key,
			smtp_host, smtp_port, smtp_user, smtp_password, smtp_from,
			wa_token, wa_phone_id,
			sms_provider, sms_api_key, sms_sender_id,
			active, created_at, updated_at
		FROM tenants
		WHERE api_key = $1 AND active = true
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRowContext(ctx, query, apiKey).Scan(
		&tenant.ID, &tenant.Name, &tenant.APIKey,
		&tenant.SMTPHost, &tenant.SMTPPort, &tenant.SMTPUser, &tenant.SMTPPassword, &tenant.SMTPFrom,
		&tenant.WAToken, &tenant.WAPhoneID,
		&tenant.SMSProvider, &tenant.SMSAPIKey, &tenant.SMSSenderID,
		&tenant.Active, &tenant.CreatedAt, &tenant.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant not found or inactive")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant by API key: %w", err)
	}

	return tenant, nil
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

// Update updates a tenant
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
		query += fmt.Sprintf(", smtp_password = $%d", argCount)
		args = append(args, *req.SMTPPassword)
		argCount++
	}

	if req.SMTPFrom != nil {
		query += fmt.Sprintf(", smtp_from = $%d", argCount)
		args = append(args, *req.SMTPFrom)
		argCount++
	}

	if req.WAToken != nil {
		query += fmt.Sprintf(", wa_token = $%d", argCount)
		args = append(args, *req.WAToken)
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
		query += fmt.Sprintf(", sms_api_key = $%d", argCount)
		args = append(args, *req.SMSAPIKey)
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

	// Fetch updated tenant
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
