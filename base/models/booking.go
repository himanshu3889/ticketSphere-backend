package models

import (
	"time"

	"github.com/lib/pq"
)

type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "PENDING"
	BookingStatusConfirmed BookingStatus = "CONFIRMED"
	BookingStatusFailed    BookingStatus = "FAILED"
	BookingStatusCancelled BookingStatus = "CANCELLED"
	BookingStatusExpired   BookingStatus = "EXPIRED"
)

type Booking struct {
	ID          int64         `json:"id" db:"id"`
	EventID     int64         `json:"event_id" db:"event_id"`
	UserID      int64         `json:"user_id" db:"user_id"`
	SeatIDs     pq.Int64Array `json:"seat_ids" db:"seat_ids"` // fixing it
	Status      BookingStatus `json:"status" db:"status"`
	TotalAmount float64       `json:"total_amount" db:"total_amount"`
	ExpiresAt   time.Time     `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	ConfirmedAt *time.Time    `json:"confirmed_at,omitempty" db:"confirmed_at"`
	CancelledAt *time.Time    `json:"cancelled_at,omitempty" db:"cancelled_at"`
}
