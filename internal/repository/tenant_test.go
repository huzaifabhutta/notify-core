package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/huzaifabhutta/notify-core/internal/database"
	"github.com/huzaifabhutta/notify-core/internal/models"
)

func setupMockDB(t *testing.T) (*database.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}

	return &database.DB{DB: db}, mock
}

func TestTenantRepository_Create(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewTenantRepository(db)

	tests := []struct {
		name    string
		req     *models.CreateTenantRequest
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful creation",
			req: &models.CreateTenantRequest{
				Name:     "test-tenant",
				SMTPHost: "smtp.test.com",
				SMTPPort: 587,
				SMTPUser: "user@test.com",
			},
			mockFn: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO tenants`)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
						AddRow(1, time.Now(), time.Now()))
			},
			wantErr: false,
		},
		{
			name: "database error",
			req: &models.CreateTenantRequest{
				Name: "test-tenant",
			},
			mockFn: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO tenants`)).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
		{
			name: "duplicate tenant name",
			req: &models.CreateTenantRequest{
				Name: "existing-tenant",
			},
			mockFn: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO tenants`)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			tenant, err := repo.Create(context.Background(), tt.req)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tenant == nil {
					t.Error("Create() returned nil tenant")
				}
				if tenant.APIKey == "" {
					t.Error("Create() did not generate API key")
				}
				if tenant.Name != tt.req.Name {
					t.Errorf("Create() name = %v, want %v", tenant.Name, tt.req.Name)
				}
			}
		})
	}
}

func TestTenantRepository_GetByID(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewTenantRepository(db)

	tests := []struct {
		name    string
		id      int
		mockFn  func()
		wantErr bool
	}{
		{
			name: "found tenant",
			id:   1,
			mockFn: func() {
				rows := sqlmock.NewRows([]string{
					"id", "name", "api_key",
					"smtp_host", "smtp_port", "smtp_user", "smtp_password", "smtp_from",
					"wa_token", "wa_phone_id",
					"sms_provider", "sms_api_key", "sms_sender_id",
					"active", "created_at", "updated_at",
				}).AddRow(
					1, "test-tenant", "test-api-key",
					"smtp.test.com", 587, "user@test.com", "password", "from@test.com",
					"wa-token", "wa-phone-id",
					"twilio", "sms-key", "sender-id",
					true, time.Now(), time.Now(),
				)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
					WithArgs(1).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "tenant not found",
			id:   999,
			mockFn: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "database error",
			id:   1,
			mockFn: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
					WithArgs(1).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			tenant, err := repo.GetByID(context.Background(), tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tenant == nil {
				t.Error("GetByID() returned nil tenant")
			}
		})
	}
}

func TestTenantRepository_GetByAPIKey(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewTenantRepository(db)

	tests := []struct {
		name    string
		apiKey  string
		mockFn  func()
		wantErr bool
	}{
		{
			name:   "valid API key",
			apiKey: "valid-api-key",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{
					"id", "name", "api_key",
					"smtp_host", "smtp_port", "smtp_user", "smtp_password", "smtp_from",
					"wa_token", "wa_phone_id",
					"sms_provider", "sms_api_key", "sms_sender_id",
					"active", "created_at", "updated_at",
				}).AddRow(
					1, "test-tenant", "valid-api-key",
					"", 0, "", "", "",
					"", "",
					"", "", "",
					true, time.Now(), time.Now(),
				)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
					WithArgs("valid-api-key").
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:   "invalid API key",
			apiKey: "invalid-key",
			mockFn: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
					WithArgs("invalid-key").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name:   "inactive tenant",
			apiKey: "inactive-key",
			mockFn: func() {
				// Query filters by active = true, so no rows returned
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
					WithArgs("inactive-key").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			tenant, err := repo.GetByAPIKey(context.Background(), tt.apiKey)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetByAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tenant == nil {
					t.Error("GetByAPIKey() returned nil tenant")
				}
				if tenant.APIKey != tt.apiKey {
					t.Errorf("GetByAPIKey() api_key = %v, want %v", tenant.APIKey, tt.apiKey)
				}
			}
		})
	}
}

func TestTenantRepository_List(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewTenantRepository(db)

	tests := []struct {
		name       string
		activeOnly bool
		mockFn     func()
		wantCount  int
		wantErr    bool
	}{
		{
			name:       "list all tenants",
			activeOnly: false,
			mockFn: func() {
				rows := sqlmock.NewRows([]string{
					"id", "name", "api_key",
					"smtp_host", "smtp_port", "smtp_user", "smtp_password", "smtp_from",
					"wa_token", "wa_phone_id",
					"sms_provider", "sms_api_key", "sms_sender_id",
					"active", "created_at", "updated_at",
				}).
					AddRow(1, "tenant1", "key1", "", 0, "", "", "", "", "", "", "", "", true, time.Now(), time.Now()).
					AddRow(2, "tenant2", "key2", "", 0, "", "", "", "", "", "", "", "", false, time.Now(), time.Now())
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
					WillReturnRows(rows)
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:       "list active only",
			activeOnly: true,
			mockFn: func() {
				rows := sqlmock.NewRows([]string{
					"id", "name", "api_key",
					"smtp_host", "smtp_port", "smtp_user", "smtp_password", "smtp_from",
					"wa_token", "wa_phone_id",
					"sms_provider", "sms_api_key", "sms_sender_id",
					"active", "created_at", "updated_at",
				}).
					AddRow(1, "tenant1", "key1", "", 0, "", "", "", "", "", "", "", "", true, time.Now(), time.Now())
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
					WillReturnRows(rows)
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:       "empty list",
			activeOnly: false,
			mockFn: func() {
				rows := sqlmock.NewRows([]string{
					"id", "name", "api_key",
					"smtp_host", "smtp_port", "smtp_user", "smtp_password", "smtp_from",
					"wa_token", "wa_phone_id",
					"sms_provider", "sms_api_key", "sms_sender_id",
					"active", "created_at", "updated_at",
				})
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
					WillReturnRows(rows)
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:       "database error",
			activeOnly: false,
			mockFn: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
					WillReturnError(sql.ErrConnDone)
			},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			tenants, err := repo.List(context.Background(), tt.activeOnly)

			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(tenants) != tt.wantCount {
					t.Errorf("List() count = %v, want %v", len(tenants), tt.wantCount)
				}
			}
		})
	}
}

func TestTenantRepository_Update(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewTenantRepository(db)

	name := "updated-name"
	active := false

	tests := []struct {
		name    string
		id      int
		req     *models.UpdateTenantRequest
		mockFn  func()
		wantErr bool
	}{
		{
			name: "update name",
			id:   1,
			req: &models.UpdateTenantRequest{
				Name: &name,
			},
			mockFn: func() {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE tenants`)).
					WillReturnResult(sqlmock.NewResult(1, 1))

				// GetByID call after update
				rows := sqlmock.NewRows([]string{
					"id", "name", "api_key",
					"smtp_host", "smtp_port", "smtp_user", "smtp_password", "smtp_from",
					"wa_token", "wa_phone_id",
					"sms_provider", "sms_api_key", "sms_sender_id",
					"active", "created_at", "updated_at",
				}).AddRow(1, "updated-name", "key", "", 0, "", "", "", "", "", "", "", "", true, time.Now(), time.Now())
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "update active status",
			id:   1,
			req: &models.UpdateTenantRequest{
				Active: &active,
			},
			mockFn: func() {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE tenants`)).
					WillReturnResult(sqlmock.NewResult(1, 1))

				rows := sqlmock.NewRows([]string{
					"id", "name", "api_key",
					"smtp_host", "smtp_port", "smtp_user", "smtp_password", "smtp_from",
					"wa_token", "wa_phone_id",
					"sms_provider", "sms_api_key", "sms_sender_id",
					"active", "created_at", "updated_at",
				}).AddRow(1, "tenant", "key", "", 0, "", "", "", "", "", "", "", "", false, time.Now(), time.Now())
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT`)).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "update error",
			id:   1,
			req: &models.UpdateTenantRequest{
				Name: &name,
			},
			mockFn: func() {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE tenants`)).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			tenant, err := repo.Update(context.Background(), tt.id, tt.req)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tenant == nil {
				t.Error("Update() returned nil tenant")
			}
		})
	}
}

func TestTenantRepository_Delete(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewTenantRepository(db)

	tests := []struct {
		name    string
		id      int
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful soft delete",
			id:   1,
			mockFn: func() {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE tenants SET active = false`)).
					WithArgs(sqlmock.AnyArg(), 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "delete non-existent tenant",
			id:   999,
			mockFn: func() {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE tenants SET active = false`)).
					WithArgs(sqlmock.AnyArg(), 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false, // Soft delete doesn't error if not found
		},
		{
			name: "database error",
			id:   1,
			mockFn: func() {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE tenants SET active = false`)).
					WithArgs(sqlmock.AnyArg(), 1).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			err := repo.Delete(context.Background(), tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTenantRepository_HardDelete(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewTenantRepository(db)

	tests := []struct {
		name    string
		id      int
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful hard delete",
			id:   1,
			mockFn: func() {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM tenants WHERE id = $1`)).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "delete with cascade",
			id:   1,
			mockFn: func() {
				// Should cascade delete templates and notifications
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM tenants WHERE id = $1`)).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "database error",
			id:   1,
			mockFn: func() {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM tenants WHERE id = $1`)).
					WithArgs(1).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			err := repo.HardDelete(context.Background(), tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("HardDelete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
