package bookingCacheStore

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
)

// Get user by Id
func (cacheStore *CacheStore) GetUserById(ctx context.Context, userId snowflake.ID) (*models.User, *appError.Error) {
	bloomUserIdKey := bloomFilter.UserIDBloomFilter
	bloomContain, err := cacheStore.BloomManager.MightContain(ctx, bloomUserIdKey, userId.String())
	if err != nil {
		cacheStore.logger.WithError(err).Debug("Error in checking user id in bloom filter")
		return nil, appError.NewInternal("Error during find user")
	}
	if !bloomContain {
		return nil, appError.NewNotFound("User does not exist")
	}
	return cacheStore.Store.GetUserByID(ctx, userId)
}

// Get user by Email
func (cacheStore *CacheStore) GetUserByEmail(ctx context.Context, email string) (*models.User, *appError.Error) {
	bloomUserEmailKey := bloomFilter.UserEmailBloomFilter
	bloomContain, err := cacheStore.BloomManager.MightContain(ctx, bloomUserEmailKey, email)
	if err != nil {
		cacheStore.logger.WithError(err).Debug("Error in checking user email in bloom filter")
		return nil, appError.NewInternal("Error during find user by the")
	}
	if !bloomContain {
		return nil, appError.NewNotFound("User does not exist")
	}
	return cacheStore.Store.GetUserByEmail(ctx, email)
}
