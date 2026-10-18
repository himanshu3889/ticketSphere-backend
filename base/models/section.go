package models

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

type Section struct {
	ID             snowflake.ID `json:"id" db:"id"`
	EventID        snowflake.ID `json:"event_id" db:"event_id"`
	Name           string       `json:"name" db:"name"`
	Capacity       int          `json:"capacity" db:"capacity"`
	SeatsRemaining int          `json:"seats_remaining" db:"seats_remaining"`
	BasePrice      int64        `json:"base_price" db:"base_price"`
	CreatedAt      time.Time    `json:"created_at" db:"created_at"`
}
