package bookingCacheStore

import (
	"context"

	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	rediskeys "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/redisKeys"
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
	cacheKey, _ := rediskeys.Keys.User.Info(user.ID)

	go func() {
		err := cacheStore.CacheManager.Set(ctx, cacheKey, nil, user, nil, 0)
		if err != nil {
			cacheStore.logger.WithError(err).Debug("Error setting the user info in cache")
		}
		// Add bloom keys for user -
		// User Id
		bloomUserEmailKey := bloomFilter.UserEmailBloomFilter
		bloomUserIdKey := bloomFilter.UserIDBloomFilter
		bloomUserIdItem := user.ID.String()
		cacheStore.BloomManager.Add(ctx, bloomUserIdKey, bloomUserIdItem)
		// User email
		bloomUserEmailItem := user.Email
		cacheStore.BloomManager.Add(ctx, bloomUserEmailKey, bloomUserEmailItem)
	}()

	return nil
}
