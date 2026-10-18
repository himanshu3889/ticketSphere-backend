package consumers

import (
	"context"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	baseRabbitMQ "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/rabbitmq"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	paymentRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/rmq"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/rmq/consumers/handlers"
)

// StartPostgresConsumer runs the worker pool consuming payment.postgres.queue
// (payment domain events projected into postgres).
func StartPostgresConsumer(conn *amqp091.Connection, logger *logrus.Logger, cacheStore *paymentCacheStore.CacheStore, workerCount int, tracer trace.Tracer) {
	// TODO: make it configurable in yml file
	cfg := baseRabbitMQ.ConsumerConfig{
		ExchangeName:  string(paymentRMQ.ExchangePayment),
		RoutingKey:    string(paymentRMQ.BindingKeyPaymentDomain),
		QueueName:     string(paymentRMQ.QueuePaymentPostgres),
		DLXExchange:   string(paymentRMQ.ExchangePaymentDLX),
		DLXRoutingKey: string(paymentRMQ.BindingKeyPaymentAll),
		PrefetchCount: 1, // each worker holds only 1 unacked message
	}

	handler := func(ctx context.Context, msg amqp091.Delivery) error {
		switch msg.RoutingKey {
		case string(paymentRMQ.RoutingKeyPaymentCreated):
			return handlers.HandlePaymentCreatedPostgres(ctx, msg.Body, cacheStore, logger)
		case string(paymentRMQ.RoutingKeyPaymentAuthorized):
			return handlers.HandlePaymentAuthorizedPostgres(ctx, msg.Body, cacheStore, logger)
		default:
			logger.WithField("routing_key", msg.RoutingKey).Warn("no postgres handler for payment event, acking")
			return nil
		}
	}

	runConsumerPool(conn, cfg, string(paymentRMQ.QueuePaymentPostgres), workerCount, handler, logger, tracer)
}
