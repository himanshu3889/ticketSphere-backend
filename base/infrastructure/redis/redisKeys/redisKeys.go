package rediskeys

import (
	"fmt"

	"github.com/bwmarrin/snowflake"
)

// User
type userKeys struct{}

func (k userKeys) Info(id snowflake.ID) (string, string) {
	return fmt.Sprintf("ticketsphere:user:%d:info", id), "user:id:info" // cacheKey, "entity:operation"
}

// Usage
var Keys = struct {
	User userKeys
}{}
