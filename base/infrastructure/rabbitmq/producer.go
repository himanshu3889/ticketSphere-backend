package baseRabbitMQ

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Producer struct {
	conn               *amqp091.Connection
	exchangeName       string
	exchangeKind       string // "direct", "fanout", "topic", "headers"
	exchangeDurable    bool
	exchangeAutoDelete bool
	exchangeInternal   bool
	exchangeNoWait     bool
	exchangeArgs       amqp091.Table

	logger       *logrus.Logger
	tracer       trace.Tracer
	deliveryMode uint8

	mu      sync.RWMutex // Avoid data race when creating new channel and using old one
	channel *amqp091.Channel
}

// One producer. One connection. Many routing keys.
func NewProducer(
	conn *amqp091.Connection,
	exchangeName string,
	kind string,
	durable bool,
	autoDelete bool,
	internal bool,
	noWait bool,
	args amqp091.Table,
	logger *logrus.Logger,
	tracer trace.Tracer,
) (*Producer, error) {
	p := &Producer{
		conn:               conn,
		exchangeName:       exchangeName,
		exchangeKind:       kind,
		exchangeDurable:    durable,
		exchangeAutoDelete: autoDelete,
		exchangeInternal:   internal,
		exchangeNoWait:     noWait,
		exchangeArgs:       args,
		logger:             logger,
		tracer:             tracer,
		deliveryMode:       amqp091.Persistent,
	}
	if err := p.init(); err != nil {
		return nil, err
	}
	return p, nil
}

// Initialize the producer
func (p *Producer) init() error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}

	if err := ch.ExchangeDeclare(p.exchangeName, p.exchangeKind, p.exchangeDurable, p.exchangeAutoDelete, p.exchangeInternal, p.exchangeNoWait, p.exchangeArgs); err != nil {
		ch.Close()
		return err
	}

	if err := ch.Confirm(false); err != nil {
		ch.Close()
		return err
	}

	p.mu.Lock()
	p.channel = ch
	p.mu.Unlock()

	go func() {
		closeCh := ch.NotifyClose(make(chan *amqp091.Error, 1))
		if err := <-closeCh; err != nil {
			p.logger.Errorf("producer channel closed: %v", err)
		}
	}()

	return nil
}

// Publish sends a message with full OpenTelemetry tracing.
// Pass the incoming request context (from HTTP handler) so the trace continues.
func (p *Producer) Publish(ctx context.Context, routingKey string, body []byte, headers amqp091.Table) error {
	p.mu.RLock()
	ch := p.channel // <-- Safely grab a local copy of the pointer
	p.mu.RUnlock()

	// Start a producer span. This becomes a child of whatever span is in ctx.
	ctx, span := p.tracer.Start(ctx, "rabbitmq.publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination", p.exchangeName),
			attribute.String("messaging.destination_kind", "exchange"),
			attribute.String("messaging.rabbitmq.routing_key", routingKey),
		),
	)
	defer span.End()

	if headers == nil {
		headers = amqp091.Table{}
	}

	// Inject the current trace context into AMQP headers.
	// This is how the consumer will pick up the trace and continue it.
	carrier := AMQPHeaderCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	msg := amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: p.deliveryMode,
		Timestamp:    time.Now(),
		Body:         body,
		Headers:      headers,
	}

	confirmation, err := ch.PublishWithDeferredConfirmWithContext(ctx, p.exchangeName, routingKey, false, false, msg)
	if err != nil {
		span.RecordError(err)
		p.FailureMetric()
		return err
	}

	select {
	case <-confirmation.Done():
		if confirmation.Acked() {
			span.SetAttributes(attribute.Bool("messaging.rabbitmq.confirm", true))
			p.SuccessMetric()
			return nil
		}
		span.SetAttributes(attribute.Bool("messaging.rabbitmq.confirm", false))
		p.FailureMetric()
		return fmt.Errorf("publish was rejected by broker")
	case <-ctx.Done():
		span.RecordError(ctx.Err())
		p.FailureMetric()
		return ctx.Err()
	}
}

// Close the producer
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.channel != nil {
		return p.channel.Close()
	}
	return nil
}

func (p *Producer) SuccessMetric() {
	// baseMetrics.RabbitMQProducerSuccessMessages.WithLabelValues(p.exchangeName, p.exchangeRoutingKey).Inc()
}

func (p *Producer) FailureMetric() {
	// baseMetrics.RabbitMQProducerFailedMessages.WithLabelValues(p.exchangeName, p.exchangeRoutingKey).Inc()
}
