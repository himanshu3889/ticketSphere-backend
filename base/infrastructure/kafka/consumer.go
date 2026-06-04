package baseKafka

import (
	"context"
	"errors"
	"time"

	baseMetrics "github.com/himanshu3889/ticketSphere-backend/base/metric"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// holds all dynamic parameters
type ConsumerConfig struct {
	Brokers []string
	GroupID string
	Topic   string

	AutoCommit bool

	StartOffset int64

	// Batching parameters
	EnableBatching bool
	BatchSize      int           // Number of messages to accumulate before processing
	BatchTimeout   time.Duration // Max time to wait before processing an incomplete batch
}

// Consumer routes by topic
type Consumer struct {
	reader *kafka.Reader
	config ConsumerConfig

	// We keep both handlers available depending on the mode
	singleHandler func(*kafka.Message) (error, *kafka.Message)
	batchHandler  func([]*kafka.Message) (error, []*kafka.Message)
	dlqHandler    func([]*kafka.Message) error

	logger *logrus.Logger
}

// New kafka cm; dynamically configures the reader based on the passed config
func NewConsumer(cfg ConsumerConfig, singleHandler func(*kafka.Message) (error, *kafka.Message), batchHandler func([]*kafka.Message) (error, []*kafka.Message), dlqHandler func([]*kafka.Message) error, logger *logrus.Logger) *Consumer {
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: false,
	}

	// Align Kafka's internal fetch timeout with your batch timeout if batching is enabled
	maxWait := 500 * time.Millisecond
	if cfg.EnableBatching && cfg.BatchTimeout > 0 {
		maxWait = cfg.BatchTimeout
	}

	config := kafka.ReaderConfig{
		Brokers: cfg.Brokers,
		GroupID: cfg.GroupID,
		Topic:   cfg.Topic,
		// Partition: 0, //

		MinBytes: 10e3,    // minimum batch size broker should send
		MaxBytes: 10e6,    // max batch size
		MaxWait:  maxWait, // wait to accumulate batch

		// Group coordination - relaxed for local Docker
		SessionTimeout:    30 * time.Second,
		RebalanceTimeout:  30 * time.Second,
		HeartbeatInterval: 3 * time.Second,
		StartOffset:       cfg.StartOffset,

		Dialer: dialer,
		// TODO: error logger here and metric for that also
		// ENABLE INTERNAL DEBUGGING
		// This prints directly to stdout so you can see the handshake
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			// logger.Infof("[KAFKA-DEBUG] "+msg, args...)
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			// logger.Errorf("[KAFKA-ERROR] "+msg, args...)
		}),
	}

	return &Consumer{
		reader:        kafka.NewReader(config),
		config:        cfg,
		singleHandler: singleHandler,
		batchHandler:  batchHandler,
		dlqHandler:    dlqHandler,
		logger:        logger,
	}
}

// Start the kafka cm
func (cm *Consumer) Start(ctx context.Context) error {
	cm.logger.Info("Consumer started, joining group...") // Add this
	defer cm.reader.Close()

	if cm.config.EnableBatching {
		return cm.runBatchMode(ctx)
	}
	return cm.runSingleMode(ctx)
}

// processes messages one by one
func (cm *Consumer) runSingleMode(ctx context.Context) error {
	if cm.config.EnableBatching {
		return errors.New("Batching is enabled for single mode")
	}
	if cm.singleHandler == nil {
		return errors.New("Single handler not provided")
	}
	if cm.config.AutoCommit {
		return cm.runSingleAutoCommit(ctx)
	}
	return cm.runSingleManualCommit(ctx)
}

// AUTO-COMMIT: ReadMessage fetches AND flags the message to be committed in the background
func (cm *Consumer) runSingleAutoCommit(ctx context.Context) error {
	for {
		msg, err := cm.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		// Process the message
		err, dlq := cm.singleHandler(&msg)
		if dlq != nil {
			cm.FailureMessagesMetric(1)
			if cm.dlqHandler != nil {
				cm.dlqHandler([]*kafka.Message{dlq})
			}
		} else {
			cm.SuccessMessagesMetric(1)
		}
	}
}

// MANUAL COMMIT: FetchMessage only fetches. You must commit explicitly.
func (cm *Consumer) runSingleManualCommit(ctx context.Context) error {
	for {
		msg, err := cm.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		// Process the message. ONLY commit if successful.
		err, dlq := cm.singleHandler(&msg)
		if err == nil {
			if commitErr := cm.reader.CommitMessages(ctx, msg); commitErr != nil {
				cm.FailureMessagesMetric(1)
				return commitErr
			}
		}
		if dlq != nil {
			cm.FailureMessagesMetric(1)
			if cm.dlqHandler != nil {
				cm.dlqHandler([]*kafka.Message{dlq})
			}
		} else {
			cm.SuccessMessagesMetric(1)
		}
	}
}

// accumulates messages and commits the full batch manually
func (cm *Consumer) runBatchMode(ctx context.Context) error {
	if !cm.config.EnableBatching {
		return errors.New("Batching is not enabled for batch mode")
	}
	if cm.batchHandler == nil {
		return errors.New("Batch handler not provided")
	}
	if cm.config.AutoCommit {
		return cm.runBatchAutoCommit(ctx)
	}
	return cm.runBatchManualCommit(ctx)
}

// AUTO-COMMIT: Let kafka handle commits, just batch the processing
func (cm *Consumer) runBatchAutoCommit(ctx context.Context) error {
	batch := make([]*kafka.Message, 0, cm.config.BatchSize)
	timer := time.NewTimer(cm.config.BatchTimeout)
	defer timer.Stop()

	// Process the batch messages
	batchProcess := func() error {
		defer func() {
			batch = batch[:0]
			timer.Reset(cm.config.BatchTimeout)
		}()

		totalMessageCnt := len(batch)
		if totalMessageCnt == 0 {
			return nil
		}

		// In auto-commit, messages already committed
		_, dlq := cm.batchHandler(batch)

		failedMessageCnt := len(dlq)
		successMessageCnt := totalMessageCnt - failedMessageCnt
		cm.FailureMessagesMetric(failedMessageCnt)
		cm.SuccessMessagesMetric(successMessageCnt)

		if dlq != nil {
			if cm.dlqHandler != nil { // TODO: can use the metric how many we are pushing to the dlq
				cm.dlqHandler(dlq)
			} else {
				// Log a warning
				cm.logger.Warn("DLQ messages received but no dlqHandler is configured")
			}
		}

		return nil
	}

	for {
		select {
		case <-ctx.Done():
			// Process remaining batch before exit
			batchProcess()
			return nil
		case <-timer.C:
			batchProcess()

		default:
			// ReadMessage is batched at network level
			// The first call fetches a big batch from network. Next remaining calls just pop from memory.
			readCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			msg, err := cm.reader.FetchMessage(readCtx)
			cancel()

			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				if errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				cm.logger.Errorf("Read message error: %v", err)
				continue
			}

			batch = append(batch, &msg)

			// Process if batch is full
			if len(batch) >= cm.config.BatchSize {
				batchProcess()
			}
		}
	}
}

// MANUAL COMMIT: Fetch, batch, process, then commit all at once
func (cm *Consumer) runBatchManualCommit(ctx context.Context) error {
	batch := make([]*kafka.Message, 0, cm.config.BatchSize)
	timer := time.NewTimer(cm.config.BatchTimeout)
	defer timer.Stop()

	// Helper to process and commit
	processAndCommit := func() error {
		defer func() {
			batch = batch[:0]
			timer.Reset(cm.config.BatchTimeout)
		}()

		totalMessageCnt := len(batch)
		if totalMessageCnt == 0 {
			return nil
		}

		// Process the batch
		err, dlq := cm.batchHandler(batch)

		failedMessageCnt := len(dlq)
		successMessageCnt := totalMessageCnt - failedMessageCnt
		cm.FailureMessagesMetric(failedMessageCnt)
		cm.SuccessMessagesMetric(successMessageCnt)

		if dlq != nil {
			if cm.dlqHandler != nil { // TODO: can use the metric how many we are pushing to the dlq
				err := cm.dlqHandler(dlq)
				if err != nil {
					// dlq failed no more process; no commit etc
					return err
				}
			} else {
				// Log a warning
				cm.logger.Warn("DLQ messages received but no dlqHandler is configured")
			}
		}

		// Commit all messages in the batch.
		commitBatch := make([]kafka.Message, len(batch))
		for i, msgPtr := range batch {
			commitBatch[i] = *msgPtr
		}

		// Commit all messages using the new value slice
		if err = cm.reader.CommitMessages(ctx, commitBatch...); err != nil {
			return err
		}

		return nil
	}

	for {
		select {
		case <-ctx.Done():
			// Process remaining batch before exit
			processAndCommit()
			return nil

		case <-timer.C:
			processAndCommit()

		default:
			// Non-blocking read with short timeout
			readCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			msg, err := cm.reader.FetchMessage(readCtx)
			cancel()

			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				if errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				cm.logger.Errorf("Read message error: %v", err)
				continue
			}
			batch = append(batch, &msg)

			// Process if batch is full
			if len(batch) >= cm.config.BatchSize {
				processAndCommit()
			}
		}
	}
}

// Success messages metric
func (cm *Consumer) SuccessMessagesMetric(cnt int) {
	if cnt <= 0 {
		return
	}
	baseMetrics.KafkaConsumerSuccessMessages.WithLabelValues(cm.config.Topic).Add(float64(cnt))
}

// Failure messages metric
func (cm *Consumer) FailureMessagesMetric(cnt int) {
	if cnt <= 0 {
		return
	}
	baseMetrics.KafkaConsumerFailedMessages.WithLabelValues(cm.config.Topic).Add(float64(cnt))
}

// Close the kafka cm
func (cm *Consumer) Close() error {
	return cm.reader.Close()
}
