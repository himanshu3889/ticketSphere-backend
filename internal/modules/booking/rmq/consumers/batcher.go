package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"

	"github.com/himanshu3889/ticketSphere-backend/base/models"
	bookingStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/store"
)

// identifies which projection a booking event belongs
type batchGroup string

const (
	batchGroupCreated   batchGroup = "booking.created"
	batchGroupConfirmed batchGroup = "booking.confirmed"
	batchGroupFailed    batchGroup = "booking.failed"
)

// batchItem is one buffered event with the delivery that carries its ack.
type batchItem struct {
	req         *models.BookingRequest
	confirmedAt time.Time // booking.confirmed only
	delivery    amqp091.Delivery
}

// accumulates booking events per group and flushes each group as ONE Postgres transaction
// Ack policy: after a successful commit every buffered delivery is acked. On
// store failure the batch backs off, then nacks with requeue
type postgresBatcher struct {
	store      *bookingStore.Store
	logger     *logrus.Logger
	batchSize  int
	flushEvery time.Duration
	backoff    time.Duration

	mu        sync.Mutex
	created   []batchItem
	confirmed []batchItem
	failed    []batchItem
	flushNow  chan batchGroup
	stopCh    chan struct{}
	stoppedCh chan struct{}
	stopOnce  sync.Once
}

func newPostgresBatcher(store *bookingStore.Store, logger *logrus.Logger, batchSize int, flushEvery time.Duration) *postgresBatcher {
	if batchSize <= 0 {
		batchSize = 250
	}
	if flushEvery <= 0 {
		flushEvery = 100 * time.Millisecond
	}
	b := &postgresBatcher{
		store:      store,
		logger:     logger,
		batchSize:  batchSize,
		flushEvery: flushEvery,
		backoff:    500 * time.Millisecond,
		flushNow:   make(chan batchGroup, 3),
		stopCh:     make(chan struct{}),
		stoppedCh:  make(chan struct{}),
	}
	go b.run()
	return b
}

// unmarshals a booking domain event body.
func decodeBookingEvent(body []byte) (*models.BookingRequest, error) {
	var bookingRequest models.BookingRequest
	if err := json.Unmarshal(body, &bookingRequest); err != nil {
		return nil, err // poison message -> DLQ
	}
	return &bookingRequest, nil
}

// rejects events that can never be projected
func validateBookingEvent(bookingRequest *models.BookingRequest) error {
	switch {
	case bookingRequest.ID <= 0:
		return errors.New("booking event has invalid id")
	case bookingRequest.EventID <= 0:
		return errors.New("booking event has invalid event_id")
	case bookingRequest.UserID <= 0:
		return errors.New("booking event has invalid user_id")
	case len(bookingRequest.SeatIDs) == 0:
		return errors.New("booking event has no seat_ids")
	}
	return nil
}

// decodes, validates and buffers one delivery. Poison events are nacked straight to the DLQ
func (b *postgresBatcher) AddEvent(group batchGroup, msg amqp091.Delivery) error {
	bookingRequest, err := decodeBookingEvent(msg.Body)
	if err == nil {
		err = validateBookingEvent(bookingRequest)
	}
	if err != nil {
		b.logger.WithError(err).WithFields(logrus.Fields{
			"routing_key": msg.RoutingKey,
			"body":        string(msg.Body),
		}).Error("poison booking event for postgres projection, dead-lettering")
		if nackErr := msg.Nack(false, false); nackErr != nil {
			b.logger.WithError(nackErr).Error("poison nack failed")
		}
		return err
	}

	item := batchItem{req: bookingRequest, delivery: msg}
	if group == batchGroupConfirmed {
		// The routing key defines the transition
		item.confirmedAt = time.Now().UTC()
		if bookingRequest.ConfirmedAt != nil {
			item.confirmedAt = *bookingRequest.ConfirmedAt
		}
	}

	b.mu.Lock()
	var count int
	switch group {
	case batchGroupCreated:
		b.created = append(b.created, item)
		count = len(b.created)
	case batchGroupConfirmed:
		b.confirmed = append(b.confirmed, item)
		count = len(b.confirmed)
	case batchGroupFailed:
		b.failed = append(b.failed, item)
		count = len(b.failed)
	}
	b.mu.Unlock()

	if count >= b.batchSize {
		select {
		case b.flushNow <- group:
		default: // a flush is already pending for this group
		}
	}
	return nil
}

// single flusher goroutine. created flushes before the transition groups within the same tick/stop
func (b *postgresBatcher) run() {
	defer close(b.stoppedCh)
	ticker := time.NewTicker(b.flushEvery)
	defer ticker.Stop()

	for {
		select {
		case group := <-b.flushNow:
			b.flushGroup(group)
		case <-ticker.C:
			b.flushGroup(batchGroupCreated)
			b.flushGroup(batchGroupConfirmed)
			b.flushGroup(batchGroupFailed)
		case <-b.stopCh:
			b.flushGroup(batchGroupCreated)
			b.flushGroup(batchGroupConfirmed)
			b.flushGroup(batchGroupFailed)
			return
		}
	}
}

// take swaps out the group's buffer so the flush runs outside the lock.
func (b *postgresBatcher) take(group batchGroup) []batchItem {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch group {
	case batchGroupCreated:
		items := b.created
		b.created = nil
		return items
	case batchGroupConfirmed:
		items := b.confirmed
		b.confirmed = nil
		return items
	case batchGroupFailed:
		items := b.failed
		b.failed = nil
		return items
	}
	return nil
}

// flushGroup writes one group as a single transaction and settles every buffered delivery
func (b *postgresBatcher) flushGroup(group batchGroup) {
	items := b.take(group)
	if len(items) == 0 {
		return
	}

	reqs := make([]*models.BookingRequest, len(items))
	confirmedAts := make([]time.Time, len(items))
	for i, item := range items {
		reqs[i] = item.req
		confirmedAts[i] = item.confirmedAt
	}

	// The flush outlives any single message ctx (and the shutdown ctx).
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	start := time.Now()

	var (
		transitioned int64
		err          error
	)
	switch group {
	case batchGroupCreated:
		err = b.store.UpsertBookingCreatedBatch(ctx, reqs)
	case batchGroupConfirmed:
		transitioned, err = b.store.ApplyBookingConfirmedBatch(ctx, reqs, confirmedAts)
	case batchGroupFailed:
		transitioned, err = b.store.ApplyBookingFailedBatch(ctx, reqs)
	}
	cancel()

	if err != nil {
		b.logger.WithError(err).WithFields(logrus.Fields{
			"group":      string(group),
			"batch_size": len(items),
		}).Error("batch projection flush failed, backing off and requeueing")
		// The backoff keeps a postgres outage from spinning a hot requeue
		time.Sleep(b.backoff)
		for _, item := range items {
			if nackErr := item.delivery.Nack(false, true); nackErr != nil {
				b.logger.WithError(nackErr).Error("batch requeue nack failed")
			}
		}
		return
	}

	fields := logrus.Fields{
		"group":      string(group),
		"batch_size": len(items),
		"flush_ms":   time.Since(start).Milliseconds(),
	}
	if group != batchGroupCreated && transitioned != int64(len(items)) {
		fields["skipped"] = int64(len(items)) - transitioned
		b.logger.WithFields(fields).Info("batch projection flush: some rows already past PENDING (replay or late event)")
	} else {
		b.logger.WithFields(fields).Info("batch projection flush committed")
	}
	for _, item := range items {
		if ackErr := item.delivery.Ack(false); ackErr != nil {
			b.logger.WithError(ackErr).Error("batch ack failed")
		}
	}
}

// FlushAndStop drains any partial batch and stops the flusher
func (b *postgresBatcher) FlushAndStop(ctx context.Context) error {
	b.stopOnce.Do(func() { close(b.stopCh) })
	select {
	case <-b.stoppedCh:
		return nil
	case <-ctx.Done():
		b.mu.Lock()
		pending := len(b.created) + len(b.confirmed) + len(b.failed)
		b.mu.Unlock()
		return fmt.Errorf("batch drain timed out with %d unacked messages: %w", pending, ctx.Err())
	}
}
