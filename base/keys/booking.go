package keys

import (
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
)

const (
	// Must outlive everything it guards (BookingSeatsLockTTL) plus payment slack,
	// so a retry with the same key always replays instead of hitting its own locks.
	BookingIdempotencyTTL = 30 * time.Minute
	UserLatestBookingsTTL = 60 * time.Minute
	BookingSeatsLockTTL   = 15 * time.Minute
)

// Operational Keys

func BookingIdempotencyKey(userID snowflake.ID, rawKey string) string {
	return fmt.Sprintf("booking:idempotency:%d:%s", userID, rawKey)
}

func UserBookingsKey(userID snowflake.ID) string {
	return fmt.Sprintf("user:%d:bookings:latest", userID)
}

func SeatKey(seatID snowflake.ID) string {
	return fmt.Sprintf("%d", seatID)
}

func SeatLockValue(userID snowflake.ID, bookingID snowflake.ID) string {
	return fmt.Sprintf("%d:%d", userID, bookingID)
}
