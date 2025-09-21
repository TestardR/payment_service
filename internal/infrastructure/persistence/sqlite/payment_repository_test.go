package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"paymentprocessor/internal/domain/payment"
	"paymentprocessor/internal/domain/shared"
)

func TestPaymentRepository_Save(t *testing.T) {
	t.Parallel()

	t.Run("saves payment successfully", func(t *testing.T) {
		t.Parallel()

		repo, db := createTestRepository(t)
		defer db.Close()

		ctx := context.Background()
		testPayment := createTestPayment(t)

		err := repo.Save(ctx, testPayment)
		require.NoError(t, err)

		// Verify payment was saved
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM payments WHERE id = ?", testPayment.ID()).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("returns error for duplicate idempotency key", func(t *testing.T) {
		t.Parallel()

		repo, db := createTestRepository(t)
		defer db.Close()

		ctx := context.Background()
		testPayment1 := createTestPayment(t)
		testPayment2 := createTestPaymentWithIdempotencyKey(t, testPayment1.IdempotencyKey())

		// Save first payment
		err := repo.Save(ctx, testPayment1)
		require.NoError(t, err)

		// Try to save second payment with same idempotency key
		err = repo.Save(ctx, testPayment2)
		assert.ErrorIs(t, err, shared.ErrDuplicateIdempotencyKey)
	})

}

func TestPaymentRepository_FindByID(t *testing.T) {
	t.Parallel()

	t.Run("finds existing payment by ID", func(t *testing.T) {
		t.Parallel()

		repo, db := createTestRepository(t)
		defer db.Close()

		ctx := context.Background()
		testPayment := createTestPayment(t)

		// Save payment first
		err := repo.Save(ctx, testPayment)
		require.NoError(t, err)

		// Find payment by ID
		foundPayment, err := repo.FindByID(ctx, testPayment.ID())
		require.NoError(t, err)
		require.NotNil(t, foundPayment)

		// Verify payment data
		assert.Equal(t, testPayment.ID(), foundPayment.ID())
		assert.Equal(t, testPayment.DebtorIBAN().Value(), foundPayment.DebtorIBAN().Value())
		assert.Equal(t, testPayment.DebtorName(), foundPayment.DebtorName())
		assert.Equal(t, testPayment.CreditorIBAN().Value(), foundPayment.CreditorIBAN().Value())
		assert.Equal(t, testPayment.CreditorName(), foundPayment.CreditorName())
		assert.Equal(t, testPayment.Amount().Cents(), foundPayment.Amount().Cents())
		assert.Equal(t, testPayment.IdempotencyKey().Value(), foundPayment.IdempotencyKey().Value())
		assert.Equal(t, testPayment.Status(), foundPayment.Status())
	})

	t.Run("returns error for non-existent payment", func(t *testing.T) {
		t.Parallel()

		repo, db := createTestRepository(t)
		defer db.Close()

		ctx := context.Background()
		foundPayment, err := repo.FindByID(ctx, "non-existent-id")
		assert.ErrorIs(t, err, shared.ErrPaymentNotFound)
		assert.Equal(t, payment.Payment{}, foundPayment)
	})

	t.Run("finds payment with different statuses", func(t *testing.T) {
		t.Parallel()

		repo, db := createTestRepository(t)
		defer db.Close()

		ctx := context.Background()

		// Test with processed payment
		testPayment := createTestPayment(t)
		err := testPayment.MarkAsProcessed(time.Now())
		require.NoError(t, err)

		err = repo.Save(ctx, testPayment)
		require.NoError(t, err)

		foundPayment, err := repo.FindByID(ctx, testPayment.ID())
		require.NoError(t, err)
		require.NotNil(t, foundPayment)
		assert.Equal(t, payment.StatusProcessed, foundPayment.Status())
	})
}

func TestPaymentRepository_FindByIdempotencyKey(t *testing.T) {
	t.Parallel()

	t.Run("finds existing payment by idempotency key", func(t *testing.T) {
		t.Parallel()

		repo, db := createTestRepository(t)
		defer db.Close()

		ctx := context.Background()
		testPayment := createTestPayment(t)

		// Save payment first
		err := repo.Save(ctx, testPayment)
		require.NoError(t, err)

		// Find payment by idempotency key
		foundPayment, err := repo.FindByIdempotencyKey(ctx, testPayment.IdempotencyKey())
		require.NoError(t, err)
		require.NotNil(t, foundPayment)

		// Verify payment data
		assert.Equal(t, testPayment.ID(), foundPayment.ID())
		assert.Equal(t, testPayment.IdempotencyKey().Value(), foundPayment.IdempotencyKey().Value())
	})

	t.Run("returns error for non-existent idempotency key", func(t *testing.T) {
		t.Parallel()

		repo, db := createTestRepository(t)
		defer db.Close()

		ctx := context.Background()
		nonExistentKey, err := shared.NewIdempotencyKey("nonexist01")
		require.NoError(t, err)

		foundPayment, err := repo.FindByIdempotencyKey(ctx, nonExistentKey)
		assert.ErrorIs(t, err, shared.ErrPaymentNotFound)
		assert.Equal(t, payment.Payment{}, foundPayment)
	})
}

func TestPaymentRepository_UpdateStatus(t *testing.T) {
	t.Parallel()

	t.Run("updates payment status successfully", func(t *testing.T) {
		t.Parallel()

		repo, db := createTestRepository(t)
		defer db.Close()

		ctx := context.Background()
		testPayment := createTestPayment(t)

		// Save payment first
		err := repo.Save(ctx, testPayment)
		require.NoError(t, err)

		// Update status
		err = repo.UpdateStatus(ctx, testPayment.ID(), payment.StatusProcessed)
		require.NoError(t, err)

		// Verify status was updated in database
		var status string
		err = db.QueryRowContext(ctx, "SELECT status FROM payments WHERE id = ?", testPayment.ID()).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, string(payment.StatusProcessed), status)
	})

	t.Run("returns error for non-existent payment", func(t *testing.T) {
		t.Parallel()

		repo, db := createTestRepository(t)
		defer db.Close()

		ctx := context.Background()
		err := repo.UpdateStatus(ctx, "non-existent-id", payment.StatusProcessed)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// createTestRepository creates a test repository with an initialized database
func createTestRepository(t *testing.T) (PaymentRepository, *Database) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_repo.db")

	config := DefaultConfig()
	config.DatabasePath = dbPath

	db, err := NewDatabase(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Initialize(ctx)
	require.NoError(t, err)

	repo := NewPaymentRepository(db)
	return repo, &db
}

// createTestPayment creates a test payment with valid data
func createTestPayment(t *testing.T) payment.Payment {
	return createTestPaymentWithID(t, "test_payment_001")
}

// createTestPaymentWithID creates a test payment with a specific ID
func createTestPaymentWithID(t *testing.T, id string) payment.Payment {
	debtorIBAN, err := shared.NewIBAN("DE89370400440532013000")
	require.NoError(t, err)

	creditorIBAN, err := shared.NewIBAN("FR1420041010050500013M02606")
	require.NoError(t, err)

	amount, err := shared.NewAmountFromCents(10050) // €100.50
	require.NoError(t, err)

	// Create a valid 10-character idempotency key
	// Use a simple hash to ensure uniqueness
	hash := 0
	for _, c := range id {
		hash = hash*31 + int(c)
	}
	keyValue := fmt.Sprintf("test%06d", hash%1000000)
	idempotencyKey, err := shared.NewIdempotencyKey(keyValue)
	require.NoError(t, err)

	now := time.Now().UTC() // Use UTC to match SQLite's CURRENT_TIMESTAMP
	testPayment, err := payment.NewPayment(
		id,
		debtorIBAN,
		"John Doe",
		creditorIBAN,
		"Jane Smith",
		amount,
		idempotencyKey,
		now,
		now,
	)
	require.NoError(t, err)

	return testPayment
}

// createTestPaymentWithIdempotencyKey creates a test payment with a specific idempotency key
func createTestPaymentWithIdempotencyKey(t *testing.T, key shared.IdempotencyKey) payment.Payment {
	debtorIBAN, err := shared.NewIBAN("DE89370400440532013000")
	require.NoError(t, err)

	creditorIBAN, err := shared.NewIBAN("FR1420041010050500013M02606")
	require.NoError(t, err)

	amount, err := shared.NewAmountFromCents(10050) // €100.50
	require.NoError(t, err)

	now := time.Now()
	testPayment, err := payment.NewPayment(
		"test_payment_duplicate",
		debtorIBAN,
		"John Doe",
		creditorIBAN,
		"Jane Smith",
		amount,
		key,
		now,
		now,
	)
	require.NoError(t, err)

	return testPayment
}
