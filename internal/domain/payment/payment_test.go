package payment

import (
	"testing"
	"time"

	"paymentprocessor/internal/domain/shared"

	"github.com/stretchr/testify/assert"
)

func TestNewPayment(t *testing.T) {
	t.Parallel()
	// Setup valid test data
	debtorIBAN, _ := shared.NewIBAN("GB82WEST12345698765432")
	creditorIBAN, _ := shared.NewIBAN("FR1420041010050500013M02606")
	amount, _ := shared.NewAmount(100.50)
	idempotencyKey, _ := shared.NewIdempotencyKey("abc123XYZ0")
	now := time.Now()

	tests := []struct {
		name           string
		id             string
		debtorIBAN     shared.IBAN
		debtorName     string
		creditorIBAN   shared.IBAN
		creditorName   string
		amount         shared.Amount
		idempotencyKey shared.IdempotencyKey
		createdAt      time.Time
		updatedAt      time.Time
		expectError    bool
	}{
		{
			name:           "valid payment",
			id:             "payment-123",
			debtorIBAN:     debtorIBAN,
			debtorName:     "John Doe",
			creditorIBAN:   creditorIBAN,
			creditorName:   "Jane Smith",
			amount:         amount,
			idempotencyKey: idempotencyKey,
			createdAt:      now,
			updatedAt:      now,
			expectError:    false,
		},
		{
			name:           "invalid debtor name too short",
			id:             "payment-123",
			debtorIBAN:     debtorIBAN,
			debtorName:     "Jo",
			creditorIBAN:   creditorIBAN,
			creditorName:   "Jane Smith",
			amount:         amount,
			idempotencyKey: idempotencyKey,
			createdAt:      now,
			updatedAt:      now,
			expectError:    true,
		},
		{
			name:           "invalid creditor name too short",
			id:             "payment-123",
			debtorIBAN:     debtorIBAN,
			debtorName:     "John Doe",
			creditorIBAN:   creditorIBAN,
			creditorName:   "Ja",
			amount:         amount,
			idempotencyKey: idempotencyKey,
			createdAt:      now,
			updatedAt:      now,
			expectError:    true,
		},
		{
			name:           "invalid zero amount",
			id:             "payment-123",
			debtorIBAN:     debtorIBAN,
			debtorName:     "John Doe",
			creditorIBAN:   creditorIBAN,
			creditorName:   "Jane Smith",
			amount:         shared.Amount{},
			idempotencyKey: idempotencyKey,
			createdAt:      now,
			updatedAt:      now,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			payment, err := NewPayment(
				tt.id,
				tt.debtorIBAN,
				tt.debtorName,
				tt.creditorIBAN,
				tt.creditorName,
				tt.amount,
				tt.idempotencyKey,
				tt.createdAt,
				tt.updatedAt,
			)

			if tt.expectError {
				assert.Error(t, err, "expected error but got none")
			} else {
				assert.NoError(t, err, "unexpected error")

				// Verify payment properties
				assert.Equal(t, tt.id, payment.ID(), "payment ID should match")
				assert.True(t, payment.DebtorIBAN().Equals(tt.debtorIBAN), "debtor IBAN should match")
				assert.Equal(t, tt.debtorName, payment.DebtorName(), "debtor name should match")
				assert.True(t, payment.CreditorIBAN().Equals(tt.creditorIBAN), "creditor IBAN should match")
				assert.Equal(t, tt.creditorName, payment.CreditorName(), "creditor name should match")
				assert.True(t, payment.Amount().Equals(tt.amount), "amount should match")
				assert.True(t, payment.IdempotencyKey().Equals(tt.idempotencyKey), "idempotency key should match")
				assert.Equal(t, StatusPending, payment.Status(), "status should be pending")
				assert.True(t, payment.CreatedAt().Equal(tt.createdAt), "createdAt should match")
				assert.True(t, payment.UpdatedAt().Equal(tt.updatedAt), "updatedAt should match")
			}
		})
	}
}

func TestPayment_MarkAsProcessed(t *testing.T) {
	t.Parallel()
	// Create a valid payment
	payment := createValidPayment(t)
	updatedAt := time.Now().Add(time.Hour)

	// Test successful transition
	updatedPayment, err := payment.MarkAsProcessed(updatedAt)
	assert.NoError(t, err, "should successfully mark payment as processed")
	assert.Equal(t, StatusProcessed, updatedPayment.Status(), "status should be processed")
	assert.True(t, updatedPayment.UpdatedAt().Equal(updatedAt), "updatedAt should match")

	// Test invalid transition from processed state
	_, err = updatedPayment.MarkAsProcessed(updatedAt)
	assert.Equal(t, shared.ErrInvalidStatusTransition, err, "should return invalid status transition error")
}

func TestPayment_MarkAsFailed(t *testing.T) {
	t.Parallel()
	// Create a valid payment
	payment := createValidPayment(t)
	updatedAt := time.Now().Add(time.Hour)

	// Test successful transition
	updatedPayment, err := payment.MarkAsFailed(updatedAt)
	assert.NoError(t, err, "should successfully mark payment as failed")
	assert.Equal(t, StatusFailed, updatedPayment.Status(), "status should be failed")
	assert.True(t, updatedPayment.UpdatedAt().Equal(updatedAt), "updatedAt should match")

	// Test invalid transition from failed state
	_, err = updatedPayment.MarkAsFailed(updatedAt)
	assert.Equal(t, shared.ErrInvalidStatusTransition, err, "should return invalid status transition error")
}

func TestPayment_StatusTransitions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		initialStatus PaymentStatus
		targetStatus  PaymentStatus
		expectError   bool
	}{
		{
			name:          "pending to processed",
			initialStatus: StatusPending,
			targetStatus:  StatusProcessed,
			expectError:   false,
		},
		{
			name:          "pending to failed",
			initialStatus: StatusPending,
			targetStatus:  StatusFailed,
			expectError:   false,
		},
		{
			name:          "processed to failed (invalid)",
			initialStatus: StatusProcessed,
			targetStatus:  StatusFailed,
			expectError:   true,
		},
		{
			name:          "failed to processed (invalid)",
			initialStatus: StatusFailed,
			targetStatus:  StatusProcessed,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			payment := createValidPayment(t)
			updatedAt := time.Now().Add(time.Hour)

			// Set initial status
			if tt.initialStatus == StatusProcessed {
				payment, _ = payment.MarkAsProcessed(updatedAt)
			} else if tt.initialStatus == StatusFailed {
				payment, _ = payment.MarkAsFailed(updatedAt)
			}

			// Attempt transition
			var err error
			var updatedPayment Payment
			if tt.targetStatus == StatusProcessed {
				updatedPayment, err = payment.MarkAsProcessed(updatedAt)
			} else if tt.targetStatus == StatusFailed {
				updatedPayment, err = payment.MarkAsFailed(updatedAt)
			}

			if tt.expectError {
				assert.Equal(t, shared.ErrInvalidStatusTransition, err, "should return invalid status transition error")
			} else {
				assert.NoError(t, err, "should successfully transition status")
				assert.Equal(t, tt.targetStatus, updatedPayment.Status(), "status should match target status")
			}
		})
	}
}

// Helper function to create a valid payment for testing
func createValidPayment(t *testing.T) Payment {
	debtorIBAN, _ := shared.NewIBAN("GB82WEST12345698765432")
	creditorIBAN, _ := shared.NewIBAN("FR1420041010050500013M02606")
	amount, _ := shared.NewAmount(100.50)
	idempotencyKey, _ := shared.NewIdempotencyKey("abc123XYZ0")
	now := time.Now()

	payment, err := NewPayment(
		"payment-123",
		debtorIBAN,
		"John Doe",
		creditorIBAN,
		"Jane Smith",
		amount,
		idempotencyKey,
		now,
		now,
	)
	if err != nil {
		t.Fatalf("failed to create valid payment: %v", err)
	}
	return payment
}
