package bookingStore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
)

// Get event
func (store *Store) GetEvent(ctx context.Context, eventId snowflake.ID) (*models.Event, *appError.Error) {
	query := `SELECT * FROM events 
				WHERE id=$1`
	var event models.Event
	err := store.PostgresDB.GetContext(ctx, &event, query, eventId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appError.NewNotFound("No event found")
		}
		store.logger.WithError(err).Error("Failed to fetch event")
		return nil, appError.NewInternal("Failed to get event")
	}
	return &event, nil
}
