-- Drop the new unique index on gateway_transaction_id
DROP INDEX IF EXISTS idx_payments_gateway_txn;

-- Rename the column back to its original name
ALTER TABLE payments RENAME COLUMN gateway_transaction_id TO provider_ref;

-- Drop the provider column
ALTER TABLE payments DROP COLUMN provider;

-- Drop the payment_provider ENUM type
DROP TYPE IF EXISTS payment_provider;

-- Drop the index on provider_ref
DROP INDEX IF EXISTS idx_payments_provider_ref;

-- Drop the partial unique index preventing double payments
DROP INDEX IF EXISTS idx_single_success_payment;
