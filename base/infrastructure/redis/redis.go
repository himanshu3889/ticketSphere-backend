package redisDatabase

import (
	"context"
	"fmt"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/ticketSphere-backend/configs"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Initialize redis
func InitRedis(redisConfigs *configs.RedisConfigurations, logger *logrus.Logger) (error, *redis.Client, *CacheManager, *bloomFilter.BloomManager) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var redisClient *redis.Client
	redisClient = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s",
			redisConfigs.Host,
			redisConfigs.Port),
		Password:     redisConfigs.Password,
		DB:           redisConfigs.DB,
		PoolSize:     redisConfigs.PoolSize,
		MinIdleConns: redisConfigs.MinIdleConns,
	})

	// Test connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.WithError(err).Fatal("Redis connection failed")
		return err, nil, nil, nil
	}

	// Initialize others
	bloomManager, err := initBloomManager(redisConfigs.BloomFilters, redisClient, logger)
	if err != nil {
		return err, nil, nil, nil
	}
	cacheManager := initCacheManager(redisClient, bloomManager, logger)

	logger.Info("Redis database connected successfully")
	return nil, redisClient, cacheManager, bloomManager
}

func initBloomManager(bloomFilterConfigs []configs.BloomFilterConfig, redisClient *redis.Client, logger *logrus.Logger) (*bloomFilter.BloomManager, error) {
	if len(bloomFilterConfigs) == 0 {
		logger.Fatal("no bloom filter configs found in yaml")
	}

	bloom, err := bloomFilter.NewBloomManager(redisClient, bloomFilterConfigs)
	if err != nil {
		logger.WithError(err).Fatal("unable to initialize global bloom filter")
		return nil, err
	}

	logger.WithField("count", len(bloomFilterConfigs)).Info("Global bloom filter initialized")
	return bloom, nil
}

func initCacheManager(redisClient *redis.Client, bloomManager *bloomFilter.BloomManager, logger *logrus.Logger) *CacheManager {
	cacheManager := NewCacheManager(redisClient, bloomManager)
	logger.Info("Global cache initialized")
	return cacheManager
}

func CloseRedisConnection(db *redis.Client) error {
	err := db.Close()
	return err
}
