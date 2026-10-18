-- Claim the payment slot for a booking BEFORE touching the gateway.
-- One atomic unit: idempotency check, slot check, write the claim.
-- Only one caller per booking can win CLAIMED.
--
-- Keys:
--   KEYS[1] = payment idempotency key (see keys.PaymentIdempotencyKey)
--   KEYS[2] = user payments hash (see keys.UserPaymentsKey)
--
-- Args:
--   ARGV[1] = bookingId (hash field)
--   ARGV[2] = claim value ("{paymentId}|{paymentJson}") — the prefix is the
--             ownership token checked by the attach/release scripts
--   ARGV[3] = idempotency TTL in seconds
--   ARGV[4] = user payment field TTL in seconds

local paymentIdempotencyKey = KEYS[1]
local userPaymentsKey = KEYS[2]

local bookingId = ARGV[1]
local claimValue = ARGV[2]
local idempotencyTTL = tonumber(ARGV[3])
local userPaymentFieldTTL = tonumber(ARGV[4])

-- Same key replayed: hand back the slot state
local existingIdempotency = redis.call('GET', paymentIdempotencyKey)
if existingIdempotency then
    local existingPayment = redis.call('HGET', userPaymentsKey, bookingId)
    if existingPayment then
        return { "IDEMPOTENCY_HIT", existingPayment }
    end
end

-- Slot taken by any flow
local existingPayment = redis.call('HGET', userPaymentsKey, bookingId)
if existingPayment then
    return { "PAYMENT_HIT", existingPayment }
end

-- Slot free: take it
redis.call('SET', paymentIdempotencyKey, bookingId, 'EX', idempotencyTTL)
redis.call('HSET', userPaymentsKey, bookingId, claimValue)
redis.call('HEXPIRE', userPaymentsKey, userPaymentFieldTTL, 'FIELDS', 1, bookingId)

return { "CLAIMED", claimValue }
