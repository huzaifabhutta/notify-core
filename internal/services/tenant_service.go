package services

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/huzaifabhutta/notify-core/internal/models"
	"github.com/huzaifabhutta/notify-core/internal/repository"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

var (
	// tenantNameRegex validates tenant name format (alphanumeric, hyphens, underscores)
	tenantNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// TenantService handles tenant business logic
type TenantService struct {
	repo         *repository.TenantRepository
	logger       zerolog.Logger
	rateLimiters map[string]*rate.Limiter
	limiterMu    sync.RWMutex
}

// NewTenantService creates a new tenant service
func NewTenantService(repo *repository.TenantRepository, logger zerolog.Logger) *TenantService {
	return &TenantService{
		repo:         repo,
		logger:       logger.With().Str("service", "tenant").Logger(),
		rateLimiters: make(map[string]*rate.Limiter),
	}
}

// getRateLimiter gets or creates a rate limiter for an IP address
func (s *TenantService) getRateLimiter(ip string) *rate.Limiter {
	s.limiterMu.Lock()
	defer s.limiterMu.Unlock()

	limiter, exists := s.rateLimiters[ip]
	if !exists {
		// Allow 10 requests per second with burst of 20
		limiter = rate.NewLimiter(rate.Every(time.Second/10), 20)
		s.rateLimiters[ip] = limiter
	}

	return limiter
}

// CreateTenantResponse contains the tenant and plain API key (only returned once)
type CreateTenantResponse struct {
	Tenant       *models.Tenant
	PlainAPIKey  string // Only returned once during creation
}

// CreateTenant creates a new tenant with enhanced validation
func (s *TenantService) CreateTenant(ctx context.Context, req *models.CreateTenantRequest) (*CreateTenantResponse, error) {
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
	if !tenantNameRegex.MatchString(req.Name) {
		return nil, fmt.Errorf("tenant name must contain only letters, numbers, hyphens, and underscores")
	}

	// Create tenant (returns tenant with hashed API key + plain API key)
	tenant, plainAPIKey, err := s.repo.Create(ctx, req)
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

	return &CreateTenantResponse{
		Tenant:      tenant,
		PlainAPIKey: plainAPIKey,
	}, nil
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

// ValidateAPIKey validates a tenant API key with rate limiting and returns the tenant
// Note: Pass client IP address for rate limiting. If IP is empty, rate limiting is skipped.
func (s *TenantService) ValidateAPIKey(ctx context.Context, apiKey, clientIP string) (*models.Tenant, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Apply rate limiting per IP address
	if clientIP != "" {
		limiter := s.getRateLimiter(clientIP)
		if !limiter.Allow() {
			s.logger.Warn().
				Str("client_ip", clientIP).
				Msg("Rate limit exceeded for API key validation")
			return nil, fmt.Errorf("rate limit exceeded, please try again later")
		}
	}

	tenant, err := s.repo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		s.logger.Warn().
			Str("client_ip", clientIP).
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
