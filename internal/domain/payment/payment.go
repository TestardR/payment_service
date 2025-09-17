package payment

import (
	"time"

	"paymentprocessor/internal/domain/shared"
)

type Payment struct {
	id             string
	debtorIBAN     shared.IBAN
	debtorName     string
	creditorIBAN   shared.IBAN
	creditorName   string
	amount         shared.Amount
	idempotencyKey shared.IdempotencyKey
	status         PaymentStatus
	createdAt      time.Time
	updatedAt      time.Time
}

func NewPayment(
	id string,
	debtorIBAN shared.IBAN,
	debtorName string,
	creditorIBAN shared.IBAN,
	creditorName string,
	amount shared.Amount,
	idempotencyKey shared.IdempotencyKey,
	createdAt time.Time,
	updatedAt time.Time,
) (*Payment, error) {
	if err := validatePaymentData(debtorName, creditorName, amount); err != nil {
		return nil, err
	}

	return &Payment{
		id:             id,
		debtorIBAN:     debtorIBAN,
		debtorName:     debtorName,
		creditorIBAN:   creditorIBAN,
		creditorName:   creditorName,
		amount:         amount,
		idempotencyKey: idempotencyKey,
		status:         StatusPending,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}, nil
}

func (p *Payment) MarkAsProcessed(updatedAt time.Time) error {
	if !p.canTransitionTo(StatusProcessed) {
		return shared.ErrInvalidStatusTransition
	}

	p.status = StatusProcessed
	p.updatedAt = updatedAt
	return nil
}

func (p *Payment) MarkAsFailed(updatedAt time.Time) error {
	if !p.canTransitionTo(StatusFailed) {
		return shared.ErrInvalidStatusTransition
	}

	p.status = StatusFailed
	p.updatedAt = updatedAt
	return nil
}

func (p *Payment) canTransitionTo(newStatus PaymentStatus) bool {
	switch p.status {
	case StatusPending:
		return newStatus == StatusProcessed || newStatus == StatusFailed
	case StatusProcessed, StatusFailed:
		return false
	default:
		return false
	}
}

func (p *Payment) ID() string                            { return p.id }
func (p *Payment) DebtorIBAN() shared.IBAN               { return p.debtorIBAN }
func (p *Payment) DebtorName() string                    { return p.debtorName }
func (p *Payment) CreditorIBAN() shared.IBAN             { return p.creditorIBAN }
func (p *Payment) CreditorName() string                  { return p.creditorName }
func (p *Payment) Amount() shared.Amount                 { return p.amount }
func (p *Payment) IdempotencyKey() shared.IdempotencyKey { return p.idempotencyKey }
func (p *Payment) Status() PaymentStatus                 { return p.status }
func (p *Payment) CreatedAt() time.Time                  { return p.createdAt }
func (p *Payment) UpdatedAt() time.Time                  { return p.updatedAt }

func validatePaymentData(debtorName, creditorName string, amount shared.Amount) error {
	if len(debtorName) < 3 {
		return shared.ErrInvalidAmount
	}

	if len(creditorName) < 3 {
		return shared.ErrInvalidAmount
	}

	if amount.IsZero() {
		return shared.ErrInvalidAmount
	}

	return nil
}
