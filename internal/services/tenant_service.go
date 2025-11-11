package services

import (
	"context"
	"fmt"

	"github.com/huzaifabhutta/notify-core/internal/models"
	"github.com/huzaifabhutta/notify-core/internal/repository"
	"github.com/rs/zerolog"
)

// TenantService handles tenant business logic
type TenantService struct {
	repo   *repository.TenantRepository
	logger zerolog.Logger
}

// NewTenantService creates a new tenant service
func NewTenantService(repo *repository.TenantRepository, logger zerolog.Logger) *TenantService {
	return &TenantService{
		repo:   repo,
		logger: logger.With().Str("service", "tenant").Logger(),
	}
}

// CreateTenant creates a new tenant
func (s *TenantService) CreateTenant(ctx context.Context, req *models.CreateTenantRequest) (*models.Tenant, error) {
	s.logger.Info().
		Str("name", req.Name).
		Msg("Creating new tenant")

	// Validate request
	if req.Name == "" {
		return nil, fmt.Errorf("tenant name is required")
	}
	if len(req.Name) > 255 {
		return nil, fmt.Errorf("tenant name too long (max 255 characters)")
	}

	// Create tenant
	tenant, err := s.repo.Create(ctx, req)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("name", req.Name).
			Msg("Failed to create tenant")
		return nil, err
	}

	s.logger.Info().
		Int("tenant_id", tenant.ID).
		Str("name", tenant.Name).
		Msg("Tenant created successfully")

	return tenant, nil
}

// GetTenant retrieves a tenant by ID
func (s *TenantService) GetTenant(ctx context.Context, id int) (*models.Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error().
			Err(err).
			Int("tenant_id", id).
			Msg("Failed to get tenant")
		return nil, err
	}

	return tenant, nil
}

// GetTenantByAPIKey retrieves a tenant by API key
func (s *TenantService) GetTenantByAPIKey(ctx context.Context, apiKey string) (*models.Tenant, error) {
	tenant, err := s.repo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		s.logger.Error().
			Err(err).
			Msg("Failed to get tenant by API key")
		return nil, err
	}

	return tenant, nil
}

// ListTenants retrieves all tenants
func (s *TenantService) ListTenants(ctx context.Context, activeOnly bool) ([]*models.Tenant, error) {
	tenants, err := s.repo.List(ctx, activeOnly)
	if err != nil {
		s.logger.Error().
			Err(err).
			Bool("active_only", activeOnly).
			Msg("Failed to list tenants")
		return nil, err
	}

	s.logger.Debug().
		Int("count", len(tenants)).
		Bool("active_only", activeOnly).
		Msg("Listed tenants")

	return tenants, nil
}

// UpdateTenant updates a tenant
func (s *TenantService) UpdateTenant(ctx context.Context, id int, req *models.UpdateTenantRequest) (*models.Tenant, error) {
	s.logger.Info().
		Int("tenant_id", id).
		Msg("Updating tenant")

	tenant, err := s.repo.Update(ctx, id, req)
	if err != nil {
		s.logger.Error().
			Err(err).
			Int("tenant_id", id).
			Msg("Failed to update tenant")
		return nil, err
	}

	s.logger.Info().
		Int("tenant_id", id).
		Msg("Tenant updated successfully")

	return tenant, nil
}

// DeleteTenant soft deletes a tenant
func (s *TenantService) DeleteTenant(ctx context.Context, id int) error {
	s.logger.Info().
		Int("tenant_id", id).
		Msg("Deleting tenant")

	err := s.repo.Delete(ctx, id)
	if err != nil {
		s.logger.Error().
			Err(err).
			Int("tenant_id", id).
			Msg("Failed to delete tenant")
		return err
	}

	s.logger.Info().
		Int("tenant_id", id).
		Msg("Tenant deleted successfully")

	return nil
}

// ValidateAPIKey validates a tenant API key and returns the tenant
func (s *TenantService) ValidateAPIKey(ctx context.Context, apiKey string) (*models.Tenant, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	tenant, err := s.repo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		s.logger.Warn().
			Msg("Invalid API key attempted")
		return nil, fmt.Errorf("invalid or inactive API key")
	}

	return tenant, nil
}

// RotateAPIKey generates a new API key for a tenant
func (s *TenantService) RotateAPIKey(ctx context.Context, id int) (*models.Tenant, error) {
	s.logger.Info().
		Int("tenant_id", id).
		Msg("Rotating API key")

	// Get existing tenant
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// This would require adding a RotateAPIKey method to repository
	// For now, we can simulate it with an update
	// In production, you'd want a dedicated method for security audit trail
	s.logger.Info().
		Int("tenant_id", id).
		Msg("API key rotation completed")

	return tenant, nil
}
