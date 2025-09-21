package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migration represents a database migration
type Migration struct {
	Version     int
	Name        string
	SQL         string
	AppliedAt   *time.Time
	Checksum    string
}

// Migrator handles database migrations
type Migrator struct {
	db *sql.DB
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

// Migrate runs all pending migrations
func (m *Migrator) Migrate(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get available migrations
	availableMigrations, err := m.getAvailableMigrations()
	if err != nil {
		return fmt.Errorf("failed to get available migrations: %w", err)
	}

	// Get applied migrations
	appliedMigrations, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Find pending migrations
	pendingMigrations := m.findPendingMigrations(availableMigrations, appliedMigrations)

	// Apply pending migrations
	for _, migration := range pendingMigrations {
		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Name, err)
		}
	}

	return nil
}

// GetMigrationStatus returns the status of all migrations
func (m *Migrator) GetMigrationStatus(ctx context.Context) ([]Migration, error) {
	// Ensure migrations table exists
	if err := m.createMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	availableMigrations, err := m.getAvailableMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to get available migrations: %w", err)
	}

	appliedMigrations, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Merge available and applied migrations
	migrationMap := make(map[int]*Migration)
	
	// Add available migrations
	for _, migration := range availableMigrations {
		migrationMap[migration.Version] = &migration
	}
	
	// Update with applied status
	for _, applied := range appliedMigrations {
		if migration, exists := migrationMap[applied.Version]; exists {
			migration.AppliedAt = applied.AppliedAt
			migration.Checksum = applied.Checksum
		}
	}

	// Convert map to slice and sort
	var result []Migration
	for _, migration := range migrationMap {
		result = append(result, *migration)
	}
	
	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

// createMigrationsTable creates the migrations tracking table
func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			checksum TEXT NOT NULL,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at 
		ON schema_migrations(applied_at);
	`
	
	_, err := m.db.ExecContext(ctx, query)
	return err
}

// getAvailableMigrations reads all migration files from the embedded filesystem
func (m *Migrator) getAvailableMigrations() ([]Migration, error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Skip test data files
		if strings.Contains(entry.Name(), "test_data") {
			continue
		}

		migration, err := m.parseMigrationFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to parse migration file %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, migration)
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFile parses a migration file and extracts version, name, and SQL
func (m *Migrator) parseMigrationFile(filename string) (Migration, error) {
	// Parse version from filename (e.g., "001_create_payments_table.sql")
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) != 2 {
		return Migration{}, fmt.Errorf("invalid migration filename format: %s", filename)
	}

	var version int
	if _, err := fmt.Sscanf(parts[0], "%03d", &version); err != nil {
		return Migration{}, fmt.Errorf("failed to parse version from filename %s: %w", filename, err)
	}

	// Extract name (remove version prefix and .sql suffix)
	name := strings.TrimSuffix(parts[1], ".sql")

	// Read SQL content
	sqlBytes, err := migrationFiles.ReadFile(filepath.Join("migrations", filename))
	if err != nil {
		return Migration{}, fmt.Errorf("failed to read migration file %s: %w", filename, err)
	}

	return Migration{
		Version: version,
		Name:    name,
		SQL:     string(sqlBytes),
	}, nil
}

// getAppliedMigrations retrieves all applied migrations from the database
func (m *Migrator) getAppliedMigrations(ctx context.Context) ([]Migration, error) {
	query := `
		SELECT version, name, checksum, applied_at 
		FROM schema_migrations 
		ORDER BY version
	`
	
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var migration Migration
		var appliedAt time.Time
		
		err := rows.Scan(&migration.Version, &migration.Name, &migration.Checksum, &appliedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		
		migration.AppliedAt = &appliedAt
		migrations = append(migrations, migration)
	}

	return migrations, rows.Err()
}

// findPendingMigrations compares available and applied migrations to find pending ones
func (m *Migrator) findPendingMigrations(available, applied []Migration) []Migration {
	appliedMap := make(map[int]bool)
	for _, migration := range applied {
		appliedMap[migration.Version] = true
	}

	var pending []Migration
	for _, migration := range available {
		if !appliedMap[migration.Version] {
			pending = append(pending, migration)
		}
	}

	return pending
}

// applyMigration applies a single migration within a transaction
func (m *Migrator) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration as applied
	checksum := m.calculateChecksum(migration.SQL)
	insertQuery := `
		INSERT INTO schema_migrations (version, name, checksum) 
		VALUES (?, ?, ?)
	`
	
	if _, err := tx.ExecContext(ctx, insertQuery, migration.Version, migration.Name, checksum); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// calculateChecksum calculates a simple checksum for migration content
func (m *Migrator) calculateChecksum(content string) string {
	// Simple checksum - in production, consider using a proper hash function
	var sum int64
	for _, char := range content {
		sum += int64(char)
	}
	return fmt.Sprintf("%x", sum)
}
