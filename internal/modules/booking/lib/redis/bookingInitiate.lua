-- Booking atomic script
--
-- Keys:
--   KEYS[1]   = idempotency key (see keys.BookingIdempotencyKey)
--   KEYS[2]   = user bookings hash (see keys.UserBookingsKey)
--   KEYS[3..] = seat keys to lock (seat ids — script builds "seat:{id}:*" keys)
--
-- Args:
--   ARGV[1] = idempotency value (request fingerprint (64 hex chars) .. booking request json)
--   ARGV[2] = bookingId
--   ARGV[3] = seats locks value ("{userId}:{bookingId}")
--   ARGV[4] = idempotency TTL in seconds
--   ARGV[5] = user bookings field TTL in seconds
--   ARGV[6] = seats lock TTL in seconds
--   ARGV[7] = user booking request json (its "total_amount" field is patched in here)
--   ARGV[8] = request fingerprint (sha256 hex of event id + sorted seat ids)
--
-- Seat price lookup is folded in: prices live inside the seat info cache JSON
-- (ticketsphere:seat:{id}:info — the same object GetEventSeats maintains). One
-- source of truth: no separate price key that could drift from it. Prices are
-- pulled out with string.find ONLY — never cjson: the payload's 63-bit
-- snowflake ids would lose precision in Lua 5.1 doubles.
-- If the info key is missing/unreadable the script returns
-- {"PRICE_UNAVAILABLE", seatId} BEFORE locking anything — the caller then warms
-- via the legacy path (cache -> db) and retries. This keeps the hot path at one
-- round trip.

local bookingIdempotencyKey = KEYS[1]
local userBookingsKey = KEYS[2]

-- ARGV[1] kept for signature compatibility; the stored value is rebuilt below
-- from the patched json so replayed responses carry the real total_amount.
local bookingId = ARGV[2]
local seatsLockValue = ARGV[3]
local bookingIdempotencyTTL = tonumber(ARGV[4])
local userBookingFieldTTL = tonumber(ARGV[5])
local seatsLockTTL = tonumber(ARGV[6])
local userBookingRequestData = ARGV[7]
local requestFingerprint = ARGV[8]

-- Idempotent booking check: a key is only replayable with the same request.
-- The stored value is "fingerprint .. request json"; the fingerprint is fixed
-- length (sha256 hex = 64 chars) so plain substring splitting avoids cjson
-- re-encoding the payload (Lua 5.1 doubles lose precision on 63-bit snowflake IDs).
local existing = redis.call('GET', bookingIdempotencyKey)
if existing then
    local storedFingerprint = string.sub(existing, 1, 64)
    if storedFingerprint ~= requestFingerprint then
        return { "IDEMPOTENCY_CONFLICT" }
    end
    return { "IDEMPOTENCY_HIT", string.sub(existing, 65) }
end

-- Check all seats: not booked, not locked, price resolvable (sum as we go).
local seatCount = #KEYS - 2
local totalAmount = 0
for i = 1, seatCount do
    local seatId = KEYS[2 + i]
    local seatKey = "seat:" .. seatId

    local seatBookingKey = seatKey .. ":booking"
    local isCurrentlyBooked = redis.call('GET', seatBookingKey)
    if isCurrentlyBooked then
        return { "SEAT_ALREADY_BOOKED", seatId }
    end

    local seatLockKey = seatKey .. ":lock"
    local isCurrentlyLock = redis.call('GET', seatLockKey)
    if isCurrentlyLock then
        return { "SEAT_ALREADY_LOCKED", seatId }
    end

    -- price comes from the seat info cache JSON — string.find, never cjson
    local seatInfoKey = "ticketsphere:seat:" .. seatId .. ":info"
    local seatInfo = redis.call('GET', seatInfoKey)
    if not seatInfo then
        return { "PRICE_UNAVAILABLE", seatId }
    end
    local priceRaw = string.match(seatInfo, '"price":(%d+)')
    if not priceRaw then
        return { "PRICE_UNAVAILABLE", seatId }
    end
    totalAmount = totalAmount + tonumber(priceRaw)
end

-- Acquire seats take locks
for i = 1, seatCount do
    local seatId = KEYS[2 + i]
    local seatLockKey = "seat:" .. seatId .. ":lock"
    redis.call('SETEX', seatLockKey, seatsLockTTL, seatsLockValue)
end

-- Patch total_amount into the stored json. String sub ONLY — never cjson the
-- payload (63-bit snowflake ids would lose precision in Lua 5.1 doubles).
local patchedJson = string.gsub(userBookingRequestData, '"total_amount":%d+', '"total_amount":' .. totalAmount)

-- Set pending booking idempotency key with TTL (value embeds the patched json,
-- so IDEMPOTENCY_HIT replays also return the real total).
redis.call('SETEX', bookingIdempotencyKey, bookingIdempotencyTTL, requestFingerprint .. patchedJson)

-- Add to user bookings hash, then expire ONLY this field
redis.call('HSET', userBookingsKey, bookingId, patchedJson)
redis.call('HEXPIRE', userBookingsKey, userBookingFieldTTL, 'FIELDS', 1, bookingId)

return { "SUCCESS", patchedJson, totalAmount }
