package inventoryStore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
	"github.com/sirupsen/logrus"
)

// Get section by ID
func (store *Store) GetSectionByID(ctx context.Context, section_id snowflake.ID) (*models.Section, *appError.Error) {
	query := `SELECT * FROM sections
				WHERE id=$1`
	var section models.Section
	err := store.PostgresDB.GetContext(ctx, &section, query, section_id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appError.NewNotFound("No section found")
		}
		store.logger.WithFields(logrus.Fields{"section_id": section_id}).WithError(err).Error("Failed to fetch section by ID")
		return nil, appError.NewInternal("Failed to get section by ID")
	}
	return &section, nil
}

// Get event all sections
func (store *Store) GetEventAllSections(ctx context.Context, event_id snowflake.ID, limit int, cursor snowflake.ID) ([]*models.Section, *appError.Error) {
	limit = min(200, max(10, limit)) // 10 to 200
	if cursor == 0 {
		cursor = utils.GetMaxSnowflakeID()
	}
	query := `SELECT * FROM sections
				WHERE event_id=$1 AND id < $2
			ORDER BY id DESC
			LIMIT $3`
	var sections = make([]*models.Section, 0)
	err := store.PostgresDB.SelectContext(ctx, &sections, query, event_id, cursor, limit)
	if err != nil {
		store.logger.WithFields(logrus.Fields{"event_id": event_id}).WithError(err).Error("Failed to fetch event sections")
		return nil, appError.NewInternal("Failed to get event sections")
	}
	if len(sections) == 0 {
		return []*models.Section{}, nil
	}
	return sections, nil
}
