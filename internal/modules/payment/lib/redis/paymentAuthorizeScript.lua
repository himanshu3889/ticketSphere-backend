-- Payment authorize script: confirm a pending booking after payment is authorized
--
-- Keys:
--   KEYS[1]   = user bookings hash (see keys.UserBookingsKey)
--
-- Args:
--   ARGV[1] = booking id
--   ARGV[2] = confirmed-at timestamp string
--   ARGV[3] = user bookings field TTL in seconds

local userBookingsKey = KEYS[1]

local bookingId = ARGV[1]
local confirmedAtString = ARGV[2]
local userBookingFieldTTL = tonumber(ARGV[3])

-- get booking details
local userBookingsJson = redis.call('HGET', userBookingsKey, bookingId)

if not userBookingsJson then
    return { "NOT_FOUND", "" }
end

-- decode booking json
local userBookingsData = cjson.decode(userBookingsJson)


-- booking COMPLETED
if userBookingsData.status ~= "PENDING" then
    return { "BOOKING_NOT_PENDING", "" }
end

-- Get all booking seat ids
local bookingSeatIds = userBookingsData.seat_ids

-- Mark seats booked
for _, seatId in ipairs(bookingSeatIds) do
    local seatKey = "seat:" .. seatId
    local seatBookingKey = seatKey .. ":booking"
    redis.call('SET', seatBookingKey, bookingId)
end

-- Update user latest booking status 
userBookingsData.status = "CONFIRMED"
userBookingsData.confirmed_at = confirmedAtString

-- Encode back to JSON and save
local updatedJson = cjson.encode(userBookingsData)
redis.call('HSET', userBookingsKey, bookingId, updatedJson)
redis.call('HEXPIRE', userBookingsKey, userBookingFieldTTL, 'FIELDS', 1, bookingId)

return { "SUCCESS", "" }
