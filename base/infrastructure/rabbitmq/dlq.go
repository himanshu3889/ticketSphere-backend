package baseRabbitMQ

import (
	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// DeclareDLQ declares one module's dead-letter topology: DLX exchange, durable
// DLQ queue, and a catch-all "#" binding. Idempotent; called once per module at
// consumer startup. The DLQ is terminal on purpose
// TODO: I think it should be configurable
func DeclareDLQ(conn *amqp091.Connection, dlxName, queueName string, logger *logrus.Logger) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(dlxName, "topic", true, false, false, false, nil); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return err
	}

	if err := ch.QueueBind(queueName, "#", dlxName, false, nil); err != nil {
		return err
	}

	logger.WithFields(logrus.Fields{"dlx": dlxName, "queue": queueName}).Info("DLQ topology declared")
	return nil
}
