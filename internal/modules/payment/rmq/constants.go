package paymentRMQ

type PaymentRMQConstants string

const (
	// Exchanges
	ExchangePayment    PaymentRMQConstants = "payment.events"
	ExchangePaymentDLX PaymentRMQConstants = "payment.dlx"
	ExchangeBooking    PaymentRMQConstants = "booking.events"
	ExchangeBookingDLX PaymentRMQConstants = "booking.dlx"

	// Exchange type
	ExchangeKindTopic PaymentRMQConstants = "topic"

	// Routing keys (what producers publish with)
	RoutingKeyPaymentCreated    PaymentRMQConstants = "payment.created"
	RoutingKeyPaymentAuthorized PaymentRMQConstants = "payment.authorized"
	RoutingKeyBookingConfirmed  PaymentRMQConstants = "booking.confirmed"
	RoutingKeyBookingFailed     PaymentRMQConstants = "booking.failed"
	RoutingKeyPaymentWebhook    PaymentRMQConstants = "payment.webhook"

	// Queue binding patterns (what consumers listen to)
	// Use # to match all sub-routing keys under payment.*
	BindingKeyPaymentAll     PaymentRMQConstants = "payment.#"
	BindingKeyPaymentWebhook PaymentRMQConstants = "payment.webhook.#"
	// Matches single-word domain events (payment.created, payment.authorized) but not payment.webhook.*
	BindingKeyPaymentDomain PaymentRMQConstants = "payment.*"
	BindingKeyBookingAll    PaymentRMQConstants = "booking.#"
)

// Queue names
const (
	QueuePaymentPostgres PaymentRMQConstants = "payment.postgres.queue"
	QueuePaymentWebhook  PaymentRMQConstants = "payment.webhook.queue"
	QueuePaymentDLQ      PaymentRMQConstants = "payment.dlq"
)
