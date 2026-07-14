package models

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/ticketSphere-backend/configs"
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
	Price     float64      `json:"price" db:"price"`
	UpdatedAt time.Time    `json:"-" db:"updated_at"`
}

// --- event helper functions ---

// bloom filter bound key for event id
func (s Seat) BloomFilterIDBoundedKey() configs.BloomFilterKey {
	return bloomFilter.SeatIDBloomFilter
}

// bloom filter key for event id
func (s Seat) BloomItemIDKey() string {
	return s.ID.String()
}

// bloom filter key for event id
func (s Seat) BloomItemIDKeySeat(seatID snowflake.ID) string {
	return seatID.String()
}

// redis cache key for event
func (s Seat) CacheIDKey() string {
	return fmt.Sprintf("ticketsphere:seat:%d:info", s.ID)
}

// redis cache key for event seat
func (s Seat) CacheIDKeySeat(seatID snowflake.ID) string {
	return fmt.Sprintf("ticketsphere:seat:%d:info", seatID)
}

// redis cache key for event bounded
func (s Seat) CacheBoundedKey() string {
	return "ticketsphere:seat:info"
}

// cache ttl with jitter
func (s Seat) CacheIDKeyTTL() time.Duration {
	baseTTL := 14 * 24 * time.Hour
	maxJitter := 12 * time.Hour

	// Generate a random duration between 0 and maxJitter
	jitter := time.Duration(rand.Int64N(int64(maxJitter)))

	return baseTTL + jitter
}

// cache value for event
func (s Seat) CacheIDValue() map[string]interface{} {
	return map[string]interface{}{
		"id":         s.ID,
		"event_id":   s.EventID,
		"section_id": s.SectionID,
		"row":        s.Row,
		"number":     s.Number,
		"status":     s.Status,
		"price":      s.Price,
	}
}
