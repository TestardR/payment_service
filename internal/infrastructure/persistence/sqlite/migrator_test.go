package sqlite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrator_Migrate(t *testing.T) {
	t.Parallel()

	t.Run("runs migrations successfully", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		migrator := NewMigrator(db.DB())
		ctx := context.Background()

		err := migrator.Migrate(ctx)
		require.NoError(t, err)

		// Verify migrations table was created
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 1)

		// Verify payments table was created
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM payments").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count) // Should be empty initially
	})

	t.Run("is idempotent - running twice doesn't cause errors", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		migrator := NewMigrator(db.DB())
		ctx := context.Background()

		// Run migrations first time
		err := migrator.Migrate(ctx)
		require.NoError(t, err)

		// Run migrations second time
		err = migrator.Migrate(ctx)
		require.NoError(t, err)

		// Verify only one migration record exists
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count) // Should have exactly one migration record
	})
}

func TestMigrator_GetMigrationStatus(t *testing.T) {
	t.Parallel()

	t.Run("returns migration status", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		migrator := NewMigrator(db.DB())
		ctx := context.Background()

		// Get status before migrations
		statusBefore, err := migrator.GetMigrationStatus(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, statusBefore)

		// Check that no migrations are applied initially
		for _, migration := range statusBefore {
			assert.Nil(t, migration.AppliedAt, "Migration %s should not be applied initially", migration.Name)
		}

		// Run migrations
		err = migrator.Migrate(ctx)
		require.NoError(t, err)

		// Get status after migrations
		statusAfter, err := migrator.GetMigrationStatus(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, statusAfter)

		// Check that at least one migration is now applied
		var appliedCount int
		for _, migration := range statusAfter {
			if migration.AppliedAt != nil {
				appliedCount++
				assert.NotEmpty(t, migration.Checksum, "Applied migration should have checksum")
			}
		}
		assert.GreaterOrEqual(t, appliedCount, 1, "At least one migration should be applied")
	})

	t.Run("handles empty database", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		migrator := NewMigrator(db.DB())
		ctx := context.Background()

		// Get status without running migrations first
		status, err := migrator.GetMigrationStatus(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, status)

		// All migrations should be unapplied
		for _, migration := range status {
			assert.Nil(t, migration.AppliedAt)
			assert.Empty(t, migration.Checksum)
		}
	})
}

func TestMigrator_getAvailableMigrations(t *testing.T) {
	t.Parallel()

	t.Run("reads available migrations from embedded files", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		migrator := NewMigrator(db.DB())

		migrations, err := migrator.getAvailableMigrations()
		require.NoError(t, err)
		assert.NotEmpty(t, migrations)

		// Check that migrations are sorted by version
		for i := 1; i < len(migrations); i++ {
			assert.Greater(t, migrations[i].Version, migrations[i-1].Version,
				"Migrations should be sorted by version")
		}

		// Check that each migration has required fields
		for _, migration := range migrations {
			assert.Greater(t, migration.Version, 0, "Migration version should be positive")
			assert.NotEmpty(t, migration.Name, "Migration name should not be empty")
			assert.NotEmpty(t, migration.SQL, "Migration SQL should not be empty")
			assert.Nil(t, migration.AppliedAt, "Available migration should not have AppliedAt set")
		}
	})
}

func TestMigrator_parseMigrationFile(t *testing.T) {
	t.Parallel()

	t.Run("parses valid migration filename", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		migrator := NewMigrator(db.DB())

		migration, err := migrator.parseMigrationFile("001_create_payments_table.sql")
		require.NoError(t, err)

		assert.Equal(t, 1, migration.Version)
		assert.Equal(t, "create_payments_table", migration.Name)
		assert.NotEmpty(t, migration.SQL)
		assert.Contains(t, migration.SQL, "CREATE TABLE")
		assert.Contains(t, migration.SQL, "payments")
	})

	t.Run("returns error for invalid filename format", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		migrator := NewMigrator(db.DB())

		_, err := migrator.parseMigrationFile("invalid_filename.sql")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected integer")
	})
}

func TestMigrator_calculateChecksum(t *testing.T) {
	t.Parallel()

	t.Run("generates consistent checksums", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		migrator := NewMigrator(db.DB())

		content := "CREATE TABLE test (id INTEGER PRIMARY KEY);"
		
		checksum1 := migrator.calculateChecksum(content)
		checksum2 := migrator.calculateChecksum(content)
		
		assert.Equal(t, checksum1, checksum2, "Checksums should be consistent")
		assert.NotEmpty(t, checksum1, "Checksum should not be empty")
	})

	t.Run("generates different checksums for different content", func(t *testing.T) {
		t.Parallel()

		db := createTestDatabase(t)
		defer db.Close()

		migrator := NewMigrator(db.DB())

		content1 := "CREATE TABLE test1 (id INTEGER PRIMARY KEY);"
		content2 := "CREATE TABLE test2 (id INTEGER PRIMARY KEY);"
		
		checksum1 := migrator.calculateChecksum(content1)
		checksum2 := migrator.calculateChecksum(content2)
		
		assert.NotEqual(t, checksum1, checksum2, "Different content should have different checksums")
	})
}
