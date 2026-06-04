package models

import (
	"time"
)

type SeatStatus string

const (
	SeatStatusAvailable SeatStatus = "AVAILABLE"
	SeatStatusHeld      SeatStatus = "HELD"
	SeatStatusBooked    SeatStatus = "BOOKED"
	SeatStatusBlocked   SeatStatus = "BLOCKED"
)

type Seat struct {
	ID        int64      `json:"id" db:"id"`
	EventID   int64      `json:"event_id" db:"event_id"`
	SectionID int64      `json:"section_id" db:"section_id"`
	Row       string     `json:"row" db:"row"`
	Number    string     `json:"number" db:"number"`
	Status    SeatStatus `json:"status" db:"status"`
	Version   int        `json:"version" db:"version"`
	Price     float64    `json:"price" db:"price"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}
