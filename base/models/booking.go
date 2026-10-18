package models

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "PENDING"
	BookingStatusConfirmed BookingStatus = "CONFIRMED"
	BookingStatusFailed    BookingStatus = "FAILED"
	BookingStatusCancelled BookingStatus = "CANCELLED"
	BookingStatusExpired   BookingStatus = "EXPIRED"
)

// Base booking
type Booking struct {
	ID          snowflake.ID  `json:"id" db:"id"`
	EventID     snowflake.ID  `json:"event_id" db:"event_id"`
	UserID      snowflake.ID  `json:"user_id" db:"user_id"`
	Status      BookingStatus `json:"status" db:"status"`
	TotalAmount int64         `json:"total_amount" db:"total_amount"`
	ExpiresAt   time.Time     `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	ConfirmedAt *time.Time    `json:"confirmed_at,omitempty" db:"confirmed_at"`
	CancelledAt *time.Time    `json:"cancelled_at,omitempty" db:"cancelled_at"`
}

// Seats in booking
type BookingSeats struct {
	BookingID snowflake.ID `json:"booking_id" db:"booking_id"`
	SeatID    snowflake.ID `json:"seat_id" db:"seat_id"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
}

// Booking Request for the seats of a event
// Not saved in sql db
type BookingRequest struct {
	// Copy of booking
	ID          snowflake.ID  `json:"id"`
	EventID     snowflake.ID  `json:"event_id"`
	UserID      snowflake.ID  `json:"user_id"`
	Status      BookingStatus `json:"status"`
	TotalAmount int64         `json:"total_amount"`
	ExpiresAt   time.Time     `json:"expires_at"`
	CreatedAt   time.Time     `json:"created_at"`
	ConfirmedAt *time.Time    `json:"confirmed_at,omitempty"`
	CancelledAt *time.Time    `json:"cancelled_at,omitempty"`
	// Extra seat ids
	SeatIDs []snowflake.ID `json:"seat_ids"`
}
