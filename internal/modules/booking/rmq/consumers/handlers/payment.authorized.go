package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/base/models"
	bookingCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/cacheStore"
	"github.com/sirupsen/logrus"
)

// paymentAuthorizedPayload mirrors the payment.authorized event contract published by the payment module.
// TODO: move to a shared events/contracts package once one exists.
type paymentAuthorizedPayload struct {
	BookingRequest *models.BookingRequest    `json:"booking_request"`
	Payment        *authorizedGatewayPayment `json:"payment"`
}

// to validate the event contract before confirming a booking.
type authorizedGatewayPayment struct {
	ID       string            `json:"id"`
	Amount   int64             `json:"amount"`
	Metadata map[string]string `json:"metadata"`
}

// confirms the booking after the payment module has captured the funds,
// then publishes booking.confirmed so downstream consumers (projections, notifications) can react.
func HandlePaymentAuthorized(ctx context.Context, body []byte, cacheStore *bookingCacheStore.CacheStore, logger *logrus.Logger) error {
	var payload paymentAuthorizedPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return err // poison message -> DLQ
	}
	if payload.BookingRequest == nil {
		return errors.New("payment.authorized event is missing booking_request")
	}
	if payload.Payment == nil {
		return errors.New("payment.authorized event is missing payment")
	}

	// contract checks — booking.confirmed is never published on mismatched money data
	if metaBookingID := payload.Payment.Metadata["booking_id"]; metaBookingID != payload.BookingRequest.ID.String() {
		return fmt.Errorf("payment.authorized booking mismatch: metadata booking_id %q vs booking %d", metaBookingID, payload.BookingRequest.ID)
	}
	if payload.Payment.Amount != payload.BookingRequest.TotalAmount {
		return fmt.Errorf("payment.authorized amount mismatch: gateway %d vs booking %d", payload.Payment.Amount, payload.BookingRequest.TotalAmount)
	}

	// The routing key defines the transition
	confirmedAt := time.Now().UTC()
	payload.BookingRequest.Status = models.BookingStatusConfirmed
	payload.BookingRequest.ConfirmedAt = &confirmedAt

	// this inbox only translates payment.authorized -> booking.confirmed.
	if err := cacheStore.EventBus.PublishBookingConfirmed(ctx, payload.BookingRequest); err != nil {
		return fmt.Errorf("failed to publish booking.confirmed: %w", err)
	}
	return nil
}
