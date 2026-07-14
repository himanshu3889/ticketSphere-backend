package bookingRMQ

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	baseRabbitMQ "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/rabbitmq"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
)

// EventBus struct
type EventBus struct {
	bookingProducer *baseRabbitMQ.Producer
}

// Global variables to hold the singleton instance and the sync.Once controller
var (
	bookingEventBus *EventBus
	once            sync.Once
	initErr         error
)

// GetEventBus returns the singleton event bus instance.
// It initializes the instance on the first call.
func GetEventBus(conn *amqp091.Connection, logger *logrus.Logger, traceProvider trace.TracerProvider) *EventBus {

	once.Do(func() {
		publisherTracer := traceProvider.Tracer("ticketSphere.booking.publisher")
		bookingProducer, err := baseRabbitMQ.NewProducer(conn, string(ExchangeBooking), string(ExchangeKindTopic), true, false, false, false, nil, logger, publisherTracer)
		if err != nil {
			initErr = err
			logger.WithError(err).Error("Failed to create booking producer")
			return
		}
		bookingEventBus = &EventBus{bookingProducer: bookingProducer}
	})

	return bookingEventBus
}

// publishes a booking created event
func (b *EventBus) PublishBookingCreated(ctx context.Context, bookingRequest *models.BookingRequest) error {
	body, _ := json.Marshal(bookingRequest)
	return b.bookingProducer.Publish(ctx, string(RoutingKeyBookingCreated), body, amqp091.Table{})
}
