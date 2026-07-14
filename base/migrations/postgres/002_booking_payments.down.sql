-- Payment Idempotency Keys
DROP TABLE IF EXISTS payment_idempotency_keys CASCADE;

-- Payments
DROP TABLE IF EXISTS payments CASCADE;

-- Booking Idempotency Keys
DROP TABLE IF EXISTS booking_idempotency_keys CASCADE;

-- Bookings
DROP TABLE IF EXISTS bookings CASCADE;

-- Enums (must be last!)
DROP TYPE IF EXISTS payment_status CASCADE;
DROP TYPE IF EXISTS booking_status CASCADE;