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

type Migration struct {
	Version   int
	SQL       string
	AppliedAt *time.Time
}

type Migrator struct {
	db *sql.DB
}

func NewMigrator(db *sql.DB) Migrator {
	return Migrator{db: db}
}

func (m *Migrator) Migrate(ctx context.Context) error {
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	availableMigrations, err := m.getAvailableMigrations()
	if err != nil {
		return fmt.Errorf("failed to get available migrations: %w", err)
	}

	appliedMigrations, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	pendingMigrations := m.findPendingMigrations(availableMigrations, appliedMigrations)

	for _, migration := range pendingMigrations {
		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}
	}

	return nil
}

func (m *Migrator) GetMigrationStatus(ctx context.Context) ([]Migration, error) {
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

	migrationMap := make(map[int]Migration)

	for _, migration := range availableMigrations {
		migrationMap[migration.Version] = migration
	}

	for _, applied := range appliedMigrations {
		if migration, exists := migrationMap[applied.Version]; exists {
			migration.AppliedAt = applied.AppliedAt
		}
	}

	var result []Migration
	for _, migration := range migrationMap {
		result = append(result, migration)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at 
		ON schema_migrations(applied_at);
	`

	_, err := m.db.ExecContext(ctx, query)
	return err
}

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

		if strings.Contains(entry.Name(), "test_data") {
			continue
		}

		migration, err := m.parseMigrationFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to parse migration file %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (m *Migrator) parseMigrationFile(filename string) (Migration, error) {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) != 2 {
		return Migration{}, fmt.Errorf("invalid migration filename format: %s", filename)
	}

	var version int
	if _, err := fmt.Sscanf(parts[0], "%03d", &version); err != nil {
		return Migration{}, fmt.Errorf("failed to parse version from filename %s: %w", filename, err)
	}

	sqlBytes, err := migrationFiles.ReadFile(filepath.Join("migrations", filename))
	if err != nil {
		return Migration{}, fmt.Errorf("failed to read migration file %s: %w", filename, err)
	}

	return Migration{
		Version: version,
		SQL:     string(sqlBytes),
	}, nil
}

func (m *Migrator) getAppliedMigrations(ctx context.Context) ([]Migration, error) {
	query := `
		SELECT version, applied_at 
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

		err := rows.Scan(&migration.Version, &appliedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}

		migration.AppliedAt = &appliedAt
		migrations = append(migrations, migration)
	}

	return migrations, rows.Err()
}

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

func (m *Migrator) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	insertQuery := `INSERT INTO schema_migrations (version) VALUES (?)`
	if _, err := tx.ExecContext(ctx, insertQuery, migration.Version); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}
