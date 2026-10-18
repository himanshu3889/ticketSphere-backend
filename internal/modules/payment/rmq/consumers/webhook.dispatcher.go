package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"

	"github.com/himanshu3889/ticketSphere-backend/base/structures"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/paymentGateway"
	paymentRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/rmq"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/rmq/consumers/handlers"
)

// newPaymentWebhookHandler decodes a gateway webhook delivery and dispatches it to the matching
// handler. The routing key (payment.webhook.<status>) is the source of truth for dispatch and is
// validated against the payload status.
func newPaymentWebhookHandler(cacheStore *paymentCacheStore.CacheStore, logger *logrus.Logger) func(context.Context, amqp091.Delivery) error {
	return func(ctx context.Context, msg amqp091.Delivery) error {
		var payment structures.PaymentGatewayWebhookPayload
		if err := json.Unmarshal(msg.Body, &payment); err != nil {
			return err // poison message -> nack -> DLQ
		}
		if payment.ID == "" {
			return errors.New("payment webhook event is missing id") // cannot dedupe -> poison -> DLQ
		}
		if expected := fmt.Sprintf("%s.%s", paymentRMQ.RoutingKeyPaymentWebhook, payment.Status); msg.RoutingKey != expected {
			return fmt.Errorf("webhook routing key %q does not match payload status %q", msg.RoutingKey, payment.Status)
		}

		gatewayClient := cacheStore.PaymentGateway

		switch msg.RoutingKey {
		case webhookRoutingKey(paymentGateway.PaymentStatusPending):
			return handlers.HandlePaymentWebhookPending(ctx, gatewayClient, &payment, cacheStore, logger)
		case webhookRoutingKey(paymentGateway.PaymentStatusAuthorized):
			return handlers.HandlePaymentWebhookAuthorized(ctx, gatewayClient, &payment, cacheStore, logger)
		case webhookRoutingKey(paymentGateway.PaymentStatusCaptured):
			return handlers.HandlePaymentWebhookCaptured(ctx, gatewayClient, &payment, cacheStore, logger)
		case webhookRoutingKey(paymentGateway.PaymentStatusCancelled):
			return handlers.HandlePaymentWebhookCancelled(ctx, gatewayClient, &payment, cacheStore, logger)
		case webhookRoutingKey(paymentGateway.PaymentStatusRefunded):
			return handlers.HandlePaymentWebhookRefunded(ctx, gatewayClient, &payment, cacheStore, logger)
		case webhookRoutingKey(paymentGateway.PaymentStatusPartiallyRefunded):
			return handlers.HandlePaymentWebhookPartiallyRefunded(ctx, gatewayClient, &payment, cacheStore, logger)
		case webhookRoutingKey(paymentGateway.PaymentStatusFailed):
			return handlers.HandlePaymentWebhookFailed(ctx, gatewayClient, &payment, cacheStore, logger)
		default:
			logger.WithField("routing_key", msg.RoutingKey).Warn("no handler for payment webhook event, acking")
			return nil
		}
	}
}

// webhookRoutingKey builds the payment.webhook.<status> routing key for a gateway payment status.
func webhookRoutingKey(status paymentGateway.PaymentStatus) string {
	return fmt.Sprintf("%s.%s", paymentRMQ.RoutingKeyPaymentWebhook, status)
}
