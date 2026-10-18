package consumers

import (
	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	baseRabbitMQ "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/rabbitmq"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	paymentRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/rmq"
)

// StartWebhookConsumer runs the worker pool consuming payment.webhook.queue (raw payment gateway webhook events).
func StartWebhookConsumer(conn *amqp091.Connection, logger *logrus.Logger, cacheStore *paymentCacheStore.CacheStore, workerCount int, tracer trace.Tracer) {
	// TODO: make it configurable in yml file
	cfg := baseRabbitMQ.ConsumerConfig{
		ExchangeName:  string(paymentRMQ.ExchangePayment),
		RoutingKey:    string(paymentRMQ.BindingKeyPaymentWebhook),
		QueueName:     string(paymentRMQ.QueuePaymentWebhook),
		DLXExchange:   string(paymentRMQ.ExchangePaymentDLX),
		DLXRoutingKey: string(paymentRMQ.BindingKeyPaymentAll),
		PrefetchCount: 1, // each worker holds only 1 unacked message
	}

	runConsumerPool(conn, cfg, string(paymentRMQ.QueuePaymentWebhook), workerCount, newPaymentWebhookHandler(cacheStore, logger), logger, tracer)
}
