package baseRabbitMQ

import (
	"fmt"

	"github.com/himanshu3889/ticketSphere-backend/configs"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// Initialize RabbitMQ
func InitRabbitMQ(rmqCfg *configs.RMQConfig, logger *logrus.Logger) (*amqp091.Connection, error) {
	connString := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		rmqCfg.User,
		rmqCfg.Password,
		rmqCfg.Host,
		rmqCfg.Port,
	)

	conn, err := amqp091.Dial(connString)
	if err != nil {
		return nil, err
	}

	logger.Info("RabbitMQ connected successfully")

	return conn, nil
}

func CloseRabbitMQConnection(conn *amqp091.Connection) error {
	return conn.Close()
}
