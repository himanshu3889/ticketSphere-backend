package models

import (
	"time"
)

type Section struct {
	ID             int64     `json:"id" db:"id"`
	EventID        int64     `json:"event_id" db:"event_id"`
	Name           string    `json:"name" db:"name"`
	Capacity       int       `json:"capacity" db:"capacity"`
	SeatsRemaining int       `json:"seats_remaining" db:"seats_remaining"`
	BasePrice      float64   `json:"base_price" db:"base_price"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
