package models

import (
	"time"
)

type EventStatus string

const (
	EventStatusActive    EventStatus = "ACTIVE"
	EventStatusOnSale    EventStatus = "ON_SALE"
	EventStatusSoldOut   EventStatus = "SOLD_OUT"
	EventStatusCancelled EventStatus = "CANCELLED"
	EventStatusCompleted EventStatus = "COMPLETED"
)

type Event struct {
	ID         int64       `json:"id" db:"id"`
	Name       string      `json:"name" db:"name"`
	VenueID    int64       `json:"venue_id" db:"venue_id"`
	StartTime  time.Time   `json:"start_time" db:"start_time"`
	OnSaleTime time.Time   `json:"on_sale_time" db:"on_sale_time"`
	Status     EventStatus `json:"status" db:"status"`
	CreatedAt  time.Time   `json:"created_at" db:"created_at"`
}
