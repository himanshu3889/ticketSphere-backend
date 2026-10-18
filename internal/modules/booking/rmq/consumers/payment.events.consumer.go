package consumers

import (
	"context"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	baseRabbitMQ "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/rabbitmq"
	bookingCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/cacheStore"
	bookingRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/rmq"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/rmq/consumers/handlers"
	paymentRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/rmq"
)

// StartPaymentEventsConsumer runs the worker pool consuming booking.payment.events.queue
// (payment domain events the booking module reacts to — the booking-payment saga inbox).
//
// TODO: the payment exchange/routing-key contract is imported from the payment module;
// move both to a shared events/contracts package once one exists.
func StartPaymentEventsConsumer(ctx context.Context, conn *amqp091.Connection, logger *logrus.Logger, cacheStore *bookingCacheStore.CacheStore, workerCount int, tracer trace.Tracer) {
	// TODO: make it configurable in yml file
	cfg := baseRabbitMQ.ConsumerConfig{
		ExchangeName:  string(paymentRMQ.ExchangePayment),
		RoutingKey:    string(paymentRMQ.RoutingKeyPaymentAuthorized), // TODO: FIX THIS KEY
		QueueName:     string(bookingRMQ.QueueBookingPaymentEvents),
		DLXExchange:   string(bookingRMQ.ExchangeDLX),
		DLXRoutingKey: string(bookingRMQ.BindingKeyBookingAll),
		PrefetchCount: 1, // each worker holds only 1 unacked message
	}

	handler := func(ctx context.Context, msg amqp091.Delivery) error {
		switch msg.RoutingKey {
		case string(paymentRMQ.RoutingKeyPaymentAuthorized):
			return handlers.HandlePaymentAuthorized(ctx, msg.Body, cacheStore, logger)
		default:
			// This queue only receives payment.authorized; ack anything unexpected
			return nil
		}
	}

	runConsumerPool(ctx, conn, cfg, string(bookingRMQ.QueueBookingPaymentEvents), workerCount, handler, logger, tracer)
}
