-- Money is stored as integer whole rupees (no paise) everywhere.
-- The gateway contract uses int64 amounts; fractional parts are rounded away here.

ALTER TABLE seats     ALTER COLUMN price        TYPE BIGINT USING ROUND(price)::BIGINT;
ALTER TABLE sections  ALTER COLUMN base_price   TYPE BIGINT USING ROUND(base_price)::BIGINT;
ALTER TABLE bookings  ALTER COLUMN total_amount TYPE BIGINT USING ROUND(total_amount)::BIGINT;
ALTER TABLE payments  ALTER COLUMN amount       TYPE BIGINT USING ROUND(amount)::BIGINT;
