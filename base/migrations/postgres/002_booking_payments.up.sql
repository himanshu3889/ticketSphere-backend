
-- booking status enum
CREATE TYPE booking_status AS ENUM ('PENDING', 'CONFIRMED', 'FAILED', 'CANCELLED', 'EXPIRED');


-- bookings table
CREATE TABLE bookings (
    id BIGINT PRIMARY KEY, -- Snowflake ID from app
    event_id BIGINT NOT NULL REFERENCES events(id),
    seat_ids BIGINT[] NOT NULL DEFAULT '{}',  -- Remove it
    user_id BIGINT NOT NULL,
    status booking_status NOT NULL DEFAULT 'PENDING',
    total_amount NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL
);

-- booking idempotency key table
CREATE TABLE booking_idempotency_keys (
    idempotency_key VARCHAR(255) PRIMARY KEY,  -- uuid from app
    booking_id BIGINT NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    response_body JSONB, -- final response 
    request_hash VARCHAR(64), -- Optional: Ensures the payload hasn't changed
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP + INTERVAL '7 days'
);


-- payment status enum
CREATE TYPE payment_status AS ENUM ('PENDING', 'SUCCESS', 'FAILED', 'REFUNDED');


-- payments table
CREATE TABLE payments (
    id BIGINT PRIMARY KEY, -- Snowflake ID from app
    booking_id BIGINT NOT NULL REFERENCES bookings(id),
    amount NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    provider_ref VARCHAR(255),
    status payment_status NOT NULL DEFAULT 'PENDING',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    refunded_at TIMESTAMPTZ
);

-- payment idempotency key table
CREATE TABLE payment_idempotency_keys (
    idempotency_key VARCHAR(255) PRIMARY KEY,  -- uuid from app
    payment_id BIGINT NOT NULL UNIQUE REFERENCES payments(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    response_body JSONB, -- final response 
    request_hash VARCHAR(64), -- Optional: Ensures the payload hasn't changed
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP + INTERVAL '7 days'
);



-- Create index on bookings table
CREATE UNIQUE INDEX idx_bookings_idempotency_user_key ON booking_idempotency_keys(user_id, idempotency_key);
CREATE INDEX idx_bookings_event_status_id ON bookings(event_id, status, id DESC);
CREATE INDEX idx_bookings_user_status_id ON bookings(user_id, status, id DESC);
CREATE INDEX idx_bookings_status_id ON bookings(status, id DESC);
CREATE INDEX idx_bookings_expires_status_id ON bookings(expires_at, status, id DESC) WHERE status = 'PENDING';  -- Partial Indexing

-- Create index on payments booking
CREATE INDEX idx_payments_booking_id ON payments(booking_id, id DESC);
CREATE UNIQUE INDEX idx_payments_idempotency_user_key ON payment_idempotency_keys(user_id, idempotency_key);
