package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/huzaifabhutta/notify-core/internal/config"
	"github.com/huzaifabhutta/notify-core/internal/database"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Parse command line flags
	action := flag.String("action", "up", "Migration action: up, down, or status")
	flag.Parse()

	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Connect to database
	dbCfg := &database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
		MaxConns: cfg.Database.MaxConns,
		MaxIdle:  cfg.Database.MaxIdle,
	}

	db, err := database.New(dbCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	log.Info().
		Str("host", cfg.Database.Host).
		Int("port", cfg.Database.Port).
		Str("database", cfg.Database.DBName).
		Msg("Connected to database")

	// Create migrator
	migrator := database.NewMigrator(db)

	ctx := context.Background()

	// Execute migration action
	switch *action {
	case "up":
		if err := runUp(ctx, migrator); err != nil {
			log.Fatal().Err(err).Msg("Migration up failed")
		}
		log.Info().Msg("✅ All migrations applied successfully")

	case "down":
		if err := runDown(ctx, migrator); err != nil {
			log.Fatal().Err(err).Msg("Migration down failed")
		}
		log.Info().Msg("✅ Migration rolled back successfully")

	case "status":
		if err := showStatus(ctx, db); err != nil {
			log.Fatal().Err(err).Msg("Failed to show migration status")
		}

	default:
		log.Fatal().Str("action", *action).Msg("Invalid action. Use: up, down, or status")
	}
}

// runUp runs all pending migrations
func runUp(ctx context.Context, migrator *database.Migrator) error {
	fmt.Println("Running migrations...")
	return migrator.Up(ctx)
}

// runDown rolls back the last migration
func runDown(ctx context.Context, migrator *database.Migrator) error {
	fmt.Println("Rolling back last migration...")
	return migrator.Down(ctx)
}

// showStatus shows current migration status
func showStatus(ctx context.Context, db *database.DB) error {
	// Get current version
	var version int
	err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		// Table might not exist yet
		version = 0
	}

	fmt.Println("Migration Status")
	fmt.Println("================")
	fmt.Printf("Current version: %d\n", version)

	// Get all applied migrations
	rows, err := db.QueryContext(ctx, `
		SELECT version, name, applied_at
		FROM schema_migrations
		ORDER BY version ASC
	`)
	if err != nil {
		// Table might not exist yet
		fmt.Println("\nNo migrations applied yet")
		return nil
	}
	defer rows.Close()

	fmt.Println("\nApplied migrations:")
	for rows.Next() {
		var v int
		var name string
		var appliedAt time.Time
		if err := rows.Scan(&v, &name, &appliedAt); err != nil {
			return err
		}
		fmt.Printf("  %d: %s (applied at %s)\n", v, name, appliedAt.Format(time.RFC3339))
	}

	// Show available migrations
	migrations := database.GetMigrations()
	fmt.Printf("\nAvailable migrations: %d\n", len(migrations))
	for _, m := range migrations {
		status := "pending"
		if m.Version <= version {
			status = "applied"
		}
		fmt.Printf("  %d: %s [%s]\n", m.Version, m.Name, status)
	}

	return nil
}
