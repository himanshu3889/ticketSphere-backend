-- Booking Rollback script
-- frees the seat locks, the idempotency key and the user bookings hash entry

--
-- Keys:
--   KEYS[1]   = idempotency key (see keys.BookingIdempotencyKey)
--   KEYS[2]   = user bookings hash (see keys.UserBookingsKey)
--   KEYS[3..] = seat ids
--
-- Args:
--   ARGV[1] = bookingId
--   ARGV[2] = seat lock value ("{userId}:{bookingId}")

-- Only free locks we still own. Once a lock's TTL expires the seat can be
-- locked by someone else, and DEL would steal their lock.
--

local idempotencyKey = KEYS[1]
local userBookingHash = KEYS[2]

local bookingId = ARGV[1]
local seatLockValue = ARGV[2]

local releasedCount = 0
for i = 3, #KEYS do
    local seatLockKey = "seat:" .. KEYS[i] .. ":lock"
    if redis.call('GET', seatLockKey) == seatLockValue then
        redis.call('DEL', seatLockKey)
        releasedCount = releasedCount + 1
    end
end

redis.call('DEL', idempotencyKey)
redis.call('HDEL', userBookingHash, bookingId)

return { "ROLLED_BACK", releasedCount }
