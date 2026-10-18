-- Booking Seats
DROP TABLE IF EXISTS booking_seats CASCADE;

-- Add the column back
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS seat_ids BIGINT[] NOT NULL DEFAULT '{}';
