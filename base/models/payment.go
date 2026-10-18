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
	PaymentStatusRefunded PaymentStatus = "REFUNDED"
)

type PaymentProvider string

const (
	PaymentProviderInternal PaymentProvider = "INTERNAL"
	PaymentProviderStripe   PaymentProvider = "STRIPE"
)

type Payment struct {
	ID                   snowflake.ID    `json:"id" db:"id"`
	BookingID            snowflake.ID    `json:"booking_id" db:"booking_id"`
	Amount               int64           `json:"amount" db:"amount"`
	Provider             PaymentProvider `json:"provider" db:"provider"`
	GatewayTransactionID *string         `json:"gateway_transaction_id" db:"gateway_transaction_id"`
	Status               PaymentStatus   `json:"status" db:"status"`
	ErrorMessage         string          `json:"error_message,omitempty" db:"error_message"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	ProcessedAt          *time.Time      `json:"processed_at,omitempty" db:"processed_at"`
	RefundedAt           *time.Time      `json:"refunded_at,omitempty" db:"refunded_at"`
}
