package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	DatabasePath      string
	MaxOpenConns      int
	MaxIdleConns      int
	ConnMaxLifetime   time.Duration
	ConnMaxIdleTime   time.Duration
	BusyTimeout       time.Duration
	EnableWAL         bool
	EnableForeignKeys bool
}

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

type Database struct {
	db       *sql.DB
	config   Config
	migrator Migrator
}

func NewDatabase(config Config) (Database, error) {
	dsn := buildDSN(config)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return Database{}, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	database := Database{
		db:       db,
		config:   config,
		migrator: NewMigrator(db),
	}

	return database, nil
}

func buildDSN(config Config) string {
	dsn := config.DatabasePath + "?"
	params := []string{
		fmt.Sprintf("_busy_timeout=%d", int(config.BusyTimeout.Milliseconds())),
		"_txlock=immediate",
		"_synchronous=NORMAL",
		"_cache_size=-64000",
	}

	if config.EnableWAL {
		params = append(params, "_journal_mode=WAL")
	}

	if config.EnableForeignKeys {
		params = append(params, "_foreign_keys=on")
	}

	for i, param := range params {
		if i > 0 {
			dsn += "&"
		}
		dsn += param
	}

	return dsn
}

func (d Database) Initialize(ctx context.Context) error {
	if err := d.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	if err := d.migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (d Database) DB() *sql.DB {
	return d.db
}

func (d Database) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func (d Database) HealthCheck(ctx context.Context) error {
	if err := d.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	var result int
	err := d.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("query test failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected query result: got %d, expected 1", result)
	}

	var count int
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM payments").Scan(&count)
	if err != nil {
		return fmt.Errorf("payments table check failed: %w", err)
	}

	return nil
}

func (d Database) GetStats() sql.DBStats {
	return d.db.Stats()
}

func (d Database) GetMigrationStatus(ctx context.Context) ([]Migration, error) {
	return d.migrator.GetMigrationStatus(ctx)
}

func (d Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d Database) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return d.db.BeginTx(ctx, opts)
}

func (d Database) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

func (d Database) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

func (d Database) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}
