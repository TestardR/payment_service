-- Test data for payments table
-- This file contains sample data for testing purposes

INSERT OR IGNORE INTO payments (
    id,
    debtor_iban,
    debtor_name,
    creditor_iban,
    creditor_name,
    amount_cents,
    currency,
    idempotency_key,
    status,
    created_at,
    updated_at
) VALUES 
(
    'payment_001',
    'DE89370400440532013000',
    'John Doe',
    'FR1420041010050500013M02606',
    'Jane Smith',
    10050,  -- €100.50
    'EUR',
    'test123456',
    'PENDING',
    '2025-01-21 10:00:00',
    '2025-01-21 10:00:00'
),
(
    'payment_002',
    'GB82WEST12345698765432',
    'Alice Johnson',
    'ES9121000418450200051332',
    'Bob Wilson',
    25000,  -- €250.00
    'EUR',
    'test789012',
    'PROCESSED',
    '2025-01-21 09:30:00',
    '2025-01-21 11:15:00'
),
(
    'payment_003',
    'IT60X0542811101000000123456',
    'Charlie Brown',
    'NL91ABNA0417164300',
    'Diana Prince',
    500,    -- €5.00
    'EUR',
    'test345678',
    'FAILED',
    '2025-01-21 08:45:00',
    '2025-01-21 09:00:00'
);
