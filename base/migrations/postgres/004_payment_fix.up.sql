-- a booking can only have one successful payment
CREATE UNIQUE INDEX idx_single_success_payment ON payments(booking_id) WHERE status = 'SUCCESS';

-- unique provider reference
CREATE UNIQUE INDEX idx_payments_provider_ref ON payments(provider_ref);


--  Create the ENUM for gateway names
CREATE TYPE payment_provider AS ENUM ('INTERNAL', 'STRIPE');

-- Add the provider column to payments table
ALTER TABLE payments 
ADD COLUMN provider payment_provider NOT NULL DEFAULT 'INTERNAL';

-- Rename the old column provider_ref
ALTER TABLE payments RENAME COLUMN provider_ref TO gateway_transaction_id;

-- Add a unique index to gateway transaction id
CREATE UNIQUE INDEX idx_payments_gateway_txn ON payments(gateway_transaction_id);