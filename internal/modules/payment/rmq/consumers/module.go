package consumers

import (
	"context"
	"sync"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	baseRabbitMQ "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/rabbitmq"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	paymentRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/rmq"
)

// StartConsumers starts all consumers for the payment module.
func StartConsumers(conn *amqp091.Connection, cacheStore *paymentCacheStore.CacheStore, logger *logrus.Logger, traceProvider trace.TracerProvider) error {
	// Create one tracer for the entire consumer layer
	consumerTracer := traceProvider.Tracer(constants.TracerPaymentConsumers)

	// Declare the durable DLQ before any pool starts so dead letters are retained from the first message
	if err := baseRabbitMQ.DeclareDLQ(conn, string(paymentRMQ.ExchangePaymentDLX), string(paymentRMQ.QueuePaymentDLQ), logger); err != nil {
		logger.WithError(err).Error("Failed to declare payment DLQ topology")
	}

	// TODO: make worker count configurable in yml file
	workerCount := 2
	go StartWebhookConsumer(conn, logger, cacheStore, workerCount, consumerTracer)
	go StartPostgresConsumer(conn, logger, cacheStore, workerCount, consumerTracer)
	return nil
}

// runConsumerPool starts workerCount workers consuming the queue described by cfg and blocks until ctx is cancelled.
func runConsumerPool(
	conn *amqp091.Connection,
	cfg baseRabbitMQ.ConsumerConfig,
	name string,
	workerCount int,
	handler func(context.Context, amqp091.Delivery) error,
	logger *logrus.Logger,
	tracer trace.Tracer,
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	for i := 1; i <= workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			consumer := baseRabbitMQ.NewConsumer(conn, cfg, tracer, handler, newDLQHandler(logger, name), logger)
			logger.WithFields(logrus.Fields{"consumer": name, "worker": id}).Info("consumer worker started")

			if err := consumer.Start(ctx); err != nil {
				logger.WithError(err).WithFields(logrus.Fields{"consumer": name, "worker": id}).Error("consumer worker crashed")
			}
		}(i)
	}

	// Blocks forever until ctx is cancelled (shutdown signal)
	wg.Wait()
}
