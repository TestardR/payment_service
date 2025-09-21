package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"paymentprocessor/internal/domain/payment"
	"paymentprocessor/internal/domain/shared"
)

type PaymentRepository struct {
	db Database
}

func NewPaymentRepository(db Database) PaymentRepository {
	return PaymentRepository{db: db}
}

func (r PaymentRepository) Save(ctx context.Context, p payment.Payment) error {

	query := `
		INSERT INTO payments (
			id, debtor_iban, debtor_name, creditor_iban, creditor_name,
			amount_cents, currency, idempotency_key, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		p.ID(),
		p.DebtorIBAN().Value(),
		p.DebtorName(),
		p.CreditorIBAN().Value(),
		p.CreditorName(),
		p.Amount().Cents(),
		"EUR",
		p.IdempotencyKey().Value(),
		string(p.Status()),
		p.CreatedAt(),
		p.UpdatedAt(),
	)

	if err != nil {
		if isUniqueConstraintError(err) {
			return shared.ErrDuplicateIdempotencyKey
		}
		return fmt.Errorf("failed to save payment: %w", err)
	}

	return nil
}

func (r PaymentRepository) FindByID(ctx context.Context, id string) (payment.Payment, error) {
	query := `
		SELECT id, debtor_iban, debtor_name, creditor_iban, creditor_name,
			   amount_cents, idempotency_key, status, created_at, updated_at
		FROM payments
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)

	p, err := r.scanPayment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return payment.Payment{}, shared.ErrPaymentNotFound
		}
		return payment.Payment{}, fmt.Errorf("failed to find payment by ID: %w", err)
	}

	return p, nil
}

func (r PaymentRepository) FindByIdempotencyKey(ctx context.Context, key shared.IdempotencyKey) (payment.Payment, error) {
	query := `
		SELECT id, debtor_iban, debtor_name, creditor_iban, creditor_name,
			   amount_cents, idempotency_key, status, created_at, updated_at
		FROM payments
		WHERE idempotency_key = ?
	`

	row := r.db.QueryRowContext(ctx, query, key.Value())

	p, err := r.scanPayment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return payment.Payment{}, shared.ErrPaymentNotFound
		}
		return payment.Payment{}, fmt.Errorf("failed to find payment by idempotency key: %w", err)
	}

	return p, nil
}

func (r PaymentRepository) UpdateStatus(ctx context.Context, id string, status payment.PaymentStatus) error {
	query := `
		UPDATE payments 
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, string(status), id)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("payment with ID %s not found", id)
	}

	return nil
}

func (r PaymentRepository) scanPayment(row *sql.Row) (payment.Payment, error) {
	var (
		id             string
		debtorIBAN     string
		debtorName     string
		creditorIBAN   string
		creditorName   string
		amountCents    int64
		idempotencyKey string
		status         string
		createdAt      time.Time
		updatedAt      time.Time
	)

	err := row.Scan(
		&id, &debtorIBAN, &debtorName, &creditorIBAN, &creditorName,
		&amountCents, &idempotencyKey, &status, &createdAt, &updatedAt,
	)
	if err != nil {
		return payment.Payment{}, err
	}

	debtorIBANObj, err := shared.NewIBAN(debtorIBAN)
	if err != nil {
		return payment.Payment{}, fmt.Errorf("invalid debtor IBAN in database: %w", err)
	}

	creditorIBANObj, err := shared.NewIBAN(creditorIBAN)
	if err != nil {
		return payment.Payment{}, fmt.Errorf("invalid creditor IBAN in database: %w", err)
	}

	amount, err := shared.NewAmountFromCents(amountCents)
	if err != nil {
		return payment.Payment{}, fmt.Errorf("invalid amount in database: %w", err)
	}

	idempotencyKeyObj, err := shared.NewIdempotencyKey(idempotencyKey)
	if err != nil {
		return payment.Payment{}, fmt.Errorf("invalid idempotency key in database: %w", err)
	}

	p, err := payment.NewPayment(
		id,
		debtorIBANObj,
		debtorName,
		creditorIBANObj,
		creditorName,
		amount,
		idempotencyKeyObj,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return payment.Payment{}, fmt.Errorf("failed to create payment domain object: %w", err)
	}

	switch payment.PaymentStatus(status) {
	case payment.StatusProcessed:
		if err := p.MarkAsProcessed(updatedAt); err != nil {
			return payment.Payment{}, fmt.Errorf("failed to set payment status to processed: %w", err)
		}
	case payment.StatusFailed:
		if err := p.MarkAsFailed(updatedAt); err != nil {
			return payment.Payment{}, fmt.Errorf("failed to set payment status to failed: %w", err)
		}
	case payment.StatusPending:
	default:
		return payment.Payment{}, fmt.Errorf("unknown payment status: %s", status)
	}

	return p, nil
}

func isUniqueConstraintError(err error) bool {
	return err != nil && (fmt.Sprintf("%v", err) == "UNIQUE constraint failed: payments.idempotency_key" ||
		fmt.Sprintf("%v", err) == "UNIQUE constraint failed: payments.id")
}
