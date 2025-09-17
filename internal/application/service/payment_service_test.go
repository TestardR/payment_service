package service

import (
	"context"
	"testing"
	"time"

	"paymentprocessor/internal/domain/payment"
	"paymentprocessor/internal/domain/shared"

	"github.com/stretchr/testify/assert"
)

// Mock repository for testing
type mockPaymentRepository struct {
	payments map[string]payment.Payment
	keys     map[string]payment.Payment
}

func newMockPaymentRepository() *mockPaymentRepository {
	return &mockPaymentRepository{
		payments: make(map[string]payment.Payment),
		keys:     make(map[string]payment.Payment),
	}
}

func (m *mockPaymentRepository) Save(ctx context.Context, p payment.Payment) error {
	m.payments[p.ID()] = p
	m.keys[p.IdempotencyKey().String()] = p
	return nil
}

func (m *mockPaymentRepository) FindByID(ctx context.Context, id string) (payment.Payment, error) {
	p, exists := m.payments[id]
	if !exists {
		return payment.Payment{}, shared.ErrPaymentNotFound
	}
	return p, nil
}

func (m *mockPaymentRepository) FindByIdempotencyKey(ctx context.Context, key shared.IdempotencyKey) (payment.Payment, error) {
	p, exists := m.keys[key.String()]
	if !exists {
		return payment.Payment{}, shared.ErrPaymentNotFound
	}
	return p, nil
}

func (m *mockPaymentRepository) UpdateStatus(ctx context.Context, id string, status payment.PaymentStatus) error {
	p, exists := m.payments[id]
	if !exists {
		return shared.ErrPaymentNotFound
	}
	// This is a simplified update - in real implementation, this would be more complex
	m.payments[id] = p
	return nil
}

func TestPaymentService_EnsureIdempotency(t *testing.T) {
	repo := newMockPaymentRepository()
	service := NewPaymentService(repo)
	ctx := context.Background()

	// Create a test payment and save it
	debtorIBAN, _ := shared.NewIBAN("GB82WEST12345698765432")
	creditorIBAN, _ := shared.NewIBAN("FR1420041010050500013M02606")
	amount, _ := shared.NewAmount(100.50)
	idempotencyKey, _ := shared.NewIdempotencyKey("abc123XYZ0")

	now := time.Now()
	existingPayment, _ := payment.NewPayment(
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
	repo.Save(ctx, existingPayment)

	tests := []struct {
		name          string
		key           shared.IdempotencyKey
		expectPayment bool
		expectError   error
	}{
		{
			name:          "existing payment found",
			key:           idempotencyKey,
			expectPayment: true,
			expectError:   shared.ErrDuplicatePayment,
		},
		{
			name:          "no existing payment",
			key:           func() shared.IdempotencyKey { k, _ := shared.NewIdempotencyKey("xyz789ABC1"); return k }(),
			expectPayment: false,
			expectError:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundPayment, err := service.EnsureIdempotency(ctx, tt.key)

			if tt.expectError != nil {
				assert.Equal(t, tt.expectError, err, "expected specific error")
				if tt.expectPayment {
					assert.Equal(t, existingPayment.ID(), foundPayment.ID(), "expected to find existing payment")
				}
			} else {
				assert.NoError(t, err, "should not return error for new payment")
				// For new payments, we expect an empty payment
				assert.Empty(t, foundPayment.ID(), "expected empty payment for new key")
			}
		})
	}
}

func TestPaymentService_ProcessStatusUpdate(t *testing.T) {
	repo := newMockPaymentRepository()
	service := NewPaymentService(repo)
	ctx := context.Background()

	// Create and save a test payment
	debtorIBAN, _ := shared.NewIBAN("GB82WEST12345698765432")
	creditorIBAN, _ := shared.NewIBAN("FR1420041010050500013M02606")
	amount, _ := shared.NewAmount(100.50)
	idempotencyKey, _ := shared.NewIdempotencyKey("abc123XYZ0")

	now := time.Now()
	testPayment, _ := payment.NewPayment(
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
	repo.Save(ctx, testPayment)

	tests := []struct {
		name        string
		paymentID   string
		newStatus   payment.PaymentStatus
		expectError bool
	}{
		{
			name:        "valid transition to processed",
			paymentID:   "payment-123",
			newStatus:   payment.StatusProcessed,
			expectError: false,
		},
		{
			name:        "valid transition to failed",
			paymentID:   "payment-123",
			newStatus:   payment.StatusFailed,
			expectError: false,
		},
		{
			name:        "payment not found",
			paymentID:   "nonexistent",
			newStatus:   payment.StatusProcessed,
			expectError: true,
		},
		{
			name:        "invalid status",
			paymentID:   "payment-123",
			newStatus:   payment.PaymentStatus("INVALID"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset payment to pending state for each test
			if tt.paymentID == "payment-123" {
				repo.Save(ctx, testPayment) // Reset to original state
			}

			err := service.ProcessStatusUpdate(ctx, tt.paymentID, tt.newStatus, time.Now())

			if tt.expectError {
				assert.Error(t, err, "expected error but got none")
			} else {
				assert.NoError(t, err, "unexpected error")

				// Verify the payment was updated in repository
				updatedPayment, _ := repo.FindByID(ctx, tt.paymentID)
				assert.Equal(t, tt.newStatus, updatedPayment.Status(), "payment status should be updated")
			}
		})
	}
}

func TestNewPaymentService(t *testing.T) {
	repo := newMockPaymentRepository()
	service := NewPaymentService(repo)

	// Test that service is created as value type
	assert.NotNil(t, service.repository, "expected repository to be set")
}
