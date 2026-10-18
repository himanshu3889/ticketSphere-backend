package handlers

import (
	"context"

	"github.com/himanshu3889/ticketSphere-backend/base/structures"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/paymentGateway"
	"github.com/sirupsen/logrus"
)

// Handle the payment.webhook.refunded event: mark the booking as refunded and release the confirmed seats
func HandlePaymentWebhookRefunded(ctx context.Context, gatewayPaymentClient *paymentGateway.Client, gatewayPayment *structures.PaymentGatewayWebhookPayload, cacheStore *paymentCacheStore.CacheStore, logger *logrus.Logger) error {
	// TODO: mark the booking refunded and release the confirmed seats back to inventory
	// via cacheStore/domain methods
	return nil
}
