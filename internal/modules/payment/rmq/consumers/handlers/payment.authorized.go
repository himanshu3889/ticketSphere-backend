package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/paymentGateway"
	"github.com/sirupsen/logrus"
)

// paymentAuthorizedEvent mirrors the payment.authorized event contract published by PublishPaymentAuthorized.
// TODO: move to a shared events/contracts package once one exists.
type paymentAuthorizedEvent struct {
	BookingRequest *models.BookingRequest  `json:"booking_request"`
	Payment        *paymentGateway.Payment `json:"payment"`
}

// decodes the payment.authorized event body.
func decodePaymentAuthorized(body []byte) (*paymentAuthorizedEvent, error) {
	var event paymentAuthorizedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err // poison message -> DLQ
	}
	if event.BookingRequest == nil || event.Payment == nil {
		return nil, errors.New("payment.authorized event must carry booking_request and payment")
	}
	return &event, nil
}

// projects payment.authorized into postgres:
// PENDING -> SUCCESS with processed_at.
func HandlePaymentAuthorizedPostgres(ctx context.Context, body []byte, cacheStore *paymentCacheStore.CacheStore, logger *logrus.Logger) error {
	event, err := decodePaymentAuthorized(body)
	if err != nil {
		return err
	}

	gatewayPayment := event.Payment
	bookingRequest := event.BookingRequest

	// our payment id is the gateway order reference, set at initiation
	paymentID, err := utils.ValidSnowflakeID(gatewayPayment.Reference)
	if err != nil {
		return fmt.Errorf("payment.authorized has invalid payment reference %q: %w", gatewayPayment.Reference, err)
	}
	if metaPaymentID := gatewayPayment.Metadata["payment_id"]; metaPaymentID != gatewayPayment.Reference {
		return fmt.Errorf("payment.authorized payment mismatch: metadata payment_id %q vs reference %q", metaPaymentID, gatewayPayment.Reference)
	}
	if metaBookingID := gatewayPayment.Metadata["booking_id"]; metaBookingID != bookingRequest.ID.String() {
		return fmt.Errorf("payment.authorized booking mismatch: metadata booking_id %q vs booking %d", metaBookingID, bookingRequest.ID)
	}
	if gatewayPayment.Amount != bookingRequest.TotalAmount {
		return fmt.Errorf("payment.authorized amount mismatch: gateway %d vs booking %d", gatewayPayment.Amount, bookingRequest.TotalAmount)
	}

	processedAt := time.Now().UTC()
	if gatewayPayment.CompletedAt != nil {
		processedAt = *gatewayPayment.CompletedAt
	}
	createdAt := gatewayPayment.CreatedAt
	if createdAt.IsZero() {
		createdAt = processedAt
	}

	payment := &models.Payment{
		ID:                   paymentID,
		BookingID:            bookingRequest.ID,
		Amount:               bookingRequest.TotalAmount,
		Provider:             models.PaymentProviderInternal,
		GatewayTransactionID: &gatewayPayment.ID,
		CreatedAt:            createdAt,
	}

	transitioned, err := cacheStore.Store.ApplyPaymentAuthorized(ctx, payment, processedAt)
	if err != nil {
		return err
	}
	if !transitioned {
		logger.WithField("payment_id", paymentID.String()).Warn("payment.authorized projection skipped: payment not PENDING (replay or late event)")
	}
	return nil
}
