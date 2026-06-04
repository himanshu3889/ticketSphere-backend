package models

import (
	"encoding/json"
	"time"
)

type BookingIdempotencKey struct {
	IdempotencyKey string          `json:"idempotencyKey" db:"idempotency_key"`
	BookingId      int64           `json:"bookingId" db:"booking_id"`
	UserId         int64           `json:"userId" db:"user_id"`
	ResponseBody   json.RawMessage `json:"responseBody,omitempty" db:"response_body"`
	RequestHash    *string         `json:"requestHash" db:"request_hash"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updatedAt" db:"updated_at"`
	ExpiresAt      time.Time       `json:"expires_at" db:"expires_at"`
}

type PaymentIdempotencyKey struct {
	IdempotencyKey string          `json:"idempotencyKey" db:"idempotency_key"`
	PaymentId      int64           `json:"paymentId" db:"payment_id"`
	UserId         int64           `json:"userId" db:"user_id"`
	ResponseBody   json.RawMessage `json:"responseBody,omitempty" db:"response_body"`
	RequestHash    *string         `json:"requestHash" db:"request_hash"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updatedAt" db:"updated_at"`
	ExpiresAt      time.Time       `json:"expires_at" db:"expires_at"`
}
