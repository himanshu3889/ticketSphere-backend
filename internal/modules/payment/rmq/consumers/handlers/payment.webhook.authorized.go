package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
	"github.com/himanshu3889/ticketSphere-backend/base/keys"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/structures"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	paymentScripts "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/lib/redis"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/paymentGateway"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
)

// Handle the payment.webhook.authorized event: capture the payment and confirm the booking
func HandlePaymentWebhookAuthorized(ctx context.Context, gatewayPaymentClient *paymentGateway.Client, gatewayPayment *structures.PaymentGatewayWebhookPayload, cacheStore *paymentCacheStore.CacheStore, logger *logrus.Logger) (err error) {
	fmt.Println("Payment authorized, confirming booking", gatewayPayment.PaymentID)

	// Start a otel span for the business logic
	ctx, span := cacheStore.Tracer.Start(ctx, constants.SpanPaymentAuthorized)
	defer utils.EndSpan(span, err)

	gatewayPaymentIdStr := gatewayPayment.PaymentID

	userIdStr, ok := gatewayPayment.Metadata["user_id"]
	if !ok || userIdStr == "" {
		return appError.NewBadRequest(fmt.Sprintf("Payment webhook %s missing user_id metadata", gatewayPaymentIdStr))
	}

	userId, err := utils.ValidSnowflakeID(userIdStr)
	if err != nil {
		return appError.NewBadRequest(fmt.Sprintf("Payment webhook %s has invalid user_id %q", gatewayPaymentIdStr, userIdStr))
	}

	bookingIdStr, ok := gatewayPayment.Metadata["booking_id"]
	if !ok || bookingIdStr == "" {
		return appError.NewBadRequest(fmt.Sprintf("Payment webhook %s missing booking_id metadata", gatewayPaymentIdStr))
	}

	_, err = utils.ValidSnowflakeID(bookingIdStr)
	if err != nil {
		return appError.NewBadRequest(fmt.Sprintf("Payment webhook %s has invalid booking_id %q", gatewayPaymentIdStr, bookingIdStr))
	}

	userBookingsKey := keys.UserBookingsKey(userId)

	// Try to deduct the amount from payment gateway
	// Check if user booking is PENDING or not
	var bookingRequest *models.BookingRequest
	state, err := cacheStore.CacheManager.HGetInto(ctx, "", userBookingsKey, bookingIdStr, nil, nil, &bookingRequest)
	if err != nil {
		return appError.NewInternal(fmt.Sprintf("Failed to load booking %s (user %s) from cache: %v", bookingIdStr, userIdStr, err))
	}
	if state != redisDatabase.CacheHit {
		return appError.NewBadRequest(fmt.Sprintf("Booking %s not found for user %s (cache state: %s, key: %s)", bookingIdStr, userIdStr, state, userBookingsKey))
	}

	// IF booking is pending then and at least one minute left for it to expire
	if bookingRequest.Status != models.BookingStatusPending {
		return appError.NewBadRequest(fmt.Sprintf("Booking %s is not pending (current status: %s)", bookingIdStr, bookingRequest.Status))
	}

	if time.Since(bookingRequest.ExpiresAt) > time.Minute {
		return appError.NewBadRequest(fmt.Sprintf("Booking %s expired at %s", bookingIdStr, bookingRequest.ExpiresAt.Format(time.RFC3339)))
	}

	// CALL THE GATEWAY to deduct the amount and if success then publish the event to queue immediately
	// Attach business attributes in span
	span.SetAttributes(
		attribute.String("booking.id", bookingIdStr),
		attribute.String("payment.id", gatewayPaymentIdStr),
	)

	capturePaymentRequest := paymentGateway.CaptureRequest{}
	capturePaymentIdempotencyKey := fmt.Sprintf("capture:%s:%s", gatewayPaymentIdStr, bookingIdStr) // TODO: Need to put it somwhere
	gatewayPaymentResponse, err := gatewayPaymentClient.Capture(ctx, gatewayPaymentIdStr, capturePaymentRequest, capturePaymentIdempotencyKey)
	if err != nil {
		return appError.NewInternal(fmt.Sprintf("Failed to capture payment %s: %v", gatewayPaymentIdStr, err))
	}

	if gatewayPaymentResponse.Status != paymentGateway.PaymentStatusCaptured {
		return appError.NewBadRequest(fmt.Sprintf("Payment %s not captured (current status: %s)", gatewayPaymentIdStr, gatewayPaymentResponse.Status))
	}

	// Publish payment authorized event immediately
	err = cacheStore.EventBus.PublishPaymentAuthorized(ctx, bookingRequest, gatewayPaymentResponse)
	if err != nil {
		return appError.NewInternal(fmt.Sprintf("Failed to publish payment authorized event: %v", err))
	}

	scriptKeys := []string{userBookingsKey} // keys

	confirmedAt := time.Now().UTC()
	confirmedAtStr := confirmedAt.Format(time.RFC3339)

	userBookingFieldTTL := int(keys.UserLatestBookingsTTL.Seconds())

	// Run the payment authorize script to confirm the booking in cache immediately, queue also will mark things
	authorizeBookingRawResult, err := cacheStore.CacheManager.RunScript(ctx, constants.BoundedKeyPaymentAuthorizeBookingGet, paymentScripts.PaymentAuthorizeScript,
		scriptKeys,
		bookingIdStr,
		confirmedAtStr,
		userBookingFieldTTL,
	)
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{"keys": scriptKeys}).Error("Failed to run payment authorize script")
		return appError.NewInternal(fmt.Sprintf("Failed to confirm booking %s in cache: %v", bookingIdStr, err))
	}

	// Check script result
	arr, ok := authorizeBookingRawResult.([]interface{})
	if !ok || len(arr) == 0 {
		logger.WithFields(logrus.Fields{"raw": authorizeBookingRawResult}).Error("Unexpected payment authorize script result")
		return appError.NewInternal(fmt.Sprintf("Unexpected authorize script result for booking %s", bookingIdStr))
	}

	// Check status
	status, ok := arr[0].(string)
	if !ok {
		logger.WithFields(logrus.Fields{"raw": authorizeBookingRawResult}).Error("Authorize script status is not a string")
		return appError.NewInternal(fmt.Sprintf("Unexpected authorize script status for booking %s", bookingIdStr))
	}

	switch status {
	case "NOT_FOUND":
		return appError.NewBadRequest(fmt.Sprintf("Booking %s not found in user %s bookings cache", bookingIdStr, userIdStr))
	case "BOOKING_NOT_PENDING":
		return appError.NewBadRequest(fmt.Sprintf("Booking %s is not pending", bookingIdStr))
	case "SUCCESS":
		return nil
	default:
		return appError.NewInternal(fmt.Sprintf("Unknown authorize script status %q for booking %s", status, bookingIdStr))
	}
}
