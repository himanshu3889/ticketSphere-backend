package baseRabbitMQ

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ConsumerConfig struct {
	QueueName     string
	ExchangeName  string
	RoutingKey    string
	DLXExchange   string
	DLXRoutingKey string
	PrefetchCount int

	// Concurrency > 1 keeps that many handlers in flight on this consumer's
	Concurrency int
	// (used by the batching projection consumer).
	ManualAck bool
	// QueueType "quorum" declares the queue with x-queue-type: quorum.
	QueueType string
	// DeliveryLimit sets the quorum x-delivery-limit arg: messages redelivered
	DeliveryLimit int
	OnStop        func(ctx context.Context) error
}

type Consumer struct {
	config     ConsumerConfig
	conn       *amqp091.Connection
	tracer     trace.Tracer
	logger     *logrus.Logger
	handler    func(context.Context, amqp091.Delivery) error
	dlqHandler func(context.Context, amqp091.Delivery) error

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func NewConsumer(
	conn *amqp091.Connection,
	cfg ConsumerConfig,
	tracer trace.Tracer,
	handler func(context.Context, amqp091.Delivery) error,
	dlqHandler func(context.Context, amqp091.Delivery) error,
	logger *logrus.Logger,
) *Consumer {
	return &Consumer{
		conn:       conn,
		config:     cfg,
		handler:    handler,
		dlqHandler: dlqHandler,
		logger:     logger,
		tracer:     tracer,
	}
}

// SetupTopology declares exchange, DLX, queue, and binding.
func (c *Consumer) SetupTopology() error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(c.config.ExchangeName, "topic", true, false, false, false, nil); err != nil {
		return err
	}

	if c.config.DLXExchange != "" {
		if err := ch.ExchangeDeclare(c.config.DLXExchange, "topic", true, false, false, false, nil); err != nil {
			return err
		}
	}

	args := amqp091.Table{}
	if c.config.DLXExchange != "" {
		args["x-dead-letter-exchange"] = c.config.DLXExchange
		if c.config.DLXRoutingKey != "" {
			args["x-dead-letter-routing-key"] = c.config.DLXRoutingKey
		}
	}
	if c.config.QueueType != "" {
		args["x-queue-type"] = c.config.QueueType
		if c.config.DeliveryLimit > 0 {
			args["x-delivery-limit"] = int32(c.config.DeliveryLimit)
		}
	}

	if _, err := ch.QueueDeclare(c.config.QueueName, true, false, false, false, args); err != nil {
		return err
	}

	if err := ch.QueueBind(c.config.QueueName, c.config.RoutingKey, c.config.ExchangeName, false, nil); err != nil {
		return err
	}

	return nil
}

// start consumer
func (c *Consumer) Start(ctx context.Context) error {
	if err := c.SetupTopology(); err != nil {
		return err
	}

	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	prefetch := c.config.PrefetchCount
	if prefetch <= 0 {
		prefetch = 1
	}
	if err := ch.Qos(prefetch, 0, false); err != nil {
		return err
	}

	// Consume messages from queue
	// TODO: MAKE IT DYNAMIC, MAKE IT CONTEXT CANCELABLE, MAKE IT CONFIGURABLE
	deliveries, err := ch.Consume(c.config.QueueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	c.logger.Infof("consumer started on queue: %s", c.config.QueueName)

	if c.config.Concurrency > 1 {
		return c.consumeConcurrent(ctx, deliveries, c.config.Concurrency)
	}
	return c.consumeSerial(ctx, deliveries)
}

// consumeSerial is the original one-message-at-a-time loop.
func (c *Consumer) consumeSerial(ctx context.Context, deliveries <-chan amqp091.Delivery) error {
	for {
		select {
		case <-ctx.Done():
			c.runOnStop()
			return nil
		case msg, ok := <-deliveries:
			if !ok {
				return errors.New("delivery channel closed")
			}
			c.handleMessage(ctx, msg)
		}
	}
}

// Acking from multiple goroutines is safe: amqp091 serializes channel writes, and every ack carries multiple=false.
func (c *Consumer) consumeConcurrent(ctx context.Context, deliveries <-chan amqp091.Delivery, concurrency int) error {
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			c.runOnStop()
			return nil
		case msg, ok := <-deliveries:
			if !ok {
				return errors.New("delivery channel closed")
			}
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				// leave msg unacked; the channel close on exit requeues it
				wg.Wait()
				c.runOnStop()
				return nil
			}
			wg.Add(1)
			go func(msg amqp091.Delivery) {
				defer wg.Done()
				defer func() { <-sem }()
				c.handleMessage(ctx, msg)
			}(msg)
		}
	}
}

// handleMessage runs the handler for one delivery and applies the ack policy.
func (c *Consumer) handleMessage(ctx context.Context, msg amqp091.Delivery) {
	// Extract trace from headers and start a child span
	msgCtx, span := c.extractAndStartSpan(ctx, msg)

	err := c.handler(msgCtx, msg)
	if err != nil {
		span.RecordError(err)

		c.FailureMetric()
		c.logger.WithError(err).Error("handler failed")

		if c.config.ManualAck {
			// The handler owns the outcome entirely (acks, nacks, DLQ routing.
			span.End()
			return
		}

		if c.dlqHandler != nil {
			c.dlqHandler(msgCtx, msg)
		}

		// nack without requeue -> lands in DLX
		if nackErr := msg.Nack(false, false); nackErr != nil {
			span.RecordError(nackErr) // Optional: record the nack error too
			c.logger.WithError(nackErr).Error("nack failed")
		}

		span.End() // End the span here, after all work is done.
		return
	}

	if !c.config.ManualAck {
		if ackErr := msg.Ack(false); ackErr != nil {
			c.logger.WithError(ackErr).Error("ack failed")
			span.RecordError(ackErr)
		}
	}

	span.End() // Success path span end
	c.SuccessMetric()
}

// runOnStop invokes the drain hook with its own deadline so a hanging hook
func (c *Consumer) runOnStop() {
	if c.config.OnStop == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.config.OnStop(ctx); err != nil {
		c.logger.WithError(err).Error("consumer OnStop hook failed")
	}
}

// Stop signals the consumer to shut down.
func (c *Consumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

// Done returns a channel that closes when the consumer exits.
func (c *Consumer) Done() <-chan struct{} {
	return c.done
}

func (c *Consumer) extractAndStartSpan(parentCtx context.Context, d amqp091.Delivery) (context.Context, trace.Span) {
	carrier := AMQPHeaderCarrier(d.Headers)
	msgCtx := otel.GetTextMapPropagator().Extract(parentCtx, carrier)

	spanName := fmt.Sprintf("rabbitmq.consume %s", c.config.QueueName)

	return c.tracer.Start(msgCtx, spanName,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination", d.Exchange),
			attribute.String("messaging.rabbitmq.routing_key", d.RoutingKey),
			attribute.String("messaging.rabbitmq.delivery_tag", fmt.Sprintf("%d", d.DeliveryTag)),
		),
	)
}

func (c *Consumer) SuccessMetric() {
	// baseMetrics.RabbitMQConsumerSuccessMessages.WithLabelValues(c.config.QueueName).Inc()
}

func (c *Consumer) FailureMetric() {
	// baseMetrics.RabbitMQConsumerFailedMessages.WithLabelValues(c.config.QueueName).Inc()
}

func (c *Consumer) Close() error {
	return nil
}
