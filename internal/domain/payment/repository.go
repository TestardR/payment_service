package payment

import (
	"context"

	"paymentprocessor/internal/domain/shared"
)

//go:generate mockgen -source=repository.go -destination=../../mocks/payment_repository_mock.go -package=mocks

type Repository interface {
	Save(ctx context.Context, payment *Payment) error
	FindByID(ctx context.Context, id string) (*Payment, error)
	FindByIdempotencyKey(ctx context.Context, key shared.IdempotencyKey) (*Payment, error)
	UpdateStatus(ctx context.Context, id string, status PaymentStatus) error
}
