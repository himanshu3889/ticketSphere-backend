package redisDatabase

import (
	"context"
	"fmt"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/ticketSphere-backend/configs"
	"go.opentelemetry.io/otel/trace"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/redis/go-redis/extra/redisotel/v9"
)

// Initialize redis
func InitRedis(redisConfigs *configs.RedisConfigurations, logger *logrus.Logger, traceProvider trace.TracerProvider) (error, *redis.Client, *CacheManager, *bloomFilter.BloomManager) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s",
			redisConfigs.Host,
			redisConfigs.Port),
		Password:     redisConfigs.Password,
		DB:           redisConfigs.DB,
		PoolSize:     redisConfigs.PoolSize,
		MinIdleConns: redisConfigs.MinIdleConns,
	})

	// Instrument the client with OpenTelemetry hooks
	if err := redisotel.InstrumentTracing(redisClient); err != nil {
		logger.WithError(err).Fatal("Failed to instrument Redis client with OpenTelemetry")
		return err, nil, nil, nil
	}

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
	cacheManager := initCacheManager(redisClient, bloomManager, logger, traceProvider)

	logger.Info("Redis database connected successfully (Instrumented with Otel)")
	return nil, redisClient, cacheManager, bloomManager
}

func initBloomManager(bloomFilterConfigs []configs.BloomFilterConfig, redisClient *redis.Client, logger *logrus.Logger) (*bloomFilter.BloomManager, error) {
	if len(bloomFilterConfigs) == 0 {
		logger.Fatal("no bloom filter configs found in yaml")
	}

	bloom, err := bloomFilter.NewBloomManager(redisClient, bloomFilterConfigs, logger)
	if err != nil {
		logger.WithError(err).Fatal("unable to initialize bloom filter")
		return nil, err
	}

	logger.WithField("count", len(bloomFilterConfigs)).Info("Bloom filter initialized")
	return bloom, nil
}

func initCacheManager(redisClient *redis.Client, bloomManager *bloomFilter.BloomManager, logger *logrus.Logger, traceProvider trace.TracerProvider) *CacheManager {
	cacheManager, err := NewCacheManager(redisClient, bloomManager, logger, traceProvider)
	if err != nil {
		logger.WithError(err).Fatal("unable to initialize cache manager")
	}
	logger.Info("Global cache initialized")
	return cacheManager
}

func CloseRedisConnection(db *redis.Client) error {
	err := db.Close()
	return err
}
