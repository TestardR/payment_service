package service

import (
	"context"
	"testing"
	"time"

	"paymentprocessor/internal/domain/payment"
	"paymentprocessor/internal/domain/shared"
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
		name           string
		key            shared.IdempotencyKey
		expectPayment  bool
		expectError    error
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
				if err != tt.expectError {
					t.Errorf("expected error %v, got %v", tt.expectError, err)
				}
				if tt.expectPayment {
					if foundPayment.ID() != existingPayment.ID() {
						t.Errorf("expected to find existing payment %q, got %q", existingPayment.ID(), foundPayment.ID())
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// For new payments, we expect an empty payment
				if foundPayment.ID() != "" {
					t.Errorf("expected empty payment for new key, got payment with ID %q", foundPayment.ID())
				}
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
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				
				// Verify the payment was updated in repository
				updatedPayment, _ := repo.FindByID(ctx, tt.paymentID)
				if updatedPayment.Status() != tt.newStatus {
					t.Errorf("expected status %q, got %q", tt.newStatus, updatedPayment.Status())
				}
			}
		})
	}
}

func TestNewPaymentService(t *testing.T) {
	repo := newMockPaymentRepository()
	service := NewPaymentService(repo)

	// Test that service is created as value type
	if service.repository == nil {
		t.Error("expected repository to be set")
	}
}
