package consumers

import (
	"context"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// newDLQHandler returns the dead-letter handler shared by all booking consumers.
func newDLQHandler(logger *logrus.Logger, consumerName string) func(context.Context, amqp091.Delivery) error {
	return func(ctx context.Context, msg amqp091.Delivery) error {
		// TODO: alert / persist for later re-drive
		logger.WithFields(logrus.Fields{
			"consumer":    consumerName,
			"exchange":    msg.Exchange,
			"routing_key": msg.RoutingKey,
			"body":        string(msg.Body),
		}).Error("message dead-lettered")
		return nil
	}
}
