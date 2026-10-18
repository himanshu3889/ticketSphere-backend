package paymentScripts

import (
	_ "embed"

	"github.com/redis/go-redis/v9"
)

// PaymentClaimSlotScript claims a booking's payment slot atomically.
// RETURNS: {"IDEMPOTENCY_HIT"|"PAYMENT_HIT"|"CLAIMED", value}
//
// KEYS:
//
//	KEYS[1] = payment idempotency key (see keys.PaymentIdempotencyKey)
//	KEYS[2] = user payments hash (see keys.UserPaymentsKey)
//
// ARGV:
//
//	ARGV[1] = bookingId
//	ARGV[2] = claim value ("{paymentId}|{paymentJson}")
//	ARGV[3] = payment idempotency TTL in seconds
//	ARGV[4] = user payment field TTL in seconds
//
//go:embed paymentClaimSlot.lua
var paymentClaimSlotLua string

var PaymentClaimSlotScript = redis.NewScript(paymentClaimSlotLua)

// PaymentAttachGatewayTxnScript attaches the gateway txn id to the claimed slot.
// Owner-checked. RETURNS: {"ATTACHED", value} | {"NOT_FOUND"} | {"NOT_OWNER"}
//
//go:embed paymentAttachGatewayTxn.lua
var paymentAttachGatewayTxnLua string

var PaymentAttachGatewayTxnScript = redis.NewScript(paymentAttachGatewayTxnLua)

// PaymentReleaseClaimScript frees the slot after a gateway failure. Owner-checked.
// RETURNS: {"RELEASED"} | {"NOT_OWNER"} | {"EMPTY"}
//
//go:embed paymentReleaseClaim.lua
var paymentReleaseClaimLua string

var PaymentReleaseClaimScript = redis.NewScript(paymentReleaseClaimLua)
