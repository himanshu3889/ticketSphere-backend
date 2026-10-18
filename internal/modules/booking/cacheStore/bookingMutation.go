package bookingCacheStore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	"github.com/himanshu3889/ticketSphere-backend/base/keys"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
	bookingScripts "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/lib/redis"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
)

type BookingResult struct {
	Status  int         `json:"status"` // 0=success, 1=idempotency, 2=seat_taken
	Message string      `json:"message"`
	ID      string      `json:"id"`
	Data    interface{} `json:"data,omitempty"` // only for status 0/1
}

// fingerprintLen is the hex sha256 length.
const fingerprintLen = sha256.Size * 2 // 64

// hashes the client-controlled fields that define request identity
// seat order is normalized so {1,2} and {2,1} are the same request.
func fingerprintBookingRequest(bookingReq *models.BookingRequest) string {
	sortedSeatIDs := slices.Clone(bookingReq.SeatIDs)
	slices.Sort(sortedSeatIDs)
	sum := sha256.Sum256([]byte(fmt.Sprintf("event:%d|seats:%v", bookingReq.EventID, sortedSeatIDs)))
	return hex.EncodeToString(sum[:])
}

// Book Seats using Redis and RabbitMQ
func (cm *CacheStore) BookSeats(ctx context.Context, rawIdempotencyKey string, bookingReq *models.BookingRequest) (bookingResult *BookingResult, appErr *appError.Error) {
	// Start a otel span for the business logic
	ctx, span := cm.Tracer.Start(ctx, constants.SpanSeatBookingExecute)
	defer utils.EndSpanAppErr(span, appErr)

	const seatsMaxCount = 10

	// Quick validations
	if _, err := uuid.Parse(rawIdempotencyKey); err != nil {
		return nil, appError.NewBadRequest("Idempotency-Key must be a valid UUID")
	}
	if bookingReq.EventID <= 0 {
		return nil, appError.NewBadRequest("Invalid event id")
	}
	if bookingReq.UserID <= 0 {
		return nil, appError.NewBadRequest("Invalid user id")
	}
	if len(bookingReq.SeatIDs) == 0 {
		return nil, appError.NewBadRequest("No seat ids provided")
	}
	if len(bookingReq.SeatIDs) > seatsMaxCount {
		return nil, appError.NewBadRequest(fmt.Sprintf("Limit of %d seats per booking exceeded", seatsMaxCount))
	}

	// built safe idempotency key
	safeIdempotencyKey := keys.BookingIdempotencyKey(bookingReq.UserID, rawIdempotencyKey)
	fingerprint := fingerprintBookingRequest(bookingReq)

	// Build request
	seatsLockTTLSec := int(keys.BookingSeatsLockTTL.Seconds()) // 900

	bookingReq.ID = utils.GenerateSnowflakeID(cm.Store.SnowflakeNode)
	bookingReq.Status = models.BookingStatusPending
	bookingReq.CreatedAt = time.Now()
	bookingReq.ExpiresAt = time.Now().Add(time.Duration(seatsLockTTLSec) * time.Second)
	// TotalAmount stays zero here: the Lua script sums prices out of the seat
	// info cache, patches it into the stored JSON and returns it as arr[2] (SUCCESS).

	// Key setup
	userBookingsKey := keys.UserBookingsKey(bookingReq.UserID)
	idempotencyTTL := int(keys.BookingIdempotencyTTL.Seconds())
	userLatestBookingsTTL := int(keys.UserLatestBookingsTTL.Seconds())

	// Add keys
	scriptKeys := []string{safeIdempotencyKey, userBookingsKey}
	for _, seat := range bookingReq.SeatIDs {
		scriptKeys = append(scriptKeys, keys.SeatKey(seat))
	}

	// Seat lock value
	seatLockValue := keys.SeatLockValue(bookingReq.UserID, bookingReq.ID)
	// Marshal booking request
	reqBytes, err := json.Marshal(bookingReq)
	if err != nil {
		cm.logger.WithError(err).WithFields(logrus.Fields{"bookingRequest": bookingReq}).Error("Failed to marshal booking request")
		return nil, appError.NewInternal("Failed to book seats", appError.WithCause(err))
	}

	bookingIdStr := bookingReq.ID.String()

	// Attach business attributes in span
	span.SetAttributes(
		attribute.String("booking.id", bookingIdStr),
		attribute.Int("booking.user_id", int(bookingReq.UserID)),
	)

	// Run the script
	runScript := func() (interface{}, error) {
		return cm.CacheManager.RunScript(ctx, constants.BoundedKeyBookSeats, bookingScripts.BookingInitiateScript,
			scriptKeys,
			fingerprint+string(reqBytes), // idempotency value: fingerprint prefix + request JSON
			bookingIdStr,
			seatLockValue,
			idempotencyTTL,
			userLatestBookingsTTL,
			seatsLockTTLSec,
			reqBytes,    // user bookings hash value stays pure JSON
			fingerprint, // compared against the stored prefix on replay
		)
	}

	// Unpack the script result into (array, status)
	parseScriptResult := func(raw interface{}) ([]interface{}, string, *appError.Error) {
		arr, ok := raw.([]interface{})
		if !ok || len(arr) == 0 {
			cm.logger.WithFields(logrus.Fields{"raw": raw}).Error("Unexpected script result")
			return nil, "", appError.NewInternal("Failed to book seats")
		}
		status, ok := arr[0].(string)
		if !ok {
			cm.logger.WithFields(logrus.Fields{"raw": raw}).Error("Failed to unmarshal data")
			return nil, "", appError.NewInternal("Failed to book seats")
		}
		return arr, status, nil
	}

	raw, err := runScript()
	if err != nil {
		cm.logger.WithError(err).WithFields(logrus.Fields{"keys": scriptKeys}).Error("Failed to run script")
		return nil, appError.NewInternal("Failed to book seats", appError.WithCause(err))
	}

	arr, status, appErr := parseScriptResult(raw)
	if appErr != nil {
		return nil, appErr
	}

	// Seat info cache evicted/flushed
	// Warm it via the legacy read path (cache -> db fallback) and retry exactly once.
	if status == "PRICE_UNAVAILABLE" {
		cm.logger.WithFields(logrus.Fields{"event_id": bookingReq.EventID, "seat_ids": bookingReq.SeatIDs}).Warn("Seat info cache cold, warming via legacy path and retrying the script once")

		seats, appErr := cm.GetEventSeats(ctx, bookingReq.EventID, bookingReq.SeatIDs)
		if appErr != nil {
			return nil, appErr
		}
		if err := cm.warmSeatInfoKeys(ctx, seats); err != nil {
			cm.logger.WithError(err).WithFields(logrus.Fields{"event_id": bookingReq.EventID, "seat_ids": bookingReq.SeatIDs}).Error("Failed to warm seat info cache")
			return nil, appError.NewInternal("Failed to book seats", appError.WithCause(err))
		}

		raw, err = runScript()
		if err != nil {
			cm.logger.WithError(err).WithFields(logrus.Fields{"keys": scriptKeys}).Error("Failed to re-run script after price warm")
			return nil, appError.NewInternal("Failed to book seats", appError.WithCause(err))
		}
		arr, status, appErr = parseScriptResult(raw)
		if appErr != nil {
			return nil, appErr
		}
	}

	bookingResult = &BookingResult{}

	switch status {
	case "SUCCESS":
		if len(arr) < 3 {
			cm.logger.WithFields(logrus.Fields{"raw": arr}).Error("Booking script returned malformed SUCCESS result")
			return nil, appError.NewInternal("Failed to book seats")
		}
		bookingResult.Status = http.StatusOK
		bookingResult.ID = bookingReq.ID.String()
		raw := arr[1]
		jsonStr, ok := raw.(string)
		if !ok {
			cm.logger.WithFields(logrus.Fields{"raw": raw}).Error("Failed to unmarshal data")
			return nil, appError.NewInternal("Failed to book seats")
		}
		err = json.Unmarshal([]byte(jsonStr), &bookingResult.Data)
		if err != nil {
			cm.logger.WithError(err).WithFields(logrus.Fields{"raw": raw}).Error("Failed to unmarshal data")
			return nil, appError.NewInternal("Failed to book seats")
		}
		// Lua numbers come back as int64. Stamp the script-summed total BEFORE the
		// publish below, or the postgres projection would store a zero amount.
		totalAmount, ok := arr[2].(int64)
		if !ok {
			cm.logger.WithFields(logrus.Fields{"raw": arr[2]}).Error("Booking script returned non-integer total")
			return nil, appError.NewInternal("Failed to book seats")
		}
		bookingReq.TotalAmount = totalAmount
		bookingResult.Message = "Seats Booked"
	case "IDEMPOTENCY_HIT":
		bookingResult.Status = http.StatusOK
		raw := arr[1]
		jsonStr, ok := raw.(string)
		if !ok {
			cm.logger.WithFields(logrus.Fields{"raw": raw}).Error("Failed to unmarshal data")
			return nil, appError.NewInternal("Failed to process idempotency")
		}
		// arr[1] is the original booking request JSON stored by the Lua script
		cachedBookingReq := &models.BookingRequest{}
		err = json.Unmarshal([]byte(jsonStr), cachedBookingReq)
		if err != nil {
			cm.logger.WithError(err).WithFields(logrus.Fields{"raw": raw}).Error("Failed to unmarshal data")
			return nil, appError.NewInternal("Failed to process idempotency")
		}
		// Return the original booking ID, not the freshly generated one
		bookingResult.ID = cachedBookingReq.ID.String()
		bookingResult.Data = cachedBookingReq
		bookingResult.Message = "Idempotency Hit"
	case "IDEMPOTENCY_CONFLICT":
		return nil, appError.NewConflict("Idempotency-Key was already used with a different request")
	case "SEAT_ALREADY_LOCKED":
		bookingResult.Status = http.StatusConflict
		raw := arr[1]
		jsonStr, ok := raw.(string)
		if !ok {
			cm.logger.WithFields(logrus.Fields{"raw": raw}).Error("Failed to unmarshal data")
			return nil, appError.NewInternal("Failed to process seat conflict")
		}
		bookingResult.Data = jsonStr
		bookingResult.Message = fmt.Sprintf("Seat(s) Already Locked %s", jsonStr)
	case "SEAT_ALREADY_BOOKED":
		bookingResult.Status = http.StatusConflict
		raw := arr[1]
		jsonStr, ok := raw.(string)
		if !ok {
			cm.logger.WithFields(logrus.Fields{"raw": raw}).Error("Failed to unmarshal data")
			return nil, appError.NewInternal("Failed to process seat conflict")
		}
		bookingResult.Data = jsonStr
		bookingResult.Message = fmt.Sprintf("Seat(s) Already Booked %s", jsonStr)
	case "PRICE_UNAVAILABLE":
		// still cold right after the bounded warm+retry above
		cm.logger.WithFields(logrus.Fields{"raw": raw}).Error("Seat info cache still cold after warm retry")
		return nil, appError.NewInternal("Failed to book seats")
	default:
		cm.logger.WithFields(logrus.Fields{"status": status, "raw": raw}).Error("Unknown booking script status")
		return nil, appError.NewInternal("Failed to book seats")
	}

	// If not success, return error
	if bookingResult.Status != http.StatusOK {
		return bookingResult, appError.NewConflict(bookingResult.Message)
	}

	// Publish to RabbitMQ on success.
	// Returns as soon as the message is handed to the broker connection; the
	// broker confirm is awaited in the background. Only a definitive nack rolls
	// the booking back — a missing/late confirm may mean the event still stored.
	// Worst case if the event never lands at all: locks free up at their TTL.
	if status == "SUCCESS" {
		err := cm.EventBus.PublishBookingCreatedAsync(ctx, bookingReq, func() {
			cm.logger.WithFields(logrus.Fields{"booking_id": bookingIdStr}).Error("Booking created event nacked by broker, rolling back")

			rbCtx, rbCancel := context.WithTimeout(context.Background(), 5*time.Second)
			cm.rollbackBooking(rbCtx, scriptKeys, bookingIdStr, seatLockValue)
			rbCancel()
		})
		if err != nil {
			// The write itself failed (connection down) — rolled back synchronously,
			// same as before; the client can safely retry.
			cm.logger.WithError(err).WithFields(logrus.Fields{"booking_id": bookingIdStr}).Error("Failed to publish booking created event, rolling back")

			rbCtx, rbCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			cm.rollbackBooking(rbCtx, scriptKeys, bookingIdStr, seatLockValue)
			rbCancel()

			return nil, appError.NewInternal("Failed to place booking")
		}
		cm.logger.WithFields(logrus.Fields{"booking_id": bookingIdStr}).Info("Booking created event published")
	}

	return bookingResult, nil
}

// rollbackBooking undoes BookSeats: seat locks, idempotency key and the user
// bookings hash entry. If it fails too, the lock TTL frees the seats later.
func (cm *CacheStore) rollbackBooking(ctx context.Context, scriptKeys []string, bookingID string, seatLockValue string) {
	raw, err := cm.CacheManager.RunScript(ctx, constants.BoundedKeyBookSeats, bookingScripts.BookingRollbackScript, scriptKeys, bookingID, seatLockValue)
	if err != nil {
		cm.logger.WithError(err).WithFields(logrus.Fields{"booking_id": bookingID}).Error("Booking rollback failed, seats free up when the lock TTL expires")
		return
	}
	cm.logger.WithFields(logrus.Fields{"booking_id": bookingID, "result": raw}).Warn("Booking rolled back after publish failure")
}
