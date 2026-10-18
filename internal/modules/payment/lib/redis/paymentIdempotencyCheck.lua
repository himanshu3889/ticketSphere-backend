local paymentIdempotencyKey = KEYS[1]
local userPaymentsKey = KEYS[2]

local bookingId = ARGV[1]

-- Idempotency key check
local existingIdempotency = redis.call('GET', paymentIdempotencyKey)
if existingIdempotency then
    local existingPayment = redis.call('HGET', userPaymentsKey, bookingId)
    if existingPayment then
        return {"IDEMPOTENCY_HIT", existingPayment}
    end
    return {"IDEMPOTENCY_HIT", ""}
end

-- Check payment details
local existingPayment = redis.call('HGET', userPaymentsKey, bookingId)
if existingPayment then
    return {"PAYMENT_HIT", existingPayment}
end

return {"MISS", ""}
