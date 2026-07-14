package baseRabbitMQ

import "github.com/rabbitmq/amqp091-go"

// Carrier adapter so OpenTelemetry knows how to write to RabbitMQ's amqp Table
type AMQPHeaderCarrier amqp091.Table

func (c AMQPHeaderCarrier) Get(key string) string {
	if v, ok := c[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (c AMQPHeaderCarrier) Set(key, value string) {
	c[key] = value
}

func (c AMQPHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
