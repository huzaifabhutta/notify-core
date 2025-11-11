package database

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMigrator_CreateMigrationsTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	dbWrapper := &DB{DB: db}
	migrator := NewMigrator(dbWrapper)

	tests := []struct {
		name    string
		mockFn  func()
		wantErr bool
	}{
		{
			name: "successful creation",
			mockFn: func() {
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name: "table already exists",
			mockFn: func() {
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			err := migrator.createMigrationsTable(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("createMigrationsTable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMigrator_GetCurrentVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	dbWrapper := &DB{DB: db}
	migrator := NewMigrator(dbWrapper)

	tests := []struct {
		name        string
		mockFn      func()
		wantVersion int
		wantErr     bool
	}{
		{
			name: "no migrations applied",
			mockFn: func() {
				mock.ExpectQuery("SELECT COALESCE\\(MAX\\(version\\), 0\\) FROM schema_migrations").
					WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(0))
			},
			wantVersion: 0,
			wantErr:     false,
		},
		{
			name: "version 3 applied",
			mockFn: func() {
				mock.ExpectQuery("SELECT COALESCE\\(MAX\\(version\\), 0\\) FROM schema_migrations").
					WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(3))
			},
			wantVersion: 3,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			version, err := migrator.getCurrentVersion(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("getCurrentVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if version != tt.wantVersion {
				t.Errorf("getCurrentVersion() = %v, want %v", version, tt.wantVersion)
			}
		})
	}
}

func TestGetMigrations(t *testing.T) {
	migrations := GetMigrations()

	if len(migrations) != 3 {
		t.Errorf("GetMigrations() returned %d migrations, want 3", len(migrations))
	}

	// Verify migration versions are sequential
	for i, m := range migrations {
		expectedVersion := i + 1
		if m.Version != expectedVersion {
			t.Errorf("Migration %d has version %d, want %d", i, m.Version, expectedVersion)
		}
	}

	// Verify all migrations have required fields
	for _, m := range migrations {
		if m.Name == "" {
			t.Errorf("Migration %d has empty name", m.Version)
		}
		if m.Up == "" {
			t.Errorf("Migration %d has empty Up SQL", m.Version)
		}
		if m.Down == "" {
			t.Errorf("Migration %d has empty Down SQL", m.Version)
		}
	}
}

func TestMigration_Structure(t *testing.T) {
	migrations := GetMigrations()

	// Test migration 1: tenants table
	m1 := migrations[0]
	if m1.Version != 1 {
		t.Errorf("Migration 1 has wrong version: %d", m1.Version)
	}
	if m1.Name != "create_tenants_table" {
		t.Errorf("Migration 1 has wrong name: %s", m1.Name)
	}

	// Verify tenants table has required fields
	requiredFields := []string{
		"CREATE TABLE tenants",
		"id SERIAL PRIMARY KEY",
		"name VARCHAR(255) NOT NULL UNIQUE",
		"api_key VARCHAR(255) NOT NULL UNIQUE",
		"smtp_host", "smtp_port", "smtp_user", "smtp_password", "smtp_from",
		"wa_token", "wa_phone_id",
		"sms_provider", "sms_api_key", "sms_sender_id",
		"active BOOLEAN",
		"created_at TIMESTAMP",
		"updated_at TIMESTAMP",
	}

	for _, field := range requiredFields {
		if !contains(m1.Up, field) {
			t.Errorf("Migration 1 Up SQL missing: %s", field)
		}
	}

	// Verify indexes
	requiredIndexes := []string{
		"idx_tenants_api_key",
		"idx_tenants_active",
	}

	for _, index := range requiredIndexes {
		if !contains(m1.Up, index) {
			t.Errorf("Migration 1 Up SQL missing index: %s", index)
		}
	}

	// Verify Down SQL
	if !contains(m1.Down, "DROP TABLE IF EXISTS tenants CASCADE") {
		t.Error("Migration 1 Down SQL doesn't drop tenants table with CASCADE")
	}

	// Test migration 2: templates table
	m2 := migrations[1]
	if m2.Version != 2 {
		t.Errorf("Migration 2 has wrong version: %d", m2.Version)
	}
	if m2.Name != "create_templates_table" {
		t.Errorf("Migration 2 has wrong name: %s", m2.Name)
	}

	// Verify foreign key constraint
	if !contains(m2.Up, "REFERENCES tenants(id) ON DELETE CASCADE") {
		t.Error("Migration 2 missing foreign key constraint with CASCADE")
	}

	// Verify JSONB support
	if !contains(m2.Up, "variables JSONB") {
		t.Error("Migration 2 missing JSONB field for variables")
	}

	// Verify unique constraint
	if !contains(m2.Up, "UNIQUE(tenant_id, channel, name)") {
		t.Error("Migration 2 missing unique constraint on tenant_id, channel, name")
	}

	// Test migration 3: notifications table
	m3 := migrations[2]
	if m3.Version != 3 {
		t.Errorf("Migration 3 has wrong version: %d", m3.Version)
	}
	if m3.Name != "create_notifications_table" {
		t.Errorf("Migration 3 has wrong name: %s", m3.Name)
	}

	// Verify status tracking
	if !contains(m3.Up, "status VARCHAR(50) NOT NULL DEFAULT 'pending'") {
		t.Error("Migration 3 missing status field with default")
	}

	// Verify JSONB for data
	if !contains(m3.Up, "data JSONB") {
		t.Error("Migration 3 missing JSONB field for data")
	}

	// Verify critical indexes for queries
	criticalIndexes := []string{
		"idx_notifications_tenant",
		"idx_notifications_status",
		"idx_notifications_channel",
		"idx_notifications_created_at",
		"idx_notifications_sent_at",
	}

	for _, index := range criticalIndexes {
		if !contains(m3.Up, index) {
			t.Errorf("Migration 3 missing critical index: %s", index)
		}
	}
}

func TestMigration_DownSafety(t *testing.T) {
	migrations := GetMigrations()

	for _, m := range migrations {
		// Verify all Down migrations use IF EXISTS
		if !contains(m.Down, "IF EXISTS") {
			t.Errorf("Migration %d Down SQL doesn't use IF EXISTS (not idempotent)", m.Version)
		}

		// Verify CASCADE is used for tables with foreign keys
		if m.Version == 1 { // tenants has dependent tables
			if !contains(m.Down, "CASCADE") {
				t.Error("Migration 1 Down SQL should use CASCADE (templates/notifications depend on it)")
			}
		}
	}
}

func TestMigration_SequentialVersions(t *testing.T) {
	migrations := GetMigrations()

	for i := 0; i < len(migrations)-1; i++ {
		current := migrations[i].Version
		next := migrations[i+1].Version

		if next != current+1 {
			t.Errorf("Migrations not sequential: version %d followed by %d", current, next)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
