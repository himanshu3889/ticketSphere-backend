package bookingCacheStore

import (
	"context"
	"encoding/json"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/keys"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
)

// GetUsersBatch fetches users using the cache only
func (cacheStore *CacheStore) GetUsersBatch(ctx context.Context, userIDs []snowflake.ID) (map[snowflake.ID]*models.User, *appError.Error) {
	if len(userIDs) > 100 {
		return nil, appError.NewBadRequest("Max user batching is 100")
	}
	if len(userIDs) == 0 {
		return map[snowflake.ID]*models.User{}, nil
	}

	userMap := make(map[snowflake.ID]*models.User, len(userIDs))

	// Prepare Redis keys
	cacheKeys := make([]string, len(userIDs))
	idMap := make(map[string]snowflake.ID, len(userIDs)) // for mapping back

	for i, id := range userIDs {
		cacheKeys[i] = keys.UserCacheIDKey(id)
		idMap[cacheKeys[i]] = id
	}

	// Bulk fetch from Redis
	cacheBoundedKey := keys.UserCacheBoundedKey()
	cachedData, err := cacheStore.CacheManager.MGet(ctx, cacheBoundedKey, cacheKeys)
	if err != nil {
		cacheStore.logger.WithError(err).Error("Redis MGet failed")
		// If Redis fails, treat all as missing
	} else {
		for _, raw := range cachedData {
			if raw == nil {
				continue
			}

			// Unmarshal user
			var user models.User
			if err := json.Unmarshal(raw, &user); err != nil {
				// cacheStore.logger.WithError(err).Error("Unmarshall error")
				continue
			}

			userMap[user.ID] = &user
		}
	}

	return userMap, nil
}
