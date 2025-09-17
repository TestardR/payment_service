package shared

import "errors"

var (
	ErrInvalidIBAN             = errors.New("invalid IBAN format")
	ErrInvalidAmount           = errors.New("invalid amount")
	ErrInvalidIdempotencyKey   = errors.New("invalid idempotency key")
	ErrInvalidPaymentStatus    = errors.New("invalid payment status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrPaymentNotFound         = errors.New("payment not found")
	ErrDuplicatePayment        = errors.New("duplicate payment")
)
