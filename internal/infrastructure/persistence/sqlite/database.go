package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Config holds database configuration
type Config struct {
	// DatabasePath is the path to the SQLite database file
	DatabasePath string
	
	// MaxOpenConns is the maximum number of open connections to the database
	MaxOpenConns int
	
	// MaxIdleConns is the maximum number of connections in the idle connection pool
	MaxIdleConns int
	
	// ConnMaxLifetime is the maximum amount of time a connection may be reused
	ConnMaxLifetime time.Duration
	
	// ConnMaxIdleTime is the maximum amount of time a connection may be idle
	ConnMaxIdleTime time.Duration
	
	// BusyTimeout is the timeout for busy database operations
	BusyTimeout time.Duration
	
	// EnableWAL enables Write-Ahead Logging mode
	EnableWAL bool
	
	// EnableForeignKeys enables foreign key constraints
	EnableForeignKeys bool
}

// DefaultConfig returns a default database configuration
func DefaultConfig() Config {
	return Config{
		DatabasePath:      "payments.db",
		MaxOpenConns:      25,
		MaxIdleConns:      5,
		ConnMaxLifetime:   5 * time.Minute,
		ConnMaxIdleTime:   1 * time.Minute,
		BusyTimeout:       30 * time.Second,
		EnableWAL:         true,
		EnableForeignKeys: true,
	}
}

// Database wraps sql.DB with additional functionality
type Database struct {
	db       *sql.DB
	config   Config
	migrator *Migrator
}

// NewDatabase creates a new database connection with the given configuration
func NewDatabase(config Config) (*Database, error) {
	// Build connection string with SQLite-specific options
	dsn := buildDSN(config)
	
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	database := &Database{
		db:       db,
		config:   config,
		migrator: NewMigrator(db),
	}

	return database, nil
}

// buildDSN constructs the SQLite data source name with appropriate options
func buildDSN(config Config) string {
	dsn := config.DatabasePath + "?"
	
	// Add SQLite-specific parameters
	params := []string{
		fmt.Sprintf("_busy_timeout=%d", int(config.BusyTimeout.Milliseconds())),
		"_txlock=immediate",
		"_synchronous=NORMAL",
		"_cache_size=-64000", // 64MB cache
	}
	
	if config.EnableWAL {
		params = append(params, "_journal_mode=WAL")
	}
	
	if config.EnableForeignKeys {
		params = append(params, "_foreign_keys=on")
	}
	
	// Join parameters
	for i, param := range params {
		if i > 0 {
			dsn += "&"
		}
		dsn += param
	}
	
	return dsn
}

// Initialize sets up the database schema by running migrations
func (d *Database) Initialize(ctx context.Context) error {
	// Test connection
	if err := d.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := d.migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// DB returns the underlying sql.DB instance
func (d *Database) DB() *sql.DB {
	return d.db
}

// Ping verifies the database connection is alive
func (d *Database) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// HealthCheck performs a comprehensive health check of the database
func (d *Database) HealthCheck(ctx context.Context) error {
	// Check if we can ping the database
	if err := d.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Check if we can execute a simple query
	var result int
	err := d.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("query test failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected query result: got %d, expected 1", result)
	}

	// Check if payments table exists and is accessible
	var count int
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM payments").Scan(&count)
	if err != nil {
		return fmt.Errorf("payments table check failed: %w", err)
	}

	return nil
}

// GetStats returns database statistics
func (d *Database) GetStats() sql.DBStats {
	return d.db.Stats()
}

// GetMigrationStatus returns the status of all migrations
func (d *Database) GetMigrationStatus(ctx context.Context) ([]Migration, error) {
	return d.migrator.GetMigrationStatus(ctx)
}

// Close closes the database connection
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// BeginTx starts a new transaction with the given options
func (d *Database) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return d.db.BeginTx(ctx, opts)
}

// ExecContext executes a query without returning any rows
func (d *Database) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows
func (d *Database) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that is expected to return at most one row
func (d *Database) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}
