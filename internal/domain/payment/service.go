package payment

import (
	"context"

	"paymentprocessor/internal/domain/shared"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) EnsureIdempotency(ctx context.Context, key shared.IdempotencyKey) (*Payment, error) {
	existingPayment, err := s.repository.FindByIdempotencyKey(ctx, key)
	if err != nil && err != shared.ErrPaymentNotFound {
		return nil, err
	}

	if existingPayment != nil {
		return existingPayment, shared.ErrDuplicatePayment
	}

	return nil, nil
}

func (s *Service) ProcessStatusUpdate(ctx context.Context, paymentID string, newStatus PaymentStatus) error {
	payment, err := s.repository.FindByID(ctx, paymentID)
	if err != nil {
		return err
	}

	switch newStatus {
	case StatusProcessed:
		if err := payment.MarkAsProcessed(); err != nil {
			return err
		}
	case StatusFailed:
		if err := payment.MarkAsFailed(); err != nil {
			return err
		}
	default:
		return shared.ErrInvalidPaymentStatus
	}

	return s.repository.UpdateStatus(ctx, paymentID, newStatus)
}
