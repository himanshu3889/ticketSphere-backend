package handlers

import (
	"context"

	"github.com/himanshu3889/ticketSphere-backend/base/structures"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/paymentGateway"
	"github.com/sirupsen/logrus"
)

// Handle the payment.webhook.partially_refunded event: record the partial refund against the booking
func HandlePaymentWebhookPartiallyRefunded(ctx context.Context, gatewayPaymentClient *paymentGateway.Client, gatewayPayment *structures.PaymentGatewayWebhookPayload, cacheStore *paymentCacheStore.CacheStore, logger *logrus.Logger) error {
	// TODO: record the partial refund amount against the booking and adjust seats if applicable
	// via cacheStore/domain methods
	return nil
}
