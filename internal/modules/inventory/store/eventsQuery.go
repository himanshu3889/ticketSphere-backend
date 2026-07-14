package inventoryStore

import (
	"database/sql"
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
)

// Get event
func (store *Store) GetEvent(ctx *gin.Context, eventID snowflake.ID) (*models.Event, *appError.Error) {
	query := `SELECT * FROM events 
				WHERE id=$1`
	var event models.Event
	err := store.PostgresDB.GetContext(ctx, &event, query, eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appError.NewNotFound("No event found")
		}
		store.logger.WithError(err).Error("Failed to fetch event")
		return nil, appError.NewInternal("Failed to get event")
	}
	return &event, nil
}

// Get venue latest events
func (store *Store) GetVenueEvents(ctx *gin.Context, venue_id snowflake.ID, limit int, cursor snowflake.ID) ([]*models.Event, *appError.Error) {
	limit = min(200, max(10, limit)) // 10 to 200
	query := `SELECT * FROM events 
				WHERE venue_id=$1 AND id < $2
				ORDER BY id DESC 
			LIMIT $3`
	var events []*models.Event
	err := store.PostgresDB.SelectContext(ctx, &events, query, venue_id, cursor, limit)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*models.Event{}, nil
		}
		store.logger.WithError(err).Error("Failed to fetch venue events")
		return nil, appError.NewInternal("Failed to get event veneus")
	}
	return events, nil
}

// Get all latest events
func (store *Store) GetAllEvents(ctx *gin.Context, limit int, cursor snowflake.ID) ([]*models.Event, *appError.Error) {
	limit = min(200, max(10, limit)) // 10 to 200
	if cursor == 0 {
		cursor = utils.GetMaxSnowflakeID()
	}
	query := `SELECT * FROM events 
			WHERE id < $1
			ORDER BY id DESC 
			LIMIT $2`
	var events = make([]*models.Event, 0)
	err := store.PostgresDB.SelectContext(ctx, &events, query, cursor, limit)
	if err != nil {
		store.logger.WithError(err).Error("Failed to fetch events")
		return nil, appError.NewInternal("Failed to get events")
	}
	if len(events) == 0 {
		return []*models.Event{}, nil
	}
	return events, nil
}

// Get upcoming or live events
func (store *Store) GetUpcomingOrLiveEvents(ctx *gin.Context, limit int, cursor snowflake.ID) ([]*models.Event, *appError.Error) {
	limit = min(200, max(10, limit)) // 10 to 200
	if cursor == 0 {
		cursor = utils.GetMaxSnowflakeID()
	}
	query := `SELECT * FROM events 
				WHERE (start_time >= NOW() OR end_time >= NOW())
					AND status NOT IN ('COMPLETED', 'CANCELLED') AND id < $1
			ORDER BY id DESC 
			LIMIT $2`
	var events []*models.Event
	err := store.PostgresDB.SelectContext(ctx, &events, query, cursor, limit)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*models.Event{}, nil
		}
		store.logger.WithError(err).Error("Failed to fetch events")
		return nil, appError.NewInternal("Failed to get events")
	}
	return events, nil
}

// Get venue upcoming or live events
func (store *Store) GetVenueUpcomingOrLiveEvents(ctx *gin.Context, venue_id snowflake.ID, limit int, cursor snowflake.ID) ([]*models.Event, *appError.Error) {
	limit = min(200, max(10, limit)) // 10 to 200
	if cursor == 0 {
		cursor = utils.GetMaxSnowflakeID()
	}
	// query is build according to the indexing
	query := `SELECT * FROM events 
				WHERE (start_time >= NOW() OR end_time >= NOW())
					AND status NOT IN ('COMPLETED', 'CANCELLED') AND venue_id=$1 AND id < $2
			ORDER BY id DESC 
			LIMIT $3`
	var events []*models.Event
	err := store.PostgresDB.SelectContext(ctx, &events, query, venue_id, cursor, limit)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*models.Event{}, nil
		}
		store.logger.WithError(err).Error("Failed to fetch venue events")
		return nil, appError.NewInternal("Failed to get venue events")
	}
	return events, nil
}
