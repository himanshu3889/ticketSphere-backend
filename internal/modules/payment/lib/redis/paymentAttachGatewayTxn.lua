-- Attach the gateway transaction id to a claimed payment slot.
-- Only the claim owner may do this. Hash field TTL survives HSET, so the
-- field expiry set at claim time keeps ticking.
--
-- Keys:
--   KEYS[1] = user payments hash (see keys.UserPaymentsKey)
--
-- Args:
--   ARGV[1] = bookingId (hash field)
--   ARGV[2] = paymentId (ownership token)
--   ARGV[3] = final value ("{paymentId}|{finalPaymentJson}")

local userPaymentsKey = KEYS[1]

local bookingId = ARGV[1]
local paymentId = ARGV[2]
local finalValue = ARGV[3]

local existing = redis.call('HGET', userPaymentsKey, bookingId)
if not existing then
    return { "NOT_FOUND" }
end

-- Match the "{paymentId}|" prefix
local sep = string.find(existing, "|", 1, true)
if not sep or string.sub(existing, 1, sep - 1) ~= paymentId then
    return { "NOT_OWNER" }
end

redis.call('HSET', userPaymentsKey, bookingId, finalValue)
return { "ATTACHED", finalValue }
