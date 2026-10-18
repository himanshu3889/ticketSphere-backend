package bookingRMQ

type RMQConstants string

const (
	// Exchanges
	ExchangeBooking RMQConstants = "booking.events"
	ExchangeDLX     RMQConstants = "booking.dlx"

	// Exchange type
	ExchangeKindTopic RMQConstants = "topic"

	// Routing keys (what producers publish with)
	RoutingKeyBookingCreated   RMQConstants = "booking.created"
	RoutingKeyBookingConfirmed RMQConstants = "booking.confirmed"
	RoutingKeyBookingFailed    RMQConstants = "booking.failed"

	// Queue binding patterns (what consumers listen to)
	BindingKeyBookingAll RMQConstants = "booking.#"
)

// Queue names
const (
	QueueBookingPostgres      RMQConstants = "booking.postgres.queue"
	QueueBookingMongoDB       RMQConstants = "booking.mongodb.queue"
	QueueBookingPaymentEvents RMQConstants = "booking.payment.events.queue"
	QueueBookingDLQ           RMQConstants = "booking.dlq"
)
