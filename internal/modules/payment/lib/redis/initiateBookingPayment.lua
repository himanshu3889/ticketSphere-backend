-- Initiate booking payment process

local paymentIdempotencyKey = KEYS[1]
local userPaymentsKey = KEYS[2]

local paymentIdempotencyValue = ARGV[1]
local bookingId = ARGV[2]
local paymentIdempotencyTTL = tonumber(ARGV[3])
local userPaymentFieldTTL = tonumber(ARGV[4])
local userPaymentRequestData = ARGV[5]

-- Idempotency key check
local existingIdempotency = redis.call('GET', paymentIdempotencyKey)
if existingIdempotency then
    local existingPayment = redis.call('HGET', userPaymentsKey, bookingId)
    if existingPayment then
        return {"IDEMPOTENCY_HIT", existingPayment}
    end
    return {"IDEMPOTENCY_HIT", ""}
end

-- Check if user has not already the payment initiated for this booking
local existingPayment = redis.call('HGET', userPaymentsKey, bookingId)
if existingPayment then
    return {"PAYMENT_ALREADY_INITIATED", existingPayment}
end

-- Save the idempotency key
redis.call('SET', paymentIdempotencyKey, paymentIdempotencyValue, 'EX', paymentIdempotencyTTL)

-- Add to user payments hash, then expire ONLY this field
redis.call('HSET', userPaymentsKey, bookingId, userPaymentRequestData)
redis.call('HEXPIRE', userPaymentsKey, userPaymentFieldTTL, 'FIELDS', 1, bookingId)

return {"SUCCESS", userPaymentRequestData}
