package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/huzaifabhutta/notify-core/internal/models"
	"github.com/huzaifabhutta/notify-core/internal/services"
	"github.com/rs/zerolog"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// TenantContextKey is the key for storing tenant in context
	TenantContextKey contextKey = "tenant"

	// RequestIDContextKey is the key for storing request ID in context
	RequestIDContextKey contextKey = "request_id"
)

// TenantAuth is middleware that validates API key and injects tenant into context
type TenantAuth struct {
	tenantService *services.TenantService
	logger        zerolog.Logger
}

// NewTenantAuth creates a new tenant authentication middleware
func NewTenantAuth(tenantService *services.TenantService, logger zerolog.Logger) *TenantAuth {
	return &TenantAuth{
		tenantService: tenantService,
		logger:        logger.With().Str("middleware", "tenant_auth").Logger(),
	}
}

// Authenticate validates API key from header and injects tenant into context
func (m *TenantAuth) Authenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract API key from header
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			// Try Authorization header as fallback
			apiKey = c.Get("Authorization")
			if apiKey != "" {
				// Remove "Bearer " prefix if present
				if len(apiKey) > 7 && apiKey[:7] == "Bearer " {
					apiKey = apiKey[7:]
				}
			}
		}

		if apiKey == "" {
			m.logger.Warn().
				Str("path", c.Path()).
				Str("ip", c.IP()).
				Msg("Missing API key")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"error":   "UNAUTHORIZED",
				"message": "API key is required. Provide X-API-Key header or Authorization: Bearer <key>",
			})
		}

		// Get client IP for rate limiting
		clientIP := c.IP()

		// Validate API key and get tenant
		ctx := c.UserContext()
		tenant, err := m.tenantService.ValidateAPIKey(ctx, apiKey, clientIP)
		if err != nil {
			m.logger.Warn().
				Err(err).
				Str("path", c.Path()).
				Str("ip", clientIP).
				Msg("Invalid API key")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"error":   "INVALID_API_KEY",
				"message": "Invalid or inactive API key",
			})
		}

		// Log successful authentication
		m.logger.Debug().
			Int("tenant_id", tenant.ID).
			Str("tenant_name", tenant.Name).
			Str("path", c.Path()).
			Msg("Tenant authenticated")

		// Inject tenant into context
		ctxWithTenant := context.WithValue(ctx, TenantContextKey, tenant)
		c.SetUserContext(ctxWithTenant)

		// Add tenant info to response headers (for debugging)
		if m.logger.GetLevel() <= zerolog.DebugLevel {
			c.Set("X-Tenant-ID", string(rune(tenant.ID)))
			c.Set("X-Tenant-Name", tenant.Name)
		}

		return c.Next()
	}
}

// Optional is middleware that extracts tenant if API key is provided, but doesn't require it
// Useful for endpoints that work with or without authentication
func (m *TenantAuth) Optional() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract API key from header
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			apiKey = c.Get("Authorization")
			if apiKey != "" && len(apiKey) > 7 && apiKey[:7] == "Bearer " {
				apiKey = apiKey[7:]
			}
		}

		// If no API key provided, continue without tenant context
		if apiKey == "" {
			return c.Next()
		}

		// Try to validate API key
		ctx := c.UserContext()
		clientIP := c.IP()
		tenant, err := m.tenantService.ValidateAPIKey(ctx, apiKey, clientIP)
		if err != nil {
			// Log but don't fail - this is optional auth
			m.logger.Debug().
				Err(err).
				Str("path", c.Path()).
				Msg("Optional auth: invalid API key provided")
			return c.Next()
		}

		// Inject tenant into context
		ctxWithTenant := context.WithValue(ctx, TenantContextKey, tenant)
		c.SetUserContext(ctxWithTenant)

		m.logger.Debug().
			Int("tenant_id", tenant.ID).
			Str("tenant_name", tenant.Name).
			Msg("Optional auth: tenant authenticated")

		return c.Next()
	}
}

// GetTenantFromContext extracts tenant from context
func GetTenantFromContext(ctx context.Context) (*models.Tenant, bool) {
	tenant, ok := ctx.Value(TenantContextKey).(*models.Tenant)
	return tenant, ok
}

// GetTenantFromFiberContext extracts tenant from Fiber context
func GetTenantFromFiberContext(c *fiber.Ctx) (*models.Tenant, bool) {
	return GetTenantFromContext(c.UserContext())
}

// MustGetTenant extracts tenant from context or panics
// Use only in handlers protected by Authenticate() middleware
func MustGetTenant(ctx context.Context) *models.Tenant {
	tenant, ok := GetTenantFromContext(ctx)
	if !ok {
		panic("tenant not found in context - ensure Authenticate() middleware is applied")
	}
	return tenant
}
