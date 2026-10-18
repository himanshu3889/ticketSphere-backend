package models

import (
	"time"
)

type QueueToken struct {
	ID         int64      `json:"id" db:"id"`
	UserID     int64      `json:"user_id" db:"user_id"`
	EventID    int64      `json:"event_id" db:"event_id"`
	Position   int        `json:"position" db:"position"`
	Status     string     `json:"status" db:"status"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at" db:"expires_at"`
	AdmittedAt *time.Time `json:"admitted_at,omitempty" db:"admitted_at"`
}
