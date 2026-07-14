package models

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "PENDING"
	PaymentStatusSuccess  PaymentStatus = "SUCCESS"
	PaymentStatusFailed   PaymentStatus = "FAILED"
	BookingStatusRefunded PaymentStatus = "REFUNDED"
)

type Payment struct {
	ID           snowflake.ID  `json:"id" db:"id"`
	BookingID    snowflake.ID  `json:"booking_id" db:"booking_id"`
	Amount       float64       `json:"amount" db:"amount"`
	ProviderRef  string        `json:"provider_ref" db:"provider_ref"`
	Status       PaymentStatus `json:"status" db:"status"`
	ErrorMessage string        `json:"error_message,omitempty" db:"error_message"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	ProcessedAt  *time.Time    `json:"processed_at,omitempty" db:"processed_at"`
	RefundedAt   *time.Time    `json:"refunded_at,omitempty" db:"refunded_at"`
}
