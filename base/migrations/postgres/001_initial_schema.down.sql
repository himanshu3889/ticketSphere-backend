-- Payment Idempotency Keys
DROP TABLE IF EXISTS payment_idempotency_keys CASCADE;

-- Payments
DROP TABLE IF EXISTS payments CASCADE;

-- Booking Idempotency Keys
DROP TABLE IF EXISTS booking_idempotency_keys CASCADE;

-- Bookings
DROP TABLE IF EXISTS bookings CASCADE;

-- Seats
DROP TABLE IF EXISTS seats CASCADE;

-- Sections
DROP TABLE IF EXISTS sections CASCADE;

-- Events
DROP TABLE IF EXISTS events CASCADE;

-- User Sessions
DROP TABLE IF EXISTS user_sessions CASCADE;

-- Users
DROP TABLE IF EXISTS users CASCADE;

-- Venues
DROP TABLE IF EXISTS venues CASCADE;

-- Enums (must be last!)
DROP TYPE IF EXISTS payment_status CASCADE;
DROP TYPE IF EXISTS booking_status CASCADE;
DROP TYPE IF EXISTS seat_status CASCADE;
DROP TYPE IF EXISTS event_status CASCADE;
