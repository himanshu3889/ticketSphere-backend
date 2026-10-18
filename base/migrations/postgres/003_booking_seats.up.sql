-- Drop the array column on the bookings table
ALTER TABLE bookings DROP COLUMN IF EXISTS seat_ids;

CREATE TABLE booking_seats (
    booking_id BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    seat_id BIGINT NOT NULL REFERENCES seats(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (booking_id, seat_id)
);

CREATE INDEX idx_booking_seats_seat_id ON booking_seats(seat_id);

