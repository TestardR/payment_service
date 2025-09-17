package service

import (
	"context"
	"errors"
	"time"

	"paymentprocessor/internal/domain/payment"
	"paymentprocessor/internal/domain/shared"
)

type PaymentService struct {
	repository payment.Repository
}

func NewPaymentService(repository payment.Repository) PaymentService {
	return PaymentService{
		repository: repository,
	}
}

func (s PaymentService) EnsureIdempotency(ctx context.Context, key shared.IdempotencyKey) (payment.Payment, error) {
	existingPayment, err := s.repository.FindByIdempotencyKey(ctx, key)
	if err != nil && !errors.Is(err, shared.ErrPaymentNotFound) {
		return payment.Payment{}, err
	}

	if err == nil {
		return existingPayment, shared.ErrDuplicatePayment
	}

	return payment.Payment{}, nil
}

func (s PaymentService) ProcessStatusUpdate(ctx context.Context, paymentID string, newStatus payment.PaymentStatus, updatedAt time.Time) error {
	existingPayment, err := s.repository.FindByID(ctx, paymentID)
	if err != nil {
		return err
	}

	var updatedPayment payment.Payment
	switch newStatus {
	case payment.StatusProcessed:
		updatedPayment, err = existingPayment.MarkAsProcessed(updatedAt)
		if err != nil {
			return err
		}
	case payment.StatusFailed:
		updatedPayment, err = existingPayment.MarkAsFailed(updatedAt)
		if err != nil {
			return err
		}
	default:
		return shared.ErrInvalidPaymentStatus
	}

	return s.repository.Save(ctx, updatedPayment)
}
