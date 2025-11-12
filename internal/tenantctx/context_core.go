package tenantctx

import (
	"context"

	"github.com/huzaifabhutta/notify-core/internal/models"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// TenantContextKey is the key for storing tenant in context
	TenantContextKey contextKey = "tenant"
)

// SetTenant injects a tenant into the context
func SetTenant(ctx context.Context, tenant *models.Tenant) context.Context {
	return context.WithValue(ctx, TenantContextKey, tenant)
}

// GetTenant extracts tenant from context
func GetTenant(ctx context.Context) (*models.Tenant, bool) {
	tenant, ok := ctx.Value(TenantContextKey).(*models.Tenant)
	return tenant, ok
}

// MustGetTenant extracts tenant from context or panics
// Use only in handlers protected by Authenticate() middleware
func MustGetTenant(ctx context.Context) *models.Tenant {
	tenant, ok := GetTenant(ctx)
	if !ok {
		panic("tenant not found in context - ensure Authenticate() middleware is applied")
	}
	return tenant
}
