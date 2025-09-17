package payment

import (
	"context"

	"paymentprocessor/internal/domain/shared"
)

type Repository interface {
	Save(ctx context.Context, payment *Payment) error
	FindByID(ctx context.Context, id string) (*Payment, error)
	FindByIdempotencyKey(ctx context.Context, key shared.IdempotencyKey) (*Payment, error)
	UpdateStatus(ctx context.Context, id string, status PaymentStatus) error
}
