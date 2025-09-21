package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatabase(t *testing.T) {
	t.Parallel()

	t.Run("creates database with default config", func(t *testing.T) {
		t.Parallel()

		// Create temporary database file
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")

		config := DefaultConfig()
		config.DatabasePath = dbPath

		db, err := NewDatabase(config)
		require.NoError(t, err)
		defer db.Close()

		assert.NotNil(t, db)
		assert.NotNil(t, db.DB())
	})

	t.Run("creates database with custom config", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "custom.db")

		config := Config{
			DatabasePath:      dbPath,
			MaxOpenConns:      10,
			MaxIdleConns:      2,
			ConnMaxLifetime:   2 * time.Minute,
			ConnMaxIdleTime:   30 * time.Second,
			BusyTimeout:       10 * time.Second,
			EnableWAL:         false,
			EnableForeignKeys: false,
		}

		db, err := NewDatabase(config)
		require.NoError(t, err)
		defer db.Close()

		assert.NotNil(t, db)

		// Verify connection pool settings
		stats := db.GetStats()
		assert.Equal(t, 10, stats.MaxOpenConnections)
	})
}

func TestDatabase_Initialize(t *testing.T) {
	t.Parallel()

	t.Run("initializes database successfully", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		ctx := context.Background()
		err := db.Initialize(ctx)
		require.NoError(t, err)

		// Verify migrations table was created
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 1) // At least one migration should be applied

		// Verify payments table was created
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM payments").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count) // Should be empty initially
	})

	t.Run("handles initialization errors gracefully", func(t *testing.T) {
		t.Parallel()

		// Create database with invalid path to trigger error
		config := DefaultConfig()
		config.DatabasePath = "/invalid/path/test.db"

		db, err := NewDatabase(config)
		require.NoError(t, err) // NewDatabase doesn't fail immediately
		defer db.Close()

		ctx := context.Background()
		err = db.Initialize(ctx)
		assert.Error(t, err)
	})
}

func TestDatabase_HealthCheck(t *testing.T) {
	t.Parallel()

	t.Run("passes health check for healthy database", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		ctx := context.Background()
		err := db.Initialize(ctx)
		require.NoError(t, err)

		err = db.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("fails health check for closed database", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		
		ctx := context.Background()
		err := db.Initialize(ctx)
		require.NoError(t, err)

		// Close the database
		db.Close()

		// Health check should fail
		err = db.HealthCheck(ctx)
		assert.Error(t, err)
	})

	t.Run("fails health check with context timeout", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		ctx := context.Background()
		err := db.Initialize(ctx)
		require.NoError(t, err)

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()

		// Wait for context to timeout
		time.Sleep(1 * time.Millisecond)

		err = db.HealthCheck(ctx)
		assert.Error(t, err)
	})
}

func TestDatabase_GetMigrationStatus(t *testing.T) {
	t.Parallel()

	t.Run("returns migration status", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		ctx := context.Background()
		err := db.Initialize(ctx)
		require.NoError(t, err)

		migrations, err := db.GetMigrationStatus(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, migrations)

		// Check that at least one migration is applied
		var appliedCount int
		for _, migration := range migrations {
			if migration.AppliedAt != nil {
				appliedCount++
			}
		}
		assert.GreaterOrEqual(t, appliedCount, 1)
	})
}

func TestDatabase_ConnectionPooling(t *testing.T) {
	t.Parallel()

	t.Run("respects connection pool limits", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "pool_test.db")

		config := DefaultConfig()
		config.DatabasePath = dbPath
		config.MaxOpenConns = 2
		config.MaxIdleConns = 1

		db, err := NewDatabase(config)
		require.NoError(t, err)
		defer db.Close()

		ctx := context.Background()
		err = db.Initialize(ctx)
		require.NoError(t, err)

		// Get initial stats
		stats := db.GetStats()
		assert.Equal(t, 2, stats.MaxOpenConnections)

		// Execute multiple queries to test connection pooling
		for i := 0; i < 5; i++ {
			var result int
			err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
			require.NoError(t, err)
			assert.Equal(t, 1, result)
		}

		// Check stats after queries
		stats = db.GetStats()
		assert.LessOrEqual(t, stats.OpenConnections, 2)
	})
}

func TestDatabase_TransactionSupport(t *testing.T) {
	t.Parallel()

	t.Run("supports transactions", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		ctx := context.Background()
		err := db.Initialize(ctx)
		require.NoError(t, err)

		// Start transaction
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		// Execute query within transaction
		_, err = tx.ExecContext(ctx, `
			INSERT INTO payments (
				id, debtor_iban, debtor_name, creditor_iban, creditor_name,
				amount_cents, currency, idempotency_key, status, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "test_id", "DE89370400440532013000", "Test User", "FR1420041010050500013M02606", 
		   "Test Recipient", 1000, "EUR", "test123456", "PENDING", time.Now(), time.Now())
		require.NoError(t, err)

		// Rollback transaction
		err = tx.Rollback()
		require.NoError(t, err)

		// Verify data was not persisted
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM payments WHERE id = ?", "test_id").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

// createTestDatabase creates a test database instance with a temporary file
func createTestDatabase(t *testing.T) *Database {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := DefaultConfig()
	config.DatabasePath = dbPath

	db, err := NewDatabase(config)
	require.NoError(t, err)

	return &db
}
