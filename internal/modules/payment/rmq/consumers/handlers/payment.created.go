package handlers

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/himanshu3889/ticketSphere-backend/base/models"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	"github.com/sirupsen/logrus"
)

// decodePaymentCreated decodes the payment.created event body published by PublishPaymentInitiated.
func decodePaymentCreated(body []byte) (*models.Payment, error) {
	var payment models.Payment
	if err := json.Unmarshal(body, &payment); err != nil {
		return nil, err // poison message -> DLQ
	}
	return &payment, nil
}

// validatePaymentProjection rejects poison events that can never be projected.
func validatePaymentProjection(payment *models.Payment) error {
	switch {
	case payment.ID <= 0:
		return errors.New("payment event has invalid id")
	case payment.BookingID <= 0:
		return errors.New("payment event has invalid booking_id")
	case payment.Amount <= 0:
		return errors.New("payment event has invalid amount")
	}
	return nil
}

// projects the payment.created event into postgres
func HandlePaymentCreatedPostgres(ctx context.Context, body []byte, cacheStore *paymentCacheStore.CacheStore, logger *logrus.Logger) error {
	payment, err := decodePaymentCreated(body)
	if err != nil {
		return err
	}
	if err := validatePaymentProjection(payment); err != nil {
		return err // poison message -> DLQ
	}
	if payment.Provider == "" {
		logger.WithField("payment_id", payment.ID.String()).Warn("payment.created event has empty provider, defaulting to INTERNAL")
		payment.Provider = models.PaymentProviderInternal
	}
	return cacheStore.Store.UpsertPaymentCreated(ctx, payment)
}
