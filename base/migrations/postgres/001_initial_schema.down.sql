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
DROP TYPE IF EXISTS seat_status CASCADE;
DROP TYPE IF EXISTS event_status CASCADE;
