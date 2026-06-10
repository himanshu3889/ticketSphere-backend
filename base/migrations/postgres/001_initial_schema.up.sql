-- Users table
CREATE TABLE users (
    id BIGINT PRIMARY KEY,  -- Snowflake ID from app
    username VARCHAR(32) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    image_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);


--  venues table
CREATE TABLE venues (
    id BIGINT PRIMARY KEY, -- Snowflake ID from app
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    capacity INT CHECK (capacity >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- event status enum
CREATE TYPE event_status AS ENUM ('ACTIVE', 'ON_SALE', 'SOLD_OUT', 'CANCELLED', 'COMPLETED');

-- events table
CREATE TABLE events (
    id BIGINT PRIMARY KEY, -- Snowflake ID from app
    name VARCHAR(255) NOT NULL,
    venue_id BIGINT NOT NULL REFERENCES venues(id),
    start_time TIMESTAMPTZ,
    on_sale_time TIMESTAMPTZ,
    status event_status NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- sections table
CREATE TABLE sections (
    id BIGINT PRIMARY KEY,  -- Snowflake ID from app
    event_id BIGINT NOT NULL REFERENCES events(id),
    name VARCHAR(100) NOT NULL,
    capacity INT NOT NULL DEFAULT 0 CHECK (capacity >= 0),
    seats_remaining INT NOT NULL DEFAULT 0 CHECK (seats_remaining >= 0),
    base_price NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seat status enum
CREATE TYPE seat_status AS ENUM ('AVAILABLE', 'HELD', 'BOOKED', 'BLOCKED');

-- seats table
CREATE TABLE seats ( 
    id BIGINT PRIMARY KEY,  -- Snowflake ID from app
    event_id BIGINT NOT NULL REFERENCES events(id),
    section_id BIGINT NOT NULL REFERENCES sections(id),
    row VARCHAR(10) NOT NULL,
    number VARCHAR(10) NOT NULL,
    status seat_status NOT NULL DEFAULT 'AVAILABLE',
    version INT NOT NULL DEFAULT 0,
    price NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- booking status enum
CREATE TYPE booking_status AS ENUM ('PENDING', 'CONFIRMED', 'FAILED', 'CANCELLED', 'EXPIRED');


-- bookings table
CREATE TABLE bookings (
    id BIGINT PRIMARY KEY, -- Snowflake ID from app
    event_id BIGINT NOT NULL REFERENCES events(id),
    seat_ids BIGINT[] NOT NULL DEFAULT '{}',  -- TODO: THINK ABOUT IT
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



-- Create index on events table
CREATE INDEX idx_events_venue_status ON events(venue_id, status);
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_events_onsale ON events(on_sale_time);

-- Create index on sections table
CREATE INDEX idx_sections_event ON sections(event_id);

-- Create index on seats table
CREATE UNIQUE INDEX idx_event_section_row_number ON seats (event_id, section_id, row, number);
CREATE INDEX idx_seats_section ON seats(section_id);
CREATE INDEX idx_seats_available ON seats(event_id, status) WHERE status = 'AVAILABLE';  -- Partial Indexing
CREATE INDEX idx_seats_status ON seats(status);

-- Create index on bookings table
CREATE INDEX idx_bookings_event_status ON bookings(event_id, status);
CREATE INDEX idx_bookings_user ON bookings(user_id);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_bookings_expires ON bookings(expires_at) WHERE status = 'PENDING';  -- Partial Indexing
CREATE INDEX idx_bookings_idempotency_user_key ON booking_idempotency_keys(user_id, idempotency_key);

-- Create index on payments booking
CREATE INDEX idx_payments_booking ON payments(booking_id);
CREATE UNIQUE INDEX idx_payments_idempotency_user_key ON payment_idempotency_keys(user_id, idempotency_key);
