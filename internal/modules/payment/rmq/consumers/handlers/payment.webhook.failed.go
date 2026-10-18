package handlers

import (
	"context"

	"github.com/himanshu3889/ticketSphere-backend/base/structures"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/paymentGateway"
	"github.com/sirupsen/logrus"
)

// Handle the payment.webhook.failed event: fail the booking and release the held seats
func HandlePaymentWebhookFailed(ctx context.Context, gatewayPaymentClient *paymentGateway.Client, gatewayPayment *structures.PaymentGatewayWebhookPayload, cacheStore *paymentCacheStore.CacheStore, logger *logrus.Logger) error {
	// TODO: fail the booking and release the held seats back to inventory
	// via cacheStore/domain methods
	return nil
}
