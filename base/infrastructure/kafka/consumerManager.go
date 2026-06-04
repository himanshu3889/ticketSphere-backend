package baseKafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// TODO: error logger here and metric for that also

// ConsumerManager runs multiple consumers with graceful shutdown
type ConsumerManager struct {
	name      string
	consumers []*Consumer
	wg        sync.WaitGroup
	cancel    context.CancelFunc
	logger    *logrus.Logger
}

// New consumer manager for the kafka
func NewConsumerManager(name string, logger *logrus.Logger) *ConsumerManager {
	return &ConsumerManager{
		name:      name,
		consumers: make([]*Consumer, 0),
		logger:    logger,
	}
}

// Add registers a consumer but doesn't start it yet
func (cm *ConsumerManager) Add(cfg ConsumerConfig, singleHandler func(*kafka.Message) (error, *kafka.Message), batchHandler func([]*kafka.Message) (error, []*kafka.Message), dlqHandler func([]*kafka.Message) error) {
	consumer := NewConsumer(cfg, singleHandler, batchHandler, dlqHandler, cm.logger)
	cm.consumers = append(cm.consumers, consumer)
}

// Start runs all consumers in parallel
func (cm *ConsumerManager) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	cm.cancel = cancel
	cm.logger.Infof("Running `%s` kafka consumer manager...", cm.name)
	for _, consumer := range cm.consumers {
		cm.wg.Add(1)
		go func(c *Consumer) {
			defer cm.wg.Done()
			if err := c.Start(ctx); err != nil {
				cm.logger.WithError(err).Error("Consumer crashed")
			}
		}(consumer)
	}

}

// Stop gracefully shuts down all consumers
func (cm *ConsumerManager) Stop(timeout time.Duration) error {
	cm.cancel() // Signal shutdown

	done := make(chan struct{})
	go func() {
		cm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout for the `%s` kafka consumer mananger", cm.name)
	}
}
