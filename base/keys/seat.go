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
	SeatCachePrefix    = "ticketsphere:seat"
	SeatCacheBaseTTL   = 14 * 24 * time.Hour
	SeatCacheMaxJitter = 12 * time.Hour
)

// Cache Keys

func SeatCacheIDKey(seatID snowflake.ID) string {
	return fmt.Sprintf("%s:%d:info", SeatCachePrefix, seatID)
}

func SeatCacheBoundedKey() string {
	return SeatCachePrefix + ":info"
}

func SeatCacheIDKeyTTL() time.Duration {
	jitter := time.Duration(rand.Int64N(int64(SeatCacheMaxJitter)))
	return SeatCacheBaseTTL + jitter
}

// Bloom Filter Keys

func SeatBloomFilterBoundedKey() configs.BloomFilterKey {
	return bloomFilter.SeatIDBloomFilter
}

func SeatBloomItemIDKey(seatID snowflake.ID) string {
	return seatID.String()
}

// SeatCacheValue flattens a seat into the map we store in redis cache.
func SeatCacheValue(s *models.Seat) map[string]interface{} {
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
