package bookingScripts

import (
	"github.com/redis/go-redis/v9"
)

// BookingScript locks seats atomically with idempotency
// RETURNS: status: int, message: string, data: string
// STATUS: 0 = success, 1 = idempotency_hit, 2 = seat_locked
var BookingScript = redis.NewScript(`
-- Booking atomic script
-- 
-- Keys:
--   KEYS[1]   = idempotency key (booking:idempotency:{bookingIdempotencyKey})
--   KEYS[2]   = user bookings hash (user:{userId}:bookings:latest)
--   KEYS[3..] = seat keys to lock (seat:{seatId})
--
-- Args:
--   ARGV[1] = idempotency value ("{bookingId}")
--   ARGV[2] = bookingId
--   ARGV[3] = seats locks value ("{userId}:{bookingId}")
--   ARGV[4] = idempotency TTL in seconds
--   ARGV[5] = user bookings field TTL in seconds
--   ARGV[6] = seats lock TTL in seconds
--   ARGV[7] = user booking request json

local bookingIdempotencyKey = KEYS[1]
local userBookingsKey = KEYS[2]

local bookingIdempotencyValue = ARGV[1]
local bookingId = ARGV[2]
local seatsLockValue = ARGV[3]
local bookingIdempotencyTTL = tonumber(ARGV[4])
local userBookingFieldTTL = tonumber(ARGV[5])
local seatsLockTTL = tonumber(ARGV[6])
local userBookingRequestData = ARGV[7]



-- Idempotent booking check
local existing = redis.call('GET', bookingIdempotencyKey)
if existing then
    return {1, "IDEMPOTENCY_HIT", existing}
end

-- Check all seats are available, then lock them
local seatCount = #KEYS - 2
for i = 1, seatCount do
    local seatKey = KEYS[2 + i]
    local current = redis.call('GET', seatKey)
    
    if current then
        return {2, "SEAT_ALREADY_LOCKED", seatKey}
    end
end

-- Acquire seats
for i = 1, seatCount do
    local seatKey = KEYS[2 + i]
    redis.call('SETEX', seatKey, seatsLockTTL, seatsLockValue)
end

-- Set pending booking idempotency key with TTL
redis.call('SETEX', bookingIdempotencyKey, bookingIdempotencyTTL, bookingIdempotencyValue)

-- Add to user bookings hash, then expire ONLY this field
redis.call('HSET', userBookingsKey, bookingId, userBookingRequestData)
redis.call('HEXPIRE', userBookingsKey, userBookingFieldTTL, 'FIELDS', 1, bookingId)

return {0, "SUCCESS", userBookingRequestData}
`)
