package database

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	Up      string
	Down    string
}

// Migrator handles database migrations
type Migrator struct {
	db         *DB
	migrations []Migration
}

// NewMigrator creates a new migrator
func NewMigrator(db *DB) *Migrator {
	return &Migrator{
		db:         db,
		migrations: GetMigrations(),
	}
}

// Up runs all pending migrations
func (m *Migrator) Up(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := m.getCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Sort migrations by version
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})

	// Run pending migrations
	for _, migration := range m.migrations {
		if migration.Version <= currentVersion {
			continue
		}

		if err := m.runMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to run migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		fmt.Printf("✓ Applied migration %d: %s\n", migration.Version, migration.Name)
	}

	return nil
}

// Down rolls back the last migration
func (m *Migrator) Down(ctx context.Context) error {
	currentVersion, err := m.getCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if currentVersion == 0 {
		return fmt.Errorf("no migrations to roll back")
	}

	// Find migration to roll back
	var migration *Migration
	for i := range m.migrations {
		if m.migrations[i].Version == currentVersion {
			migration = &m.migrations[i]
			break
		}
	}

	if migration == nil {
		return fmt.Errorf("migration %d not found", currentVersion)
	}

	// Run down migration
	tx, err := m.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, migration.Down); err != nil {
		return fmt.Errorf("failed to execute down migration: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM schema_migrations WHERE version = $1", currentVersion); err != nil {
		return fmt.Errorf("failed to delete migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("✓ Rolled back migration %d: %s\n", migration.Version, migration.Name)
	return nil
}

// createMigrationsTable creates the schema_migrations table
func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := m.db.ExecContext(ctx, query)
	return err
}

// getCurrentVersion gets the current migration version
func (m *Migrator) getCurrentVersion(ctx context.Context) (int, error) {
	var version int
	err := m.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// runMigration runs a single migration
func (m *Migrator) runMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration
	if _, err := tx.ExecContext(ctx, migration.Up); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Record migration
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, $3)",
		migration.Version, migration.Name, time.Now(),
	); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// GetMigrations returns all available migrations
func GetMigrations() []Migration {
	return []Migration{
		{
			Version: 1,
			Name:    "create_tenants_table",
			Up: `
				CREATE TABLE tenants (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL UNIQUE,
					api_key VARCHAR(255) NOT NULL UNIQUE,

					-- Email configuration
					smtp_host VARCHAR(255),
					smtp_port INT,
					smtp_user VARCHAR(255),
					smtp_password VARCHAR(255),
					smtp_from VARCHAR(255),

					-- WhatsApp configuration
					wa_token TEXT,
					wa_phone_id VARCHAR(255),

					-- SMS configuration
					sms_provider VARCHAR(50),
					sms_api_key TEXT,
					sms_sender_id VARCHAR(255),

					-- Status
					active BOOLEAN NOT NULL DEFAULT true,

					-- Timestamps
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				);

				CREATE INDEX idx_tenants_api_key ON tenants(api_key);
				CREATE INDEX idx_tenants_active ON tenants(active);
			`,
			Down: `DROP TABLE IF EXISTS tenants CASCADE;`,
		},
		{
			Version: 2,
			Name:    "create_templates_table",
			Up: `
				CREATE TABLE templates (
					id SERIAL PRIMARY KEY,
					tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE,
					channel VARCHAR(50) NOT NULL,
					name VARCHAR(255) NOT NULL,
					subject VARCHAR(500),
					body TEXT NOT NULL,

					-- Template metadata
					variables JSONB,
					active BOOLEAN NOT NULL DEFAULT true,

					-- Timestamps
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

					UNIQUE(tenant_id, channel, name)
				);

				CREATE INDEX idx_templates_tenant ON templates(tenant_id);
				CREATE INDEX idx_templates_channel ON templates(channel);
				CREATE INDEX idx_templates_active ON templates(active);
			`,
			Down: `DROP TABLE IF EXISTS templates CASCADE;`,
		},
		{
			Version: 3,
			Name:    "create_notifications_table",
			Up: `
				CREATE TABLE notifications (
					id SERIAL PRIMARY KEY,
					tenant_id INT REFERENCES tenants(id) ON DELETE CASCADE,

					-- Notification details
					channel VARCHAR(50) NOT NULL,
					recipient VARCHAR(255) NOT NULL,
					template_name VARCHAR(255),
					subject VARCHAR(500),

					-- Status tracking
					status VARCHAR(50) NOT NULL DEFAULT 'pending',
					message_id VARCHAR(255),
					error TEXT,

					-- Metadata
					data JSONB,

					-- Timestamps
					sent_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				);

				CREATE INDEX idx_notifications_tenant ON notifications(tenant_id);
				CREATE INDEX idx_notifications_status ON notifications(status);
				CREATE INDEX idx_notifications_channel ON notifications(channel);
				CREATE INDEX idx_notifications_created_at ON notifications(created_at);
				CREATE INDEX idx_notifications_sent_at ON notifications(sent_at);
				CREATE INDEX idx_notifications_tenant_created ON notifications(tenant_id, created_at DESC);
			`,
			Down: `DROP TABLE IF EXISTS notifications CASCADE;`,
		},
		{
			Version: 4,
			Name:    "add_whatsapp_vendor_agnostic_fields",
			Up: `
				-- Add vendor-agnostic WhatsApp configuration fields
				-- Allows tenants to use any WhatsApp Business API provider (Facebook, 360dialog, Twilio, etc.)
				ALTER TABLE tenants
					ADD COLUMN wa_base_url VARCHAR(255),
					ADD COLUMN wa_api_version VARCHAR(50);

				-- Set default values for existing rows (Facebook Cloud API)
				UPDATE tenants
				SET wa_base_url = 'https://graph.facebook.com',
					wa_api_version = 'v21.0'
				WHERE wa_token IS NOT NULL AND wa_token != '';

				-- Add comment for documentation
				COMMENT ON COLUMN tenants.wa_base_url IS 'WhatsApp API base URL (e.g., https://graph.facebook.com, https://waba.360dialog.io)';
				COMMENT ON COLUMN tenants.wa_api_version IS 'WhatsApp API version (e.g., v21.0)';
			`,
			Down: `
				ALTER TABLE tenants
					DROP COLUMN IF EXISTS wa_base_url,
					DROP COLUMN IF EXISTS wa_api_version;
			`,
		},
	}
}
