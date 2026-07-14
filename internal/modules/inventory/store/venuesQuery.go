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

// Get all venues
func (store *Store) GetVenues(ctx *gin.Context, limit int, cursor snowflake.ID) ([]*models.Venue, *appError.Error) {
	limit = min(200, max(10, limit)) // 10 to 200
	if cursor == 0 {
		cursor = utils.GetMaxSnowflakeID()
	}
	query := `SELECT * FROM venues 
				WHERE id < $1
			ORDER BY id DESC 
			LIMIT $2`
	venues := make([]*models.Venue, 0)
	err := store.PostgresDB.SelectContext(ctx, &venues, query, cursor, limit)
	if err != nil {
		store.logger.WithError(err).Error("Failed to fetch venues")
		return nil, appError.NewInternal("Failed to get venues")
	}
	if len(venues) == 0 {
		return []*models.Venue{}, nil
	}
	return venues, nil
}

// Get venue by ID
func (store *Store) GetVenue(ctx *gin.Context, venue_id snowflake.ID) (*models.Venue, *appError.Error) {
	query := `SELECT * FROM venues 
				WHERE id=$1`
	var venue models.Venue
	err := store.PostgresDB.GetContext(ctx, &venue, query, venue_id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appError.NewNotFound("No venue found")
		}
		store.logger.WithField("venue_id", venue_id).WithError(err).Error("Failed to fetch venue")
		return nil, appError.NewInternal("Failed to get venue")
	}
	return &venue, nil
}
