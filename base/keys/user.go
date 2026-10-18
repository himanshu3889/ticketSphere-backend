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
	UserCachePrefix    = "ticketsphere:user"
	UserCacheBaseTTL   = 14 * 24 * time.Hour
	UserCacheMaxJitter = 12 * time.Hour
)

// Cache Keys

func UserCacheIDKey(userID snowflake.ID) string {
	return fmt.Sprintf("%s:%d:info", UserCachePrefix, userID)
}

func UserCacheBoundedKey() string {
	return UserCachePrefix + ":info"
}

func UserCacheIDKeyTTL() time.Duration {
	jitter := time.Duration(rand.Int64N(int64(UserCacheMaxJitter)))
	return UserCacheBaseTTL + jitter
}

// Bloom Filter Keys

func UserBloomFilterBoundedKey() configs.BloomFilterKey {
	return bloomFilter.UserIDBloomFilter
}

func UserEmailBloomFilterBoundedKey() configs.BloomFilterKey {
	return bloomFilter.UserEmailBloomFilter
}

func UserBloomItemIDKey(userID snowflake.ID) string {
	return userID.String()
}

func UserBloomItemEmailKey(email string) string {
	return email
}

// UserCacheValue flattens a user into the map we store in redis cache.
func UserCacheValue(u *models.User) map[string]interface{} {
	return map[string]interface{}{
		"id":       u.ID,
		"username": u.Username,
		"email":    u.Email,
		"name":     u.Name,
	}
}
