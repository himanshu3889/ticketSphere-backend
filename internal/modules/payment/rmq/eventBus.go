package paymentRMQ

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	baseRabbitMQ "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/rabbitmq"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/structures"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/paymentGateway"
)

// EventBus struct
type EventBus struct {
	paymentProducer *baseRabbitMQ.Producer
}

// Global variables to hold the singleton instance and the sync.Once controller
var (
	paymentEventBus *EventBus
	once            sync.Once
	initErr         error
)

// GetEventBus returns the singleton event bus instance.
// It initializes the instance on the first call.
func GetEventBus(conn *amqp091.Connection, logger *logrus.Logger, traceProvider trace.TracerProvider) *EventBus {

	once.Do(func() {
		publisherTracer := traceProvider.Tracer(constants.TracerPaymentPublisher)
		paymentProducer, err := baseRabbitMQ.NewProducer(conn, string(ExchangePayment), string(ExchangeKindTopic), true, false, false, false, nil, logger, publisherTracer)
		if err != nil {
			initErr = err
			logger.WithError(err).Error("Failed to create payment producer")
			return
		}
		paymentEventBus = &EventBus{paymentProducer: paymentProducer}
	})

	return paymentEventBus
}

// publish a payment gateway event
func (b *EventBus) PublishPaymentWebhookEvent(ctx context.Context, payment *structures.PaymentGatewayWebhookPayload) error {
	body, _ := json.Marshal(payment)
	// Event ID header so consumers can deduplicate
	headers := amqp091.Table{"x-event-id": payment.ID}
	// Route by status: payment.webhook.authorized / .captured / .failed
	routingKey := fmt.Sprintf("%s.%s", string(RoutingKeyPaymentWebhook), payment.Status)
	return b.paymentProducer.Publish(ctx, routingKey, body, headers)
}

// publishes a payment initiated event
func (b *EventBus) PublishPaymentInitiated(ctx context.Context, payment *models.Payment) error {
	body, _ := json.Marshal(payment)
	return b.paymentProducer.Publish(ctx, string(RoutingKeyPaymentCreated), body, amqp091.Table{})
}

// publish a payment authorized event
func (b *EventBus) PublishPaymentAuthorized(ctx context.Context, bookingRequest *models.BookingRequest, payment *paymentGateway.Payment) error {
	payload := struct {
		BookingRequest *models.BookingRequest  `json:"booking_request"`
		Payment        *paymentGateway.Payment `json:"payment"`
	}{
		BookingRequest: bookingRequest,
		Payment:        payment,
	}
	body, _ := json.Marshal(payload)
	return b.paymentProducer.Publish(ctx, string(RoutingKeyPaymentAuthorized), body, amqp091.Table{})
}
