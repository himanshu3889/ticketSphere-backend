package models

import (
	"time"

	"github.com/bwmarrin/snowflake"
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
	ID         snowflake.ID `json:"id" db:"id"`
	Name       string       `json:"name" db:"name"`
	VenueID    snowflake.ID `json:"venue_id" db:"venue_id"`
	StartTime  time.Time    `json:"start_time" db:"start_time"`
	EndTime    time.Time    `json:"end_time" db:"end_time"`
	OnSaleTime time.Time    `json:"on_sale_time" db:"on_sale_time"`
	Status     EventStatus  `json:"status" db:"status"`
	CreatedAt  time.Time    `json:"-" db:"created_at"`
}
