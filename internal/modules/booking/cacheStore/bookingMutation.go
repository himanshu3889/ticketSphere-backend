package bookingCacheStore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	// redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
	"github.com/google/uuid"
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

// Book Seats using Redis and RabbitMQ
func (cm *CacheStore) BookSeats(ctx context.Context, rawIdempotencyKey string, req *models.BookingRequest) (bookingResult *BookingResult, appErr *appError.Error) {
	// Start a otel span for the business logic
	ctx, span := cm.Tracer.Start(ctx, "Seat.Booking.Execute")
	defer utils.EndSpanAppErr(span, appErr)

	const seatsMaxCount = 10

	// Quick validations
	if _, err := uuid.Parse(rawIdempotencyKey); err != nil {
		return nil, appError.NewBadRequest("Idempotency-Key must be a valid UUID")
	}
	if req.EventID <= 0 {
		return nil, appError.NewBadRequest("Invalid event id")
	}
	if req.UserID <= 0 {
		return nil, appError.NewBadRequest("Invalid user id")
	}
	if len(req.SeatIDs) == 0 {
		return nil, appError.NewBadRequest("No seat ids provided")
	}
	if len(req.SeatIDs) > seatsMaxCount {
		return nil, appError.NewBadRequest(fmt.Sprintf("Limit of %d seats per booking exceeded", seatsMaxCount))
	}

	// Get total amount
	totalAmount, appErr := cm.GetEventSeatsTotalPrice(ctx, req.EventID, req.SeatIDs)
	if appErr != nil {
		return nil, appErr
	}

	// Build request
	seatsLockTTL := int((15 * time.Minute).Seconds())

	req.ID = utils.GenerateSnowflakeID(cm.Store.SnowflakeNode)
	req.Status = models.BookingStatusPending
	req.CreatedAt = time.Now()
	req.ExpiresAt = time.Now().Add(time.Duration(seatsLockTTL))
	req.TotalAmount = totalAmount

	// Key setup
	safeIdempotencyKey := fmt.Sprintf("booking:idempotency:%s:%s", req.UserID, rawIdempotencyKey)
	userBookingsKey := fmt.Sprintf("user:%d:bookings:latest", req.UserID)
	idempotencyTTL := int((5 * time.Minute).Seconds())
	userLatestBookingsTTL := int((60 * time.Minute).Seconds())

	// Add keys
	keys := []string{safeIdempotencyKey, userBookingsKey}
	for _, seat := range req.SeatIDs {
		keys = append(keys, fmt.Sprintf("seat:%d", seat))
	}

	// Seat lock value
	seatLockValue := fmt.Sprintf("%d:%d", req.UserID, req.ID)
	// Marshal booking request
	reqBytes, err := json.Marshal(req)
	if err != nil {
		cm.logger.WithError(err).WithFields(logrus.Fields{"bookingRequest": req}).Error("Failed to marshal booking request")
		return nil, appError.NewInternal("Failed to book seats")
	}

	bookingIdStr := req.ID.String()

	// Attach business attributes in span
	span.SetAttributes(
		attribute.String("booking.id", bookingIdStr),
		attribute.Int("booking.user_id", int(req.UserID)),
	)

	// Run the script
	raw, err := cm.CacheManager.RunScript(ctx, "book_seats", bookingScripts.BookingScript, keys,
		reqBytes,
		bookingIdStr,
		seatLockValue,
		idempotencyTTL,
		userLatestBookingsTTL,
		seatsLockTTL,
		reqBytes,
	)
	if err != nil {
		cm.logger.WithError(err).WithFields(logrus.Fields{"keys": keys}).Error("Failed to run script")
		return nil, appError.NewInternal("Failed to book seats")
	}

	// Check script result
	arr, ok := raw.([]interface{})
	if !ok || len(arr) == 0 {
		cm.logger.WithFields(logrus.Fields{"raw": raw}).Error("Unexpected script result")
		return nil, appError.NewInternal("Failed to book seats")
	}

	// Check status
	status, _ := arr[0].(int64)
	bookingResult = &BookingResult{Status: int(status), ID: req.ID.String()}

	switch status {
	case 0:
		bookingResult.Status = http.StatusOK
		bookingResult.ID = req.ID.String()
		raw := arr[2]
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
		bookingResult.Message = "Seats Booked"
	case 1:
		bookingResult.Status = http.StatusOK
		bookingResult.ID = req.ID.String()
		raw := arr[2]
		jsonStr, ok := raw.(string)
		if !ok {
			cm.logger.WithFields(logrus.Fields{"raw": raw}).Error("Failed to unmarshal data")
			return nil, appError.NewInternal("Failed to process idempotency")
		}
		err = json.Unmarshal([]byte(jsonStr), &bookingResult.Data)
		if err != nil {
			cm.logger.WithError(err).WithFields(logrus.Fields{"raw": raw}).Error("Failed to unmarshal data")
			return nil, appError.NewInternal("Failed to process idempotency")
		}
		bookingResult.Message = "Idempotency Hit"
	case 2:
		bookingResult.Status = http.StatusConflict
		raw := arr[2]
		jsonStr, ok := raw.(string)
		if !ok {
			cm.logger.WithFields(logrus.Fields{"raw": raw}).Error("Failed to unmarshal data")
			return nil, appError.NewInternal("Failed to process seat conflict")
		}
		bookingResult.Data = jsonStr
		bookingResult.Message = fmt.Sprintf("Seat(s) Already Locked %s", jsonStr)
	}

	// If not success, return error
	if bookingResult.Status != http.StatusOK {
		return bookingResult, appError.NewConflict(bookingResult.Message)
	}

	// Publish to RabbitMQ on success (status == 0)
	if status == 0 {
		err := cm.EventBus.PublishBookingCreated(ctx, req)
		if err != nil {
			cm.logger.WithError(err).WithFields(logrus.Fields{"req": req}).Error("Failed to publish booking created event")
			return bookingResult, appError.NewInternal("Failed to place booking")
		}
		fmt.Println("Booking created event published")
	}

	return bookingResult, nil
}
