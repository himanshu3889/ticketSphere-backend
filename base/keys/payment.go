package keys

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
)

const (
	PaymentCachePrefix    = "ticketsphere:payment"
	PaymentCacheBaseTTL   = 14 * 24 * time.Hour
	PaymentCacheMaxJitter = 12 * time.Hour

	PaymentIdempotencyTTL = 30 * time.Minute
	UserLatestPaymentsTTL = 60 * time.Minute
)

// Model Cache Keys

func PaymentCacheIDKey(paymentID snowflake.ID) string {
	return fmt.Sprintf("%s:%d:info", PaymentCachePrefix, paymentID)
}

func PaymentCacheBoundedKey() string {
	return PaymentCachePrefix + ":info"
}

func PaymentCacheIDKeyTTL() time.Duration {
	jitter := time.Duration(rand.Int64N(int64(PaymentCacheMaxJitter)))
	return PaymentCacheBaseTTL + jitter
}

// Operational Keys

func PaymentIdempotencyKey(userID snowflake.ID, rawKey string) string {
	return fmt.Sprintf("payment:idempotency:%d:%s", userID, rawKey)
}

func UserPaymentsKey(userID snowflake.ID) string {
	return fmt.Sprintf("user:%d:payments:latest", userID)
}

// PaymentCacheValue flattens a payment into the map we store in redis cache.
func PaymentCacheValue(p *models.Payment) map[string]interface{} {
	return map[string]interface{}{
		"id":                     p.ID,
		"booking_id":             p.BookingID,
		"amount":                 p.Amount,
		"provider":               p.Provider,
		"gateway_transaction_id": p.GatewayTransactionID,
		"status":                 p.Status,
		"error_message":          p.ErrorMessage,
	}
}
