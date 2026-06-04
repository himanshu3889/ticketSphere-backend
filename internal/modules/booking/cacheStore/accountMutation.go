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
	// Write Back cache

	// DB
	appErr := cacheStore.Store.CreateUser(ctx, user)
	if appErr != nil {
		return appErr
	}

	// async Set cache
	cacheKey, _ := rediskeys.Keys.User.Info(user.ID)
	bloomKey := bloomFilter.UserIDBloomFilter
	bloomItem := user.ID.String()

	go func() {
		err := cacheStore.CacheManager.Set(ctx, cacheKey, &bloomKey, user, &bloomItem, 0)
		if err != nil {
			// set cache error
		}
	}()

	return nil
}
