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
	// Use # to match all sub-routing keys under booking.*
	BindingKeyBookingAll RMQConstants = "booking.#"
)

// Queue names
const (
	QueueBookingPostgres RMQConstants = "booking.postgres.queue"
	QueueBookingMongoDB  RMQConstants = "booking.mongodb.queue"
	QueueBookingRedis    RMQConstants = "booking.redis.queue"
)
