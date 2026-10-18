package handlers

import (
	"context"

	"github.com/himanshu3889/ticketSphere-backend/base/structures"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/paymentGateway"
	"github.com/sirupsen/logrus"
)

// Handle the payment.webhook.pending event: record the payment as pending while awaiting authorization
func HandlePaymentWebhookPending(ctx context.Context, gatewayPaymentClient *paymentGateway.Client, gatewayPayment *structures.PaymentGatewayWebhookPayload, cacheStore *paymentCacheStore.CacheStore, logger *logrus.Logger) error {
	// TODO: persist/track the pending payment state and notify the user
	// via cacheStore/domain methods
	return nil
}
