package models

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

type Venue struct {
	ID        snowflake.ID `json:"id" db:"id"`
	Name      string       `json:"name" db:"name"`
	Location  string       `json:"location" db:"location"`
	Capacity  int          `json:"capacity" db:"capacity"`
	CreatedAt time.Time    `json:"-" db:"created_at"`
}
