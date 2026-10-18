package bookingCacheStore

import (
	"context"
	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
	"github.com/himanshu3889/ticketSphere-backend/base/keys"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
)

// Struct to hold get event results
type eventResult struct {
	event  *models.Event
	appErr *appError.Error
}

// Get Event details from cache or DB
func (cm *CacheStore) GetEvent(ctx context.Context, eventID snowflake.ID) (*models.Event, *appError.Error) {
	cacheKey := "event:" + eventID.String()

	// Singleflight
	result, _, _ := cm.Store.RequestGroup.Do(cacheKey, func() (interface{}, error) {
		event := &models.Event{ID: eventID}
		eventCacheBoundedKey := keys.EventCacheBoundedKey()
		eventCacheId := keys.EventCacheIDKey(eventID)
		eventBloomIdBoundedKey := keys.EventBloomFilterBoundedKey()
		eventBloomItemIDKey := keys.EventBloomItemIDKey(eventID)

		cacheState, err := cm.CacheManager.GetInto(ctx, eventCacheBoundedKey, eventCacheId, &eventBloomIdBoundedKey, &eventBloomItemIDKey, event)
		if err != nil {
			cm.logger.WithError(err).WithFields(logrus.Fields{"eventID": eventID}).Error("Failed to get event from cache")
			return eventResult{event: nil, appErr: appError.NewInternal("Failed to get event from cache")}, nil
		}

		if cacheState == redisDatabase.CacheHit {
			return eventResult{event: event, appErr: nil}, nil
		}

		if cacheState == redisDatabase.CacheBloomBlock || cacheState == redisDatabase.CacheNullBlock {
			return eventResult{event: nil, appErr: appError.NewNotFound("Invalid Event")}, nil
		}

		// Search in db and add to cache synchronously (to protect DB from thundering herd)
		dbEvent, appErr := cm.Store.GetEvent(ctx, eventID)
		if appErr != nil {
			return eventResult{event: nil, appErr: appErr}, nil
		}

		// set data in cache synchronously, if error log it (not return error)
		eventTTL := keys.EventCacheIDKeyTTL()
		err = cm.CacheManager.Set(ctx, eventCacheId, &eventBloomIdBoundedKey, &dbEvent, &eventBloomItemIDKey, eventTTL)
		if err != nil {
			cm.logger.WithError(err).WithFields(logrus.Fields{"eventID": eventID}).Error("Failed to set event in cache")
		}

		return eventResult{event: dbEvent, appErr: nil}, nil
	})

	// Unpack the result, every waiting goroutine will receive this exact same struct
	typedResult := result.(eventResult)
	return typedResult.event, typedResult.appErr
}

// Get Event's Seats from redis if not exist search in the postgres
func (cm *CacheStore) GetEventSeats(ctx context.Context, eventID snowflake.ID, seatIDs []snowflake.ID) ([]*models.Seat, *appError.Error) {
	// Search in bloom filter first for all seatIds
	// If some not found then invalid request
	bloomSeatsIdBoundedKey := keys.SeatBloomFilterBoundedKey()
	bloomSeatsItemIdKey := make([]string, len(seatIDs))
	for i, seatID := range seatIDs {
		bloomSeatsItemIdKey[i] = keys.SeatBloomItemIDKey(seatID)
	}

	// One pipelined flush: bloom sweep (existence) + MGET (values).
	cacheBoundedKey := keys.SeatCacheBoundedKey()

	cacheKeys := make([]string, len(seatIDs))
	seats := make([]*models.Seat, len(cacheKeys))
	dests := make([]any, len(seats))
	for i, seatID := range seatIDs {
		cacheKeys[i] = keys.SeatCacheIDKey(seatID)
		seat := &models.Seat{}
		seats[i] = seat
		dests[i] = seat
	}

	matches, cacheStates, err := cm.CacheManager.MGetIntoWithBloom(ctx, bloomSeatsIdBoundedKey, bloomSeatsItemIdKey, cacheBoundedKey, cacheKeys, dests)
	if err != nil {
		cm.logger.WithError(err).WithFields(logrus.Fields{"eventID": eventID, "seatIDs": seatIDs}).Error("Failed to fetch seats (bloom+cache pipeline)")
		return nil, appError.NewInternal("Failed to get seats from cache")
	}

	// If any is false in matches, then invalid request
	for i, match := range matches {
		if !match {
			return nil, appError.NewNotFound(fmt.Sprintf("Invalid Seat %d", seatIDs[i]))
		}
	}

	// Check valid seats
	missedSeatsCount := 0
	for i := range seatIDs {
		// states are keyed by cache key, not seat id
		cacheState := cacheStates[cacheKeys[i]]
		if cacheState == redisDatabase.CacheNullBlock || cacheState == redisDatabase.CacheBloomBlock {
			return nil, appError.NewNotFound(fmt.Sprintf("Seat %d Not Found", seatIDs[i]))
		}
		if cacheState == redisDatabase.CacheMiss {
			missedSeatsCount++
			continue
		}
		if seats[i].EventID != eventID {
			return nil, appError.NewBadRequest(fmt.Sprintf("Invalid request for seat %s", seatIDs[i]))
		}
		if seats[i].Status != models.SeatStatusAvailable {
			return nil, appError.NewConflict(fmt.Sprintf("Seat %d is not available", seatIDs[i]))
		}
	}

	// If all are found in cache
	if missedSeatsCount == 0 {
		return seats, nil
	}

	// Add the missed ids
	missedSeatIDs := make([]snowflake.ID, 0, missedSeatsCount)
	for i, seatID := range seatIDs {
		cacheState := cacheStates[cacheKeys[i]]
		if cacheState == redisDatabase.CacheMiss {
			missedSeatIDs = append(missedSeatIDs, seatID)
		}
	}

	// Seats missed from the cache
	// Fetch seats from db
	seatsFromDb, appErr := cm.Store.GetEventSeatsByIds(ctx, eventID, missedSeatIDs)
	if appErr != nil {
		return nil, appErr
	}

	// Add into the key value and bloom filter asynchronously
	// FIXME: Too many requests trying to write at once causing thundering herd
	go func() {
		ctxSet := context.WithoutCancel(ctx)

		// Add all to bloom filter
		bloomItems := make([]string, len(seatsFromDb))
		for i, seat := range seatsFromDb {
			bloomItems[i] = keys.SeatBloomItemIDKey(seat.ID)
		}
		cm.BloomManager.AddMulti(ctxSet, bloomSeatsIdBoundedKey, bloomItems)

		// Pair each seat with its own key — rows from id = ANY($2) come back in
		// arbitrary order, so never trust request order here.
		seatTTL := keys.SeatCacheIDKeyTTL()
		seatCacheKeys := make([]string, len(seatsFromDb))
		seatCacheData := make([]interface{}, len(seatsFromDb))
		for i, seat := range seatsFromDb {
			seatCacheKeys[i] = keys.SeatCacheIDKey(seat.ID)
			seatCacheData[i] = seat
		}
		if err := cm.CacheManager.SetMulti(ctxSet, seatCacheKeys, seatCacheData, seatTTL); err != nil {
			cm.logger.WithError(err).WithFields(logrus.Fields{"seatIDs": missedSeatIDs}).Error("Failed to warm seat cache")
		}
	}()

	// If some no found return error
	if len(seatsFromDb) != missedSeatsCount {
		return nil, appError.NewNotFound("Invalid Seat")
	}

	// Build all seats
	seatsFromDBMap := make(map[string]*models.Seat)
	for _, seat := range seatsFromDb {
		seatsFromDBMap[seat.ID.String()] = seat
	}

	for i, seatID := range seatIDs {
		if cacheStates[cacheKeys[i]] == redisDatabase.CacheMiss {
			seats[i] = seatsFromDBMap[seatID.String()]
		}
	}
	return seats, nil
}

// warmSeatInfoKeys writes the seat info cache synchronously.
// Cold path only: the booking Lua script returned PRICE_UNAVAILABLE, meaning the
// info keys were evicted/flushed while the bloom filter still knew the seats
// (or the event predates cache warming). The script reads prices straight out
// of this JSON, so re-warming the info cache is the whole fix — no separate
// price key to keep in sync.
func (cm *CacheStore) warmSeatInfoKeys(ctx context.Context, seats []*models.Seat) error {
	seatCacheKeys := make([]string, len(seats))
	seatData := make([]interface{}, len(seats))
	for i, seat := range seats {
		seatCacheKeys[i] = keys.SeatCacheIDKey(seat.ID)
		seatData[i] = seat
	}
	return cm.CacheManager.SetMulti(ctx, seatCacheKeys, seatData, keys.SeatCacheIDKeyTTL())
}

// Get Event seats total price
func (cm *CacheStore) GetEventSeatsTotalPrice(ctx context.Context, eventID snowflake.ID, seatIDs []snowflake.ID) (totalPrice int64, appErr *appError.Error) {
	// Start a custom span!
	ctx, span := cm.Tracer.Start(ctx, constants.SpanEventSeatsTotalPrice)
	defer utils.EndSpanAppErr(span, appErr)

	// Add attributes so you can search by EventID in Grafana
	span.SetAttributes(
		attribute.String("event.id", eventID.String()),
		attribute.Int("seats.count", len(seatIDs)),
	)

	// Get event from redis if not exist search in the postgres
	_, appErr = cm.GetEvent(ctx, eventID)
	if appErr != nil {
		return 0, appErr
	}

	// Get Event's Seats from redis if not exist search in the postgres
	eventSeats, appErr := cm.GetEventSeats(ctx, eventID, seatIDs)
	if appErr != nil {
		return 0, appErr
	}

	if len(eventSeats) != len(seatIDs) {
		return 0, appError.NewNotFound("Invalid Seat")
	}

	// Calculate total price
	for _, seat := range eventSeats {
		totalPrice += seat.Price
	}

	// Return total price
	return totalPrice, nil
}
