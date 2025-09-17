package payment

import (
	"testing"
	"time"

	"paymentprocessor/internal/domain/shared"
)

func TestNewPayment(t *testing.T) {
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
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// Payment is now a value type, so we don't need to check for nil

				// Verify payment properties
				if payment.ID() != tt.id {
					t.Errorf("expected ID %q, got %q", tt.id, payment.ID())
				}
				if !payment.DebtorIBAN().Equals(tt.debtorIBAN) {
					t.Errorf("expected debtor IBAN %q, got %q", tt.debtorIBAN.String(), payment.DebtorIBAN().String())
				}
				if payment.DebtorName() != tt.debtorName {
					t.Errorf("expected debtor name %q, got %q", tt.debtorName, payment.DebtorName())
				}
				if !payment.CreditorIBAN().Equals(tt.creditorIBAN) {
					t.Errorf("expected creditor IBAN %q, got %q", tt.creditorIBAN.String(), payment.CreditorIBAN().String())
				}
				if payment.CreditorName() != tt.creditorName {
					t.Errorf("expected creditor name %q, got %q", tt.creditorName, payment.CreditorName())
				}
				if !payment.Amount().Equals(tt.amount) {
					t.Errorf("expected amount %f, got %f", tt.amount.Value(), payment.Amount().Value())
				}
				if !payment.IdempotencyKey().Equals(tt.idempotencyKey) {
					t.Errorf("expected idempotency key %q, got %q", tt.idempotencyKey.String(), payment.IdempotencyKey().String())
				}
				if payment.Status() != StatusPending {
					t.Errorf("expected status %q, got %q", StatusPending, payment.Status())
				}
				if !payment.CreatedAt().Equal(tt.createdAt) {
					t.Errorf("expected createdAt %v, got %v", tt.createdAt, payment.CreatedAt())
				}
				if !payment.UpdatedAt().Equal(tt.updatedAt) {
					t.Errorf("expected updatedAt %v, got %v", tt.updatedAt, payment.UpdatedAt())
				}
			}
		})
	}
}

func TestPayment_MarkAsProcessed(t *testing.T) {
	// Create a valid payment
	payment := createValidPayment(t)
	updatedAt := time.Now().Add(time.Hour)

	// Test successful transition
	updatedPayment, err := payment.MarkAsProcessed(updatedAt)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedPayment.Status() != StatusProcessed {
		t.Errorf("expected status %q, got %q", StatusProcessed, updatedPayment.Status())
	}
	if !updatedPayment.UpdatedAt().Equal(updatedAt) {
		t.Errorf("expected updatedAt %v, got %v", updatedAt, updatedPayment.UpdatedAt())
	}

	// Test invalid transition from processed state
	_, err = updatedPayment.MarkAsProcessed(updatedAt)
	if err != shared.ErrInvalidStatusTransition {
		t.Errorf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

func TestPayment_MarkAsFailed(t *testing.T) {
	// Create a valid payment
	payment := createValidPayment(t)
	updatedAt := time.Now().Add(time.Hour)

	// Test successful transition
	updatedPayment, err := payment.MarkAsFailed(updatedAt)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedPayment.Status() != StatusFailed {
		t.Errorf("expected status %q, got %q", StatusFailed, updatedPayment.Status())
	}
	if !updatedPayment.UpdatedAt().Equal(updatedAt) {
		t.Errorf("expected updatedAt %v, got %v", updatedAt, updatedPayment.UpdatedAt())
	}

	// Test invalid transition from failed state
	_, err = updatedPayment.MarkAsFailed(updatedAt)
	if err != shared.ErrInvalidStatusTransition {
		t.Errorf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

func TestPayment_StatusTransitions(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  PaymentStatus
		targetStatus   PaymentStatus
		expectError    bool
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
				if err != shared.ErrInvalidStatusTransition {
					t.Errorf("expected ErrInvalidStatusTransition, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if updatedPayment.Status() != tt.targetStatus {
					t.Errorf("expected status %q, got %q", tt.targetStatus, updatedPayment.Status())
				}
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
