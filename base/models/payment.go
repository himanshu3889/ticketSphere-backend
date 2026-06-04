package models

import (
	"time"
)

type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "PENDING"
	PaymentStatusSuccess  PaymentStatus = "SUCCESS"
	PaymentStatusFailed   PaymentStatus = "FAILED"
	BookingStatusRefunded PaymentStatus = "REFUNDED"
)

type Payment struct {
	ID           int64         `json:"id" db:"id"`
	BookingID    int64         `json:"booking_id" db:"booking_id"`
	Amount       float64       `json:"amount" db:"amount"`
	ProviderRef  string        `json:"provider_ref" db:"provider_ref"`
	Status       PaymentStatus `json:"status" db:"status"`
	ErrorMessage string        `json:"error_message,omitempty" db:"error_message"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	ProcessedAt  *time.Time    `json:"processed_at,omitempty" db:"processed_at"`
	RefundedAt   *time.Time    `json:"refunded_at,omitempty" db:"refunded_at"`
}
