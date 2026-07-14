package models

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/ticketSphere-backend/configs"
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

// --- event helper functions ---

// bloom filter bound key for event id
func (e Event) BloomFilterIDBoundedKey() configs.BloomFilterKey {
	return bloomFilter.EventIDBloomFilter
}

// bloom filter key for event id
func (e Event) BloomItemIDKey() string {
	return e.ID.String()
}

// redis cache key for event
func (e Event) CacheIDKey() string {
	return fmt.Sprintf("ticketsphere:event:%d:info", e.ID)
}

// redis cache key for event bounded
func (e Event) CacheBoundedKey() string {
	return "ticketsphere:event:info"
}

// cache ttl with jitter
func (e Event) CacheIDKeyTTL() time.Duration {
	baseTTL := 14 * 24 * time.Hour
	maxJitter := 12 * time.Hour

	// Generate a random duration between 0 and maxJitter
	jitter := time.Duration(rand.Int64N(int64(maxJitter)))

	return baseTTL + jitter
}

// cache value for event
func (e Event) CacheIDValue() map[string]interface{} {
	return map[string]interface{}{
		"id":           e.ID,
		"name":         e.Name,
		"venue_id":     e.VenueID,
		"start_time":   e.StartTime,
		"end_time":     e.EndTime,
		"on_sale_time": e.OnSaleTime,
		"status":       e.Status,
	}
}
