package bookingCacheStore

import (
	"context"
	"fmt"

	"github.com/bwmarrin/snowflake"
	redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
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
		eventCacheBoundedKey := event.CacheBoundedKey()
		eventCacheId := event.CacheIDKey()
		eventBloomIdBoundedKey := event.BloomFilterIDBoundedKey()
		eventBloomItemIDKey := event.BloomItemIDKey()

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
		eventTTL := dbEvent.CacheIDKeyTTL()
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
	tempSeat := models.Seat{}
	bloomSeatsIdBoundedKey := tempSeat.BloomFilterIDBoundedKey()
	bloomSeatsItemIdKey := make([]string, len(seatIDs))
	for i, seatID := range seatIDs {
		bloomSeatsItemIdKey[i] = tempSeat.BloomItemIDKeySeat(seatID)
	}

	matches, err := cm.BloomManager.CheckMulti(ctx, bloomSeatsIdBoundedKey, bloomSeatsItemIdKey)
	if err != nil {
		cm.logger.WithError(err).WithFields(logrus.Fields{"eventID": eventID, "seatIDs": seatIDs}).Error("Failed to check seats in bloom filter")
		return nil, appError.NewInternal("Failed to check seats in bloom filter")
	}

	// If any is false in matches, then invalid request
	for i, match := range matches {
		if !match {
			return nil, appError.NewNotFound(fmt.Sprintf("Invalid Seat %d", seatIDs[i]))
		}
	}

	// Search in redis key value set
	cacheBoundedKey := tempSeat.CacheBoundedKey()

	cacheKeys := make([]string, len(seatIDs))
	seats := make([]*models.Seat, len(cacheKeys))
	dests := make([]any, len(seats))
	for i, seatID := range seatIDs {
		cacheKeys[i] = tempSeat.CacheIDKeySeat(seatID)
		seat := &models.Seat{}
		seats[i] = seat
		dests[i] = seat
	}

	cacheStates, err := cm.CacheManager.MGetInto(ctx, cacheBoundedKey, cacheKeys, dests)
	if err != nil {
		cm.logger.WithError(err).WithFields(logrus.Fields{"eventID": eventID, "seatIDs": seatIDs}).Error("Failed to get seats from cache")
		return nil, appError.NewInternal("Failed to get seats from cache")
	}

	// Check valid seats
	missedSeatsCount := 0
	for i, seatID := range seatIDs {
		cacheState := cacheStates[seatID.String()]
		if cacheState == redisDatabase.CacheNullBlock || cacheState == redisDatabase.CacheBloomBlock {
			return nil, appError.NewNotFound(fmt.Sprintf("Seat %d Not Found", seatID))
		}
		if seats[i].EventID != eventID {
			return nil, appError.NewBadRequest(fmt.Sprintf("Invalid request for seat %s", seatID))
		}
		if seats[i].Status != models.SeatStatusAvailable {
			return nil, appError.NewConflict(fmt.Sprintf("Seat %d is not available", seatID))
		}

		if cacheState == redisDatabase.CacheMiss {
			missedSeatsCount++
		}
	}

	// If all are found in cache
	if missedSeatsCount == 0 {
		return seats, nil
	}

	// Add the missed ids
	missedSeatIDs := make([]snowflake.ID, 0, missedSeatsCount)
	for _, seatID := range seatIDs {
		cacheState := cacheStates[seatID.String()]
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
		ctxSet := context.Background()

		// Add all to bloom filter
		bloomItems := make([]string, len(seatsFromDb))
		for i, seat := range seatsFromDb {
			bloomItems[i] = seat.BloomItemIDKeySeat(seat.ID)
		}
		cm.BloomManager.AddMulti(ctxSet, bloomSeatsIdBoundedKey, bloomItems)

		// Add all to key value set
		seatTTL := tempSeat.CacheIDKeyTTL()
		seatCacheData := make([]interface{}, len(seatsFromDb))
		for i := range seatsFromDb {
			seatCacheData[i] = seatsFromDb[i]
		}
		cm.CacheManager.SetMulti(ctxSet, cacheKeys, seatCacheData, seatTTL)
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
		if cacheStates[seatID.String()] == redisDatabase.CacheMiss {
			seats[i] = seatsFromDBMap[seatID.String()]
		}
	}
	return seats, nil
}

// Get Event seats total price
func (cm *CacheStore) GetEventSeatsTotalPrice(ctx context.Context, eventID snowflake.ID, seatIDs []snowflake.ID) (totalPrice float64, appErr *appError.Error) {
	// Start a custom span!
	ctx, span := cm.Tracer.Start(ctx, "GetEventSeatsTotalPrice")
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
