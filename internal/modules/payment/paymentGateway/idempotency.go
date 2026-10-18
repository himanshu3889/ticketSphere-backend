package paymentGateway

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// GenerateIdempotencyKey returns a fresh UUID, prefixed when prefix is given.
func GenerateIdempotencyKey(prefix string) string {
	id := uuid.Must(uuid.NewRandom()).String()
	if prefix != "" {
		return fmt.Sprintf("%s-%s", prefix, id)
	}
	return id
}

// HashRequestBody is the SHA256 hex digest of a request body.
// NOTE: use this if wanna cache idempotency responses locally
func HashRequestBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
