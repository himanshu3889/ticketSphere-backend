package inventoryStore

import (
	"database/sql"
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
	"github.com/sirupsen/logrus"
)

// Get seat by ID
func (store *Store) GetSeatByID(ctx *gin.Context, seat_id snowflake.ID) (*models.Seat, *appError.Error) {
	query := `SELECT * FROM seats 
				WHERE id=$1`
	var seat *models.Seat
	err := store.PostgresDB.GetContext(ctx, &seat, query, seat_id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appError.NewNotFound("No seat found")
		}
		store.logger.WithFields(logrus.Fields{"seat_id": seat_id}).WithError(err).Error("Failed to fetch seat by ID")
		return nil, appError.NewInternal("Failed to get seat by ID")
	}
	return seat, nil
}

// Get event available seats
func (store *Store) GetEventAvailableSeats(ctx *gin.Context, event_id snowflake.ID, limit int, cursor snowflake.ID) ([]*models.Seat, *appError.Error) {
	limit = min(200, max(10, limit)) // 10 to 200
	if cursor == 0 {
		cursor = utils.GetMaxSnowflakeID()
	}
	query := `SELECT * FROM seats 
				WHERE event_id=$1 AND status='AVAILABLE' AND id < $2
				ORDER BY id DESC 
				LIMIT $3`
	var seats = make([]*models.Seat, 0)
	err := store.PostgresDB.SelectContext(ctx, &seats, query, event_id, cursor, limit)
	if err != nil {
		store.logger.WithFields(logrus.Fields{"event_id": event_id}).WithError(err).Error("Failed to fetch event available seats")
		return nil, appError.NewInternal("Failed to get event available seats")
	}
	if len(seats) == 0 {
		return []*models.Seat{}, nil
	}
	return seats, nil
}

// Get event available or held seats
func (store *Store) GetEventAvailableOrHeldSeats(ctx *gin.Context, event_id snowflake.ID, limit int, cursor snowflake.ID) ([]*models.Seat, *appError.Error) {
	limit = min(200, max(10, limit)) // 10 to 200
	if cursor == 0 {
		cursor = utils.GetMaxSnowflakeID()
	}
	query := `SELECT * FROM seats 
				WHERE event_id=$1 AND status IN ('AVAILABLE', 'HELD') AND id < $2
				ORDER BY id DESC 
				LIMIT $3`
	var seats = make([]*models.Seat, 0)
	err := store.PostgresDB.SelectContext(ctx, &seats, query, event_id, cursor, limit)
	if err != nil {
		store.logger.WithFields(logrus.Fields{"event_id": event_id}).WithError(err).Error("Failed to fetch event available or held seats")
		return nil, appError.NewInternal("Failed to get event available or held seats")
	}
	if len(seats) == 0 {
		return []*models.Seat{}, nil
	}
	return seats, nil
}

// Get section available seats
func (store *Store) GetSectionAvailableSeats(ctx *gin.Context, section_id snowflake.ID, limit int, cursor snowflake.ID) ([]*models.Seat, *appError.Error) {
	limit = min(200, max(10, limit)) // 10 to 200
	if cursor == 0 {
		cursor = utils.GetMaxSnowflakeID()
	}
	query := `SELECT * FROM seats 
				WHERE section_id=$1 AND status='AVAILABLE' AND id < $2
				ORDER BY id DESC 
				LIMIT $3`
	var seats = make([]*models.Seat, 0)
	err := store.PostgresDB.SelectContext(ctx, &seats, query, section_id, cursor, limit)
	if err != nil {
		store.logger.WithFields(logrus.Fields{"section_id": section_id}).WithError(err).Error("Failed to fetch section available seats")
		return nil, appError.NewInternal("Failed to get section available seats")
	}
	if len(seats) == 0 {
		return []*models.Seat{}, nil
	}
	return seats, nil
}

// Get section available or held seats
func (store *Store) GetSectionAvailableOrHeldSeats(ctx *gin.Context, section_id snowflake.ID, limit int, cursor snowflake.ID) ([]*models.Seat, *appError.Error) {
	limit = min(200, max(10, limit)) // 10 to 200
	if cursor == 0 {
		cursor = utils.GetMaxSnowflakeID()
	}
	query := `SELECT * FROM seats 
				WHERE section_id=$1 AND status IN ('AVAILABLE', 'HELD') AND id < $2
				ORDER BY id DESC 
				LIMIT $3`
	var seats = make([]*models.Seat, 0)
	err := store.PostgresDB.SelectContext(ctx, &seats, query, section_id, cursor, limit)
	if err != nil {
		store.logger.WithFields(logrus.Fields{"section_id": section_id}).WithError(err).Error("Failed to fetch section available or held seats")
		return nil, appError.NewInternal("Failed to get section available or held seats")
	}
	if len(seats) == 0 {
		return []*models.Seat{}, nil
	}
	return seats, nil
}
