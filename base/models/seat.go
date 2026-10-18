package models

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

type SeatStatus string

const (
	SeatStatusAvailable SeatStatus = "AVAILABLE"
	SeatStatusHeld      SeatStatus = "HELD"
	SeatStatusBooked    SeatStatus = "BOOKED"
	SeatStatusBlocked   SeatStatus = "BLOCKED"
)

type Seat struct {
	ID        snowflake.ID `json:"id" db:"id"`
	EventID   snowflake.ID `json:"event_id" db:"event_id"`
	SectionID snowflake.ID `json:"section_id" db:"section_id"`
	Row       string       `json:"row" db:"row"`
	Number    string       `json:"number" db:"number"`
	Status    SeatStatus   `json:"status" db:"status"`
	Version   int          `json:"-" db:"version"`
	Price     int64        `json:"price" db:"price"`
	UpdatedAt time.Time    `json:"-" db:"updated_at"`
}
