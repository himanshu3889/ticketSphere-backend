package constants

// Span names for OpenTelemetry tracing.
const (
	SpanRabbitMQPublish       = "rabbitmq.publish"
	SpanRedisRunScript        = "Redis.RunScript"
	SpanSeatBookingExecute    = "Seat.Booking.Execute"
	SpanEventSeatsTotalPrice  = "GetEventSeatsTotalPrice"
	SpanBookingGetUserBooking = "Booking.GetUserBooking"
	SpanPaymentInitiate       = "Payment.Initiate"
	SpanPaymentAuthorized     = "Payment.Authorized"
	SpanGatewayRequest        = "gateway.request" // static replacement for dynamic fmt.Sprintf
)

// Tracer names for OpenTelemetry tracers.
const (
	TracerPaymentGateway    = "ticketSphere.payment.gateway"
	TracerPaymentCacheStore = "ticketSphere.payment.cacheStore"
	TracerPaymentPublisher  = "ticketSphere.payment.publisher"
	TracerPaymentConsumers  = "ticketSphere.payment.consumers"

	TracerBookingCacheStore = "ticketSphere.booking.cacheStore"
	TracerBookingConsumers  = "ticketSphere.booking.consumers"
	TracerBookingPublisher  = "ticketSphere.booking.publisher"

	TracerCacheManager = "ticketSphere.cacheManager"
)

// Bounded keys identify Redis Lua scripts and cache operations.
const (
	BoundedKeyBookSeats                  = "book_seats"
	BoundedKeyPaymentInitiateBookingGet  = "payment.initiate.booking.get"
	BoundedKeyPaymentAuthorizeBookingGet = "payment.authorize.booking"
)
