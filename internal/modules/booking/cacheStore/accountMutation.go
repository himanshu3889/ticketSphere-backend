package bookingCacheStore

import (
	"context"

	"github.com/himanshu3889/ticketSphere-backend/base/keys"
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

	// async Set cache (detached from request lifecycle)
	go func() {
		bgCtx := context.WithoutCancel(ctx)
		err := cacheStore.CacheManager.Set(bgCtx, keys.UserCacheIDKey(user.ID), nil, user, nil, 0)
		if err != nil {
			cacheStore.logger.WithError(err).Debug("Error setting the user info in cache")
		}
		// Add bloom keys for user -
		// User Id
		cacheStore.BloomManager.Add(bgCtx, keys.UserBloomFilterBoundedKey(), keys.UserBloomItemIDKey(user.ID))
		// User email
		cacheStore.BloomManager.Add(bgCtx, keys.UserEmailBloomFilterBoundedKey(), keys.UserBloomItemEmailKey(user.Email))
	}()

	return nil
}
