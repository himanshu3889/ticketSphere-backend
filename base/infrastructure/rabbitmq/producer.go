package baseRabbitMQ

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Publish channel pool bounds (env RABBITMQ_PRODUCER_CHANNELS).
const (
	producerChannelDefault = 16
	producerChannelMax     = 32
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

	// Pool of confirm-mode channels picked round-robin
	channels []*amqp091.Channel
	next     atomic.Uint64

	mu     sync.Mutex // channel lifecycle only
	closed bool
}

// NewProducer wires a producer: one connection, one exchange, many routing keys.
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

// resolveChannelCount reads RABBITMQ_PRODUCER_CHANNELS
func resolveChannelCount() int {
	n, err := strconv.Atoi(os.Getenv("RABBITMQ_PRODUCER_CHANNELS"))
	if err != nil {
		return producerChannelDefault
	}
	if n < 1 {
		return 1
	}
	if n > producerChannelMax {
		return producerChannelMax
	}
	return n
}

// declares the exchange once and opens a pool of confirm-mode channels
func (p *Producer) init() error {
	count := resolveChannelCount()
	channels := make([]*amqp091.Channel, 0, count)

	for i := 0; i < count; i++ {
		ch, err := p.conn.Channel()
		if err != nil {
			closeChannels(channels)
			return err
		}

		if i == 0 {
			if err := ch.ExchangeDeclare(p.exchangeName, p.exchangeKind, p.exchangeDurable, p.exchangeAutoDelete, p.exchangeInternal, p.exchangeNoWait, p.exchangeArgs); err != nil {
				ch.Close()
				closeChannels(channels)
				return err
			}
		}

		if err := ch.Confirm(false); err != nil {
			ch.Close()
			closeChannels(channels)
			return err
		}

		channels = append(channels, ch)

		go func(ch *amqp091.Channel) {
			closeCh := ch.NotifyClose(make(chan *amqp091.Error, 1))
			if err := <-closeCh; err != nil {
				p.logger.Errorf("producer channel closed: %v", err)
			}
		}(ch)
	}

	p.mu.Lock()
	p.channels = channels
	p.mu.Unlock()

	p.logger.Infof("rabbitmq producer ready (exchange=%s channels=%d)", p.exchangeName, count)

	return nil
}

func closeChannels(channels []*amqp091.Channel) {
	for _, ch := range channels {
		ch.Close()
	}
}

// nextChannel picks the next channel round-robin.
func (p *Producer) nextChannel() *amqp091.Channel {
	return p.channels[p.next.Add(1)%uint64(len(p.channels))]
}

// Publish sends a message and waits for the broker confirm.
// Pass the request ctx so the trace continues.
func (p *Producer) Publish(ctx context.Context, routingKey string, body []byte, headers amqp091.Table) error {
	ch := p.nextChannel()

	// Start a producer span. This becomes a child of whatever span is in ctx.
	ctx, span := p.tracer.Start(ctx, constants.SpanRabbitMQPublish,
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

	// inject the trace context into AMQP headers; the consumer extracts it
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

// PublishAsync hands the message to the broker connection and returns WITHOUT
// waiting for the broker confirm, so the caller's request path never blocks on broker fsync.
func (p *Producer) PublishAsync(ctx context.Context, routingKey string, body []byte, headers amqp091.Table, confirmTimeout time.Duration, onNack func()) error {
	ch := p.nextChannel()

	// Start a producer span. This becomes a child of whatever span is in ctx.
	// It outlives the caller's request and ends when the confirm resolves.
	ctx, span := p.tracer.Start(ctx, constants.SpanRabbitMQPublish,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination", p.exchangeName),
			attribute.String("messaging.destination_kind", "exchange"),
			attribute.String("messaging.rabbitmq.routing_key", routingKey),
		),
	)

	if headers == nil {
		headers = amqp091.Table{}
	}

	// inject the trace context into AMQP headers; the consumer extracts it
	carrier := AMQPHeaderCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	msg := amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: p.deliveryMode,
		Timestamp:    time.Now(),
		Body:         body,
		Headers:      headers,
	}

	// Detached: the publish + confirm must finish even if the client hangs up.
	pubCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), confirmTimeout)

	confirmation, err := ch.PublishWithDeferredConfirmWithContext(pubCtx, p.exchangeName, routingKey, false, false, msg)
	if err != nil {
		cancel()
		span.RecordError(err)
		span.End()
		p.FailureMetric()
		return err
	}

	go func() {
		defer cancel()
		defer span.End()

		select {
		case <-confirmation.Done():
			if confirmation.Acked() {
				span.SetAttributes(attribute.Bool("messaging.rabbitmq.confirm", true))
				p.SuccessMetric()
				return
			}
			span.SetAttributes(attribute.Bool("messaging.rabbitmq.confirm", false))
			p.FailureMetric()
			// Definitive rejection: the broker never accepted it — safe to undo.
			if onNack != nil {
				onNack()
			}
		case <-pubCtx.Done():
			// Ambiguous: broker may still hold the message. Log loudly with the
			// routing key so these are greppable for reconciliation.
			span.SetAttributes(attribute.Bool("messaging.rabbitmq.confirm_timeout", true))
			p.FailureMetric()
			p.logger.Errorf("rabbitmq publish confirm timeout (exchange=%s routing_key=%s)", p.exchangeName, routingKey)
		}
	}()

	return nil
}

// Close shuts all producer channels.
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true

	var firstErr error
	for _, ch := range p.channels {
		if err := ch.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (p *Producer) SuccessMetric() {
	// baseMetrics.RabbitMQProducerSuccessMessages.WithLabelValues(p.exchangeName, p.exchangeRoutingKey).Inc()
}

func (p *Producer) FailureMetric() {
	// baseMetrics.RabbitMQProducerFailedMessages.WithLabelValues(p.exchangeName, p.exchangeRoutingKey).Inc()
}
