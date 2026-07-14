package models

import (
	"encoding/json"
	"time"

	"github.com/bwmarrin/snowflake"
)

// Booking idempotency key
type BookingIdempotencKey struct {
	IdempotencyKey string          `json:"idempotencyKey" db:"idempotency_key"`
	BookingId      snowflake.ID    `json:"bookingId" db:"booking_id"`
	UserId         snowflake.ID    `json:"userId" db:"user_id"`
	ResponseBody   json.RawMessage `json:"responseBody,omitempty" db:"response_body"`
	RequestHash    *string         `json:"requestHash" db:"request_hash"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updatedAt" db:"updated_at"`
	ExpiresAt      time.Time       `json:"expires_at" db:"expires_at"`
}

// Payment idempotency key
type PaymentIdempotencyKey struct {
	IdempotencyKey string          `json:"idempotencyKey" db:"idempotency_key"`
	PaymentId      snowflake.ID    `json:"paymentId" db:"payment_id"`
	UserId         snowflake.ID    `json:"userId" db:"user_id"`
	ResponseBody   json.RawMessage `json:"responseBody,omitempty" db:"response_body"`
	RequestHash    *string         `json:"requestHash" db:"request_hash"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updatedAt" db:"updated_at"`
	ExpiresAt      time.Time       `json:"expires_at" db:"expires_at"`
}
