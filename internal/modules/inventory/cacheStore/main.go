package iventoryCacheStore

import (
	"errors"

	redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	invntoryStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/inventory/store"
	"github.com/sirupsen/logrus"
)

type CacheStore struct {
	Store        *invntoryStore.Store
	CacheManager *redisDatabase.CacheManager
	BloomManager *bloomFilter.BloomManager
	logger       *logrus.Logger
}

func New(store *invntoryStore.Store,
	cacheManager *redisDatabase.CacheManager,
	bloomManager *bloomFilter.BloomManager,
	logger *logrus.Logger,
) (*CacheStore, error) {
	if cacheManager == nil {
		return nil, errors.New("Cache manager is nil")
	}
	if bloomManager == nil {
		return nil, errors.New("Bloom manager is nil")
	}
	if logger == nil {
		return nil, errors.New("Logger is nil")
	}
	return &CacheStore{Store: store, CacheManager: cacheManager, BloomManager: bloomManager, logger: logger}, nil
}
