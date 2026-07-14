package models

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/ticketSphere-backend/configs"
)

type User struct {
	ID        snowflake.ID `db:"id" json:"id"`
	Username  string       `db:"username" json:"username"`
	Email     string       `db:"email" json:"email"`
	Password  string       `db:"password" json:"-"`
	Name      string       `db:"name" json:"name"`
	ImageUrl  string       `db:"image_url" json:"imageUrl"`
	CreatedAt time.Time    `db:"created_at" json:"-"`
	UpdatedAt time.Time    `db:"updated_at" json:"-"`
	DeletedAt *time.Time   `db:"deleted_at" json:"-"`
}

// --- user helper functions ---

// bloom filter bound key for user id
func (u User) BloomFilterIDBoundedKey() configs.BloomFilterKey {
	return bloomFilter.UserIDBloomFilter
}

// bloom filter bound key for user email
func (u User) BloomFilterEmailBoundedKey() configs.BloomFilterKey {
	return bloomFilter.UserEmailBloomFilter
}

// bloom filter key for user id
func (u User) BloomItemIDKey() string {
	return u.ID.String()
}

// bloom filter key for user email
func (u User) BloomItemEmailKey() string {
	return u.Email
}

// redis cache key for user
func (u User) CacheIDKey() string {
	return fmt.Sprintf("ticketsphere:user:%d:info", u.ID)
}

// redis cache key for user bounded
func (u User) CacheBoundedKey() string {
	return "ticketsphere:user:info"
}

// cache ttl with jitter
func (u User) CacheIDKeyTTL() time.Duration {
	baseTTL := 14 * 24 * time.Hour
	maxJitter := 12 * time.Hour

	// Generate a random duration between 0 and maxJitter
	jitter := time.Duration(rand.Int64N(int64(maxJitter)))

	return baseTTL + jitter
}

// cache value for user
func (u User) CacheIDValue() map[string]interface{} {
	return map[string]interface{}{
		"id":       u.ID,
		"username": u.Username,
		"email":    u.Email,
		"name":     u.Name,
	}
}
