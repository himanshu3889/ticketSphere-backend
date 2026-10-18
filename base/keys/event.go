package keys

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/configs"
)

const (
	EventCachePrefix    = "ticketsphere:event"
	EventCacheBaseTTL   = 14 * 24 * time.Hour
	EventCacheMaxJitter = 12 * time.Hour
)

// Cache Keys

func EventCacheIDKey(eventID snowflake.ID) string {
	return fmt.Sprintf("%s:%d:info", EventCachePrefix, eventID)
}

func EventCacheBoundedKey() string {
	return EventCachePrefix + ":info"
}

func EventCacheIDKeyTTL() time.Duration {
	jitter := time.Duration(rand.Int64N(int64(EventCacheMaxJitter)))
	return EventCacheBaseTTL + jitter
}

// Bloom Filter Keys

func EventBloomFilterBoundedKey() configs.BloomFilterKey {
	return bloomFilter.EventIDBloomFilter
}

func EventBloomItemIDKey(eventID snowflake.ID) string {
	return eventID.String()
}

// EventCacheValue flattens an event into the map we store in redis cache.
func EventCacheValue(e *models.Event) map[string]interface{} {
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
