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

// PaymentRepository implements the payment.Repository interface using SQLite
type PaymentRepository struct {
	db *Database
}

// NewPaymentRepository creates a new SQLite payment repository
func NewPaymentRepository(db *Database) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Save persists a payment to the database
func (r *PaymentRepository) Save(ctx context.Context, p *payment.Payment) error {
	if p == nil {
		return fmt.Errorf("payment cannot be nil")
	}

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
		"EUR", // Default currency
		p.IdempotencyKey().Value(),
		string(p.Status()),
		p.CreatedAt(),
		p.UpdatedAt(),
	)

	if err != nil {
		// Check for unique constraint violation on idempotency key
		if isUniqueConstraintError(err) {
			return shared.ErrDuplicateIdempotencyKey
		}
		return fmt.Errorf("failed to save payment: %w", err)
	}

	return nil
}

// FindByID retrieves a payment by its ID
func (r *PaymentRepository) FindByID(ctx context.Context, id string) (*payment.Payment, error) {
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
			return nil, nil // Payment not found
		}
		return nil, fmt.Errorf("failed to find payment by ID: %w", err)
	}

	return p, nil
}

// FindByIdempotencyKey retrieves a payment by its idempotency key
func (r *PaymentRepository) FindByIdempotencyKey(ctx context.Context, key shared.IdempotencyKey) (*payment.Payment, error) {
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
			return nil, nil // Payment not found
		}
		return nil, fmt.Errorf("failed to find payment by idempotency key: %w", err)
	}

	return p, nil
}

// UpdateStatus updates the status of a payment
func (r *PaymentRepository) UpdateStatus(ctx context.Context, id string, status payment.PaymentStatus) error {
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

// scanPayment scans a database row into a Payment domain object
func (r *PaymentRepository) scanPayment(row *sql.Row) (*payment.Payment, error) {
	var (
		id               string
		debtorIBAN       string
		debtorName       string
		creditorIBAN     string
		creditorName     string
		amountCents      int64
		idempotencyKey   string
		status           string
		createdAt        time.Time
		updatedAt        time.Time
	)

	err := row.Scan(
		&id, &debtorIBAN, &debtorName, &creditorIBAN, &creditorName,
		&amountCents, &idempotencyKey, &status, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert database values to domain objects
	debtorIBANObj, err := shared.NewIBAN(debtorIBAN)
	if err != nil {
		return nil, fmt.Errorf("invalid debtor IBAN in database: %w", err)
	}

	creditorIBANObj, err := shared.NewIBAN(creditorIBAN)
	if err != nil {
		return nil, fmt.Errorf("invalid creditor IBAN in database: %w", err)
	}

	amount, err := shared.NewAmountFromCents(amountCents)
	if err != nil {
		return nil, fmt.Errorf("invalid amount in database: %w", err)
	}

	idempotencyKeyObj, err := shared.NewIdempotencyKey(idempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("invalid idempotency key in database: %w", err)
	}

	// Create payment domain object
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
		return nil, fmt.Errorf("failed to create payment domain object: %w", err)
	}

	// Set the correct status (NewPayment always creates with PENDING status)
	switch payment.PaymentStatus(status) {
	case payment.StatusProcessed:
		if err := p.MarkAsProcessed(updatedAt); err != nil {
			return nil, fmt.Errorf("failed to set payment status to processed: %w", err)
		}
	case payment.StatusFailed:
		if err := p.MarkAsFailed(updatedAt); err != nil {
			return nil, fmt.Errorf("failed to set payment status to failed: %w", err)
		}
	case payment.StatusPending:
		// Already set by NewPayment
	default:
		return nil, fmt.Errorf("unknown payment status: %s", status)
	}

	return p, nil
}

// isUniqueConstraintError checks if the error is a unique constraint violation
func isUniqueConstraintError(err error) bool {
	// SQLite unique constraint error message contains "UNIQUE constraint failed"
	return err != nil && (
		fmt.Sprintf("%v", err) == "UNIQUE constraint failed: payments.idempotency_key" ||
		fmt.Sprintf("%v", err) == "UNIQUE constraint failed: payments.id")
}
