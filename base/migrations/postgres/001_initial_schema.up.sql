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
    end_time TIMESTAMPTZ,
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
    version INT NOT NULL DEFAULT 0,  -- for optimistic locking
    price NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);



-- Create index on events table; user facing think about the pagination by id
CREATE INDEX idx_events_venue_status_id ON events(venue_id, status, id DESC);
CREATE INDEX idx_events_status_id ON events(status, id DESC);
CREATE INDEX idx_events_onsale_id ON events(on_sale_time, status, id DESC) WHERE status = 'ON_SALE';  
CREATE INDEX idx_events_start_time_end_time_status_id ON events(start_time, end_time, status, id DESC) WHERE status NOT IN ('CANCELLED', 'COMPLETED');

-- Create index on sections table
CREATE INDEX idx_sections_event ON sections(event_id);

-- Create index on seats table
CREATE UNIQUE INDEX idx_seats_event_section_row_number ON seats (event_id, section_id, row, number);
CREATE INDEX idx_seats_section_status_id ON seats(section_id, status, id DESC) WHERE status IN ('AVAILABLE', 'HELD');  -- Partial Indexing
CREATE INDEX idx_seats_event_status_id ON seats(event_id, status, id DESC) WHERE status IN ('AVAILABLE', 'HELD');  -- Partial Indexing
CREATE INDEX idx_seats_status_id ON seats(status, id DESC);

