package bookingCacheStore

import (
	"context"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
)

// Create user by write back cache strategy
func (cacheStore *CacheStore) CreateUser(ctx context.Context, user *models.User) *appError.Error {
	// DB create user
	appErr := cacheStore.Store.CreateUser(ctx, user)
	if appErr != nil {
		return appErr
	}

	// async Set cache
	go func() {
		err := cacheStore.CacheManager.Set(ctx, user.CacheIDKey(), nil, user, nil, 0)
		if err != nil {
			cacheStore.logger.WithError(err).Debug("Error setting the user info in cache")
		}
		// Add bloom keys for user -
		// User Id
		cacheStore.BloomManager.Add(ctx, user.BloomFilterIDBoundedKey(), user.BloomItemIDKey())
		// User email
		cacheStore.BloomManager.Add(ctx, user.BloomFilterEmailBoundedKey(), user.BloomItemEmailKey())
	}()

	return nil
}
