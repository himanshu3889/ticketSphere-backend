package bookingScripts

import (
	_ "embed"

	"github.com/redis/go-redis/v9"
)

// BookingInitiateScript locks seats atomically with idempotency.
// RETURNS: status: string, data: string
//
//go:embed bookingInitiate.lua
var bookingInitiateLua string

var BookingInitiateScript = redis.NewScript(bookingInitiateLua)

// BookingRollbackScript frees what BookingInitiateScript created when the
// publish after it fails.
// RETURNS: status: string, released lock count
//
//go:embed bookingRollback.lua
var bookingRollbackLua string

var BookingRollbackScript = redis.NewScript(bookingRollbackLua)
