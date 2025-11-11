package services

import (
	"context"
	"errors"
	"testing"

	"github.com/huzaifabhutta/notify-core/internal/models"
	"github.com/rs/zerolog"
)

// mockTenantRepository mocks the tenant repository for testing
type mockTenantRepository struct {
	createFn     func(ctx context.Context, req *models.CreateTenantRequest) (*models.Tenant, error)
	getByIDFn    func(ctx context.Context, id int) (*models.Tenant, error)
	getByAPIKeyFn func(ctx context.Context, apiKey string) (*models.Tenant, error)
	listFn       func(ctx context.Context, activeOnly bool) ([]*models.Tenant, error)
	updateFn     func(ctx context.Context, id int, req *models.UpdateTenantRequest) (*models.Tenant, error)
	deleteFn     func(ctx context.Context, id int) error
}

func (m *mockTenantRepository) Create(ctx context.Context, req *models.CreateTenantRequest) (*models.Tenant, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTenantRepository) GetByID(ctx context.Context, id int) (*models.Tenant, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTenantRepository) GetByAPIKey(ctx context.Context, apiKey string) (*models.Tenant, error) {
	if m.getByAPIKeyFn != nil {
		return m.getByAPIKeyFn(ctx, apiKey)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTenantRepository) List(ctx context.Context, activeOnly bool) ([]*models.Tenant, error) {
	if m.listFn != nil {
		return m.listFn(ctx, activeOnly)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTenantRepository) Update(ctx context.Context, id int, req *models.UpdateTenantRequest) (*models.Tenant, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTenantRepository) Delete(ctx context.Context, id int) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *mockTenantRepository) HardDelete(ctx context.Context, id int) error {
	return errors.New("not implemented")
}

func TestTenantService_CreateTenant(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name    string
		req     *models.CreateTenantRequest
		mockFn  func() *mockTenantRepository
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful creation",
			req: &models.CreateTenantRequest{
				Name:     "test-tenant",
				SMTPHost: "smtp.test.com",
				SMTPPort: 587,
			},
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					createFn: func(ctx context.Context, req *models.CreateTenantRequest) (*models.Tenant, error) {
						return &models.Tenant{
							ID:     1,
							Name:   req.Name,
							APIKey: "generated-key",
							Active: true,
						}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "empty tenant name",
			req: &models.CreateTenantRequest{
				Name: "",
			},
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{}
			},
			wantErr: true,
			errMsg:  "tenant name is required",
		},
		{
			name: "repository error",
			req: &models.CreateTenantRequest{
				Name: "test-tenant",
			},
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					createFn: func(ctx context.Context, req *models.CreateTenantRequest) (*models.Tenant, error) {
						return nil, errors.New("database error")
					},
				}
			},
			wantErr: true,
		},
		{
			name: "duplicate tenant name",
			req: &models.CreateTenantRequest{
				Name: "existing-tenant",
			},
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					createFn: func(ctx context.Context, req *models.CreateTenantRequest) (*models.Tenant, error) {
						return nil, errors.New("duplicate key value violates unique constraint")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mockFn()
			service := NewTenantService(repo, logger)

			tenant, err := service.CreateTenant(context.Background(), tt.req)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTenant() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("CreateTenant() error message = %v, want %v", err.Error(), tt.errMsg)
			}

			if !tt.wantErr {
				if tenant == nil {
					t.Error("CreateTenant() returned nil tenant")
				}
				if tenant.APIKey == "" {
					t.Error("CreateTenant() tenant has no API key")
				}
			}
		})
	}
}

func TestTenantService_GetTenant(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name    string
		id      int
		mockFn  func() *mockTenantRepository
		wantErr bool
	}{
		{
			name: "found tenant",
			id:   1,
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					getByIDFn: func(ctx context.Context, id int) (*models.Tenant, error) {
						return &models.Tenant{
							ID:     id,
							Name:   "test-tenant",
							APIKey: "test-key",
							Active: true,
						}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "tenant not found",
			id:   999,
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					getByIDFn: func(ctx context.Context, id int) (*models.Tenant, error) {
						return nil, errors.New("tenant not found: 999")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mockFn()
			service := NewTenantService(repo, logger)

			tenant, err := service.GetTenant(context.Background(), tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetTenant() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tenant == nil {
				t.Error("GetTenant() returned nil tenant")
			}
		})
	}
}

func TestTenantService_ValidateAPIKey(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name    string
		apiKey  string
		mockFn  func() *mockTenantRepository
		wantErr bool
		errMsg  string
	}{
		{
			name:   "valid API key",
			apiKey: "valid-key",
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					getByAPIKeyFn: func(ctx context.Context, apiKey string) (*models.Tenant, error) {
						return &models.Tenant{
							ID:     1,
							Name:   "test-tenant",
							APIKey: apiKey,
							Active: true,
						}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:   "empty API key",
			apiKey: "",
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{}
			},
			wantErr: true,
			errMsg:  "API key is required",
		},
		{
			name:   "invalid API key",
			apiKey: "invalid-key",
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					getByAPIKeyFn: func(ctx context.Context, apiKey string) (*models.Tenant, error) {
						return nil, errors.New("tenant not found or inactive")
					},
				}
			},
			wantErr: true,
			errMsg:  "invalid or inactive API key",
		},
		{
			name:   "inactive tenant",
			apiKey: "inactive-key",
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					getByAPIKeyFn: func(ctx context.Context, apiKey string) (*models.Tenant, error) {
						return nil, errors.New("tenant not found or inactive")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mockFn()
			service := NewTenantService(repo, logger)

			tenant, err := service.ValidateAPIKey(context.Background(), tt.apiKey)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.errMsg != "" && err != nil && err.Error() != tt.errMsg {
				t.Errorf("ValidateAPIKey() error message = %v, want %v", err.Error(), tt.errMsg)
			}

			if !tt.wantErr {
				if tenant == nil {
					t.Error("ValidateAPIKey() returned nil tenant")
				}
				if tenant.APIKey != tt.apiKey {
					t.Errorf("ValidateAPIKey() api_key = %v, want %v", tenant.APIKey, tt.apiKey)
				}
			}
		})
	}
}

func TestTenantService_ListTenants(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name       string
		activeOnly bool
		mockFn     func() *mockTenantRepository
		wantCount  int
		wantErr    bool
	}{
		{
			name:       "list all tenants",
			activeOnly: false,
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					listFn: func(ctx context.Context, activeOnly bool) ([]*models.Tenant, error) {
						return []*models.Tenant{
							{ID: 1, Name: "tenant1", Active: true},
							{ID: 2, Name: "tenant2", Active: false},
						}, nil
					},
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:       "list active only",
			activeOnly: true,
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					listFn: func(ctx context.Context, activeOnly bool) ([]*models.Tenant, error) {
						return []*models.Tenant{
							{ID: 1, Name: "tenant1", Active: true},
						}, nil
					},
				}
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:       "empty list",
			activeOnly: false,
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					listFn: func(ctx context.Context, activeOnly bool) ([]*models.Tenant, error) {
						return []*models.Tenant{}, nil
					},
				}
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:       "repository error",
			activeOnly: false,
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					listFn: func(ctx context.Context, activeOnly bool) ([]*models.Tenant, error) {
						return nil, errors.New("database error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mockFn()
			service := NewTenantService(repo, logger)

			tenants, err := service.ListTenants(context.Background(), tt.activeOnly)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListTenants() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(tenants) != tt.wantCount {
					t.Errorf("ListTenants() count = %v, want %v", len(tenants), tt.wantCount)
				}
			}
		})
	}
}

func TestTenantService_UpdateTenant(t *testing.T) {
	logger := zerolog.Nop()

	name := "updated-name"

	tests := []struct {
		name    string
		id      int
		req     *models.UpdateTenantRequest
		mockFn  func() *mockTenantRepository
		wantErr bool
	}{
		{
			name: "successful update",
			id:   1,
			req: &models.UpdateTenantRequest{
				Name: &name,
			},
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					updateFn: func(ctx context.Context, id int, req *models.UpdateTenantRequest) (*models.Tenant, error) {
						return &models.Tenant{
							ID:     id,
							Name:   *req.Name,
							Active: true,
						}, nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "tenant not found",
			id:   999,
			req: &models.UpdateTenantRequest{
				Name: &name,
			},
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					updateFn: func(ctx context.Context, id int, req *models.UpdateTenantRequest) (*models.Tenant, error) {
						return nil, errors.New("tenant not found: 999")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mockFn()
			service := NewTenantService(repo, logger)

			tenant, err := service.UpdateTenant(context.Background(), tt.id, tt.req)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateTenant() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tenant == nil {
				t.Error("UpdateTenant() returned nil tenant")
			}
		})
	}
}

func TestTenantService_DeleteTenant(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name    string
		id      int
		mockFn  func() *mockTenantRepository
		wantErr bool
	}{
		{
			name: "successful delete",
			id:   1,
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					deleteFn: func(ctx context.Context, id int) error {
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "repository error",
			id:   1,
			mockFn: func() *mockTenantRepository {
				return &mockTenantRepository{
					deleteFn: func(ctx context.Context, id int) error {
						return errors.New("database error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.mockFn()
			service := NewTenantService(repo, logger)

			err := service.DeleteTenant(context.Background(), tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteTenant() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
