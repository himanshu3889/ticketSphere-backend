package bookingStore

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
	"github.com/sirupsen/logrus"
)

// Get Event seats by IDs
func (store *Store) GetEventSeatsByIds(ctx context.Context, eventID snowflake.ID, seatIDs []snowflake.ID) ([]*models.Seat, *appError.Error) {
	if len(seatIDs) == 0 {
		return nil, appError.NewBadRequest("No seat ids provided")
	}

	if len(seatIDs) > 100 {
		return nil, appError.NewBadRequest("Too many seat ids provided")
	}

	query := `SELECT * FROM event_seats 
				WHERE event_id=$1 and id =ANY($2)`

	var eventSeats []*models.Seat
	err := store.PostgresDB.SelectContext(ctx, &eventSeats, query, eventID, seatIDs)
	if err != nil {
		store.logger.WithError(err).
			WithFields(logrus.Fields{"eventID": eventID, "seatIDs": seatIDs}).
			Error("Failed to fetch event seats")
		return nil, appError.NewInternal("Failed to get event seats")
	}

	// Handle zero rows manually (SelectContext doesn't return ErrNoRows)
	if len(eventSeats) == 0 {
		return nil, appError.NewNotFound("No event seats found")
	}

	return eventSeats, nil
}
