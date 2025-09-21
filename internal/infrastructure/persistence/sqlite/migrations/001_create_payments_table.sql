-- Migration: 001_create_payments_table.sql
-- Description: Create payments table with proper indexes and constraints
-- Created: 2025-01-21

-- Create payments table
CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY NOT NULL,
    debtor_iban TEXT NOT NULL,
    debtor_name TEXT NOT NULL CHECK(length(debtor_name) >= 3 AND length(debtor_name) <= 30),
    creditor_iban TEXT NOT NULL,
    creditor_name TEXT NOT NULL CHECK(length(creditor_name) >= 3 AND length(creditor_name) <= 30),
    amount_cents INTEGER NOT NULL CHECK(amount_cents > 0),
    currency TEXT NOT NULL DEFAULT 'EUR',
    idempotency_key TEXT NOT NULL UNIQUE CHECK(length(idempotency_key) = 10),
    status TEXT NOT NULL CHECK(status IN ('PENDING', 'PROCESSED', 'FAILED')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient querying
CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_idempotency_key ON payments(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at);
CREATE INDEX IF NOT EXISTS idx_payments_updated_at ON payments(updated_at);
CREATE INDEX IF NOT EXISTS idx_payments_debtor_iban ON payments(debtor_iban);
CREATE INDEX IF NOT EXISTS idx_payments_creditor_iban ON payments(creditor_iban);

-- Create trigger to automatically update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_payments_updated_at
    AFTER UPDATE ON payments
    FOR EACH ROW
BEGIN
    UPDATE payments SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
