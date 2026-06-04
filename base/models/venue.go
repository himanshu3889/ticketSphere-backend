package models

import (
	"time"
)

type Venue struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Location  string    `json:"location" db:"location"`
	Capacity  int       `json:"capacity" db:"capacity"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
