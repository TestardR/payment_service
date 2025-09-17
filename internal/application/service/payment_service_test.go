package service

import (
	"context"
	"testing"
	"time"

	"paymentprocessor/internal/domain/payment"
	"paymentprocessor/internal/domain/shared"
	"paymentprocessor/internal/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestPaymentService_EnsureIdempotency(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create test data
	debtorIBAN, _ := shared.NewIBAN("GB82WEST12345698765432")
	creditorIBAN, _ := shared.NewIBAN("FR1420041010050500013M02606")
	amount, _ := shared.NewAmount(100.50)
	existingKey, _ := shared.NewIdempotencyKey("abc123XYZ0")
	newKey, _ := shared.NewIdempotencyKey("xyz789ABC1")

	now := time.Now()
	existingPayment, _ := payment.NewPayment(
		"payment-123",
		debtorIBAN,
		"John Doe",
		creditorIBAN,
		"Jane Smith",
		amount,
		existingKey,
		now,
		now,
	)

	tests := []struct {
		name          string
		key           shared.IdempotencyKey
		setupMock     func(mockRepo *mocks.MockRepository)
		expectPayment bool
		expectError   error
	}{
		{
			name: "existing payment found",
			key:  existingKey,
			setupMock: func(mockRepo *mocks.MockRepository) {
				mockRepo.EXPECT().
					FindByIdempotencyKey(ctx, existingKey).
					Return(existingPayment, nil)
			},
			expectPayment: true,
			expectError:   shared.ErrDuplicatePayment,
		},
		{
			name: "no existing payment",
			key:  newKey,
			setupMock: func(mockRepo *mocks.MockRepository) {
				mockRepo.EXPECT().
					FindByIdempotencyKey(ctx, newKey).
					Return(payment.Payment{}, shared.ErrPaymentNotFound)
			},
			expectPayment: false,
			expectError:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			service := NewPaymentService(mockRepo)

			tt.setupMock(mockRepo)

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
	t.Parallel()
	ctx := context.Background()

	// Create test payment data
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

	tests := []struct {
		name        string
		paymentID   string
		newStatus   payment.PaymentStatus
		setupMock   func(mockRepo *mocks.MockRepository)
		expectError bool
	}{
		{
			name:      "valid transition to processed",
			paymentID: "payment-123",
			newStatus: payment.StatusProcessed,
			setupMock: func(mockRepo *mocks.MockRepository) {
				mockRepo.EXPECT().
					FindByID(ctx, "payment-123").
					Return(testPayment, nil)
				mockRepo.EXPECT().
					Save(ctx, gomock.Cond(func(p interface{}) bool {
						if pmt, ok := p.(payment.Payment); ok {
							return pmt.ID() == "payment-123" && pmt.Status() == payment.StatusProcessed
						}
						return false
					})).
					Return(nil)
			},
			expectError: false,
		},
		{
			name:      "valid transition to failed",
			paymentID: "payment-123",
			newStatus: payment.StatusFailed,
			setupMock: func(mockRepo *mocks.MockRepository) {
				mockRepo.EXPECT().
					FindByID(ctx, "payment-123").
					Return(testPayment, nil)
				mockRepo.EXPECT().
					Save(ctx, gomock.Cond(func(p interface{}) bool {
						if pmt, ok := p.(payment.Payment); ok {
							return pmt.ID() == "payment-123" && pmt.Status() == payment.StatusFailed
						}
						return false
					})).
					Return(nil)
			},
			expectError: false,
		},
		{
			name:      "payment not found",
			paymentID: "nonexistent",
			newStatus: payment.StatusProcessed,
			setupMock: func(mockRepo *mocks.MockRepository) {
				mockRepo.EXPECT().
					FindByID(ctx, "nonexistent").
					Return(payment.Payment{}, shared.ErrPaymentNotFound)
			},
			expectError: true,
		},
		{
			name:      "invalid status",
			paymentID: "payment-123",
			newStatus: payment.PaymentStatus("INVALID"),
			setupMock: func(mockRepo *mocks.MockRepository) {
				mockRepo.EXPECT().
					FindByID(ctx, "payment-123").
					Return(testPayment, nil)
				// No Save call expected because the service should return error before calling Save
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			service := NewPaymentService(mockRepo)

			tt.setupMock(mockRepo)

			err := service.ProcessStatusUpdate(ctx, tt.paymentID, tt.newStatus, time.Now())

			if tt.expectError {
				assert.Error(t, err, "expected error but got none")
			} else {
				assert.NoError(t, err, "unexpected error")
			}
		})
	}
}

func TestNewPaymentService(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	service := NewPaymentService(mockRepo)

	// Test that service is created as value type
	assert.NotNil(t, service.repository, "expected repository to be set")
}
