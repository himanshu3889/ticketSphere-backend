package bookingCacheStore

import (
	"context"

	"github.com/bwmarrin/snowflake"
	redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
)

// Get user by Id
func (cacheStore *CacheStore) GetUserById(ctx context.Context, userId snowflake.ID) (*models.User, *appError.Error) {
	// Cache Aside pattern
	user := &models.User{ID: userId}

	bloomUserIdKey := user.BloomFilterIDBoundedKey()
	bloomItemUserIdKey := user.BloomItemIDKey()

	userCacheBoundedKey := user.CacheBoundedKey()
	userCacheIdKey := user.CacheIDKey()
	userCacheIdKeyTTL := user.CacheIDKeyTTL()

	cacheState, err := cacheStore.CacheManager.GetInto(ctx, userCacheBoundedKey, userCacheIdKey, &bloomUserIdKey, &bloomItemUserIdKey, user)
	if err != nil {
		return nil, appError.NewInternal("Error to find user")
	}
	if cacheState == redisDatabase.CacheBloomBlock || cacheState == redisDatabase.CacheNullBlock {
		return nil, appError.NewNotFound("User not found")
	}

	// Find in db
	user, appErr := cacheStore.Store.GetUserByID(ctx, userId)
	if appErr != nil {
		return nil, appErr
	}

	// Set key in cache async
	// FIXME: Too many requests trying to write at once causing thundering herd
	go func() {
		ctxSet := context.Background()
		cacheStore.CacheManager.Set(ctxSet, userCacheIdKey, &bloomUserIdKey, user, &bloomItemUserIdKey, userCacheIdKeyTTL)
	}()

	return user, nil
}

// Get user by Email
func (cacheStore *CacheStore) GetUserByEmail(ctx context.Context, email string) (*models.User, *appError.Error) {
	// Bloom filter to check if the user exist by email or not
	bloomUserEmailKey := bloomFilter.UserEmailBloomFilter
	bloomContain, err := cacheStore.BloomManager.MightContain(ctx, bloomUserEmailKey, email)
	if err != nil {
		cacheStore.logger.WithError(err).Debug("Error in checking user email in bloom filter")
		return nil, appError.NewInternal("Error during find user by email")
	}
	if !bloomContain {
		// NOTE: for development purpose, we can return "User does not exist" error
		// But for production, we should return a generic error to prevent user enumeration
		return nil, appError.NewNotFound("User does not exist")
	}
	return cacheStore.Store.GetUserByEmail(ctx, email)
}
