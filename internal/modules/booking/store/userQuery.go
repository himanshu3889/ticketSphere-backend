package bookingStore

import (
	"context"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"

	"github.com/bwmarrin/snowflake"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Fetches multiple users in a single query; max Limit 100
func (store *Store) GetUsersBatch(ctx context.Context, userIDs []snowflake.ID) (map[snowflake.ID]*models.User, *appError.Error) {
	if len(userIDs) > 100 {
		return nil, appError.NewBadRequest("Max user batching is 100")
	}
	if len(userIDs) == 0 {
		return map[snowflake.ID]*models.User{}, nil
	}

	query := `
		SELECT id, username, email, name, image_url
		FROM users
		WHERE id = ANY($1)
	`

	// Convert slice to pq.Array for PostgreSQL
	var users []*models.User
	err := store.PostgresDB.SelectContext(ctx, &users, query, pq.Array(userIDs))
	if err != nil {
		store.logger.WithFields(logrus.Fields{
			"user_ids_len": len(userIDs),
		}).WithError(err).Error("Failed to fetch users batch")

		return nil, appError.NewInternal("failed to fetch users")
	}

	// Convert to map for O(1) lookups
	userMap := make(map[snowflake.ID]*models.User, len(users))
	for _, user := range users {
		userMap[user.ID] = user
	}

	return userMap, nil
}
