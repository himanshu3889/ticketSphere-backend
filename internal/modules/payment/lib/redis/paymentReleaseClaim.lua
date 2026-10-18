-- Release a payment claim after a gateway failure, so a retry starts clean.
-- Owner-checked: the slot may have expired and been re-claimed by another
-- attempt meanwhile, and HDEL must not steal it.
--
-- Keys:
--   KEYS[1] = payment idempotency key (see keys.PaymentIdempotencyKey)
--   KEYS[2] = user payments hash (see keys.UserPaymentsKey)
--
-- Args:
--   ARGV[1] = bookingId (hash field)
--   ARGV[2] = paymentId (ownership token)

local paymentIdempotencyKey = KEYS[1]
local userPaymentsKey = KEYS[2]

local bookingId = ARGV[1]
local paymentId = ARGV[2]

local existing = redis.call('HGET', userPaymentsKey, bookingId)
if not existing then
    return { "EMPTY" }
end

-- Match the "{paymentId}|" prefix
local sep = string.find(existing, "|", 1, true)
if sep and string.sub(existing, 1, sep - 1) == paymentId then
    redis.call('HDEL', userPaymentsKey, bookingId)
    redis.call('DEL', paymentIdempotencyKey)
    return { "RELEASED" }
end

return { "NOT_OWNER" }
