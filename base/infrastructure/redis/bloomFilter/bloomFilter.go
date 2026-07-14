package bloomFilter

import (
	"context"
	"errors"
	"fmt"

	baseMetrics "github.com/himanshu3889/ticketSphere-backend/base/metric"
	baseConfigs "github.com/himanshu3889/ticketSphere-backend/configs"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// bloomState is unexported; used strictly for internal Prometheus labels
type BloomState int

const (
	bloomMightExist BloomState = iota
	bloomDefinitelyNotExists
	bloomError
)

// String satisfies the label requirement for Prometheus
func (s BloomState) String() string {
	switch s {
	case bloomMightExist:
		return "might_exist"
	case bloomDefinitelyNotExists:
		return "definitely_not_exists"
	case bloomError:
		return "error"
	default:
		return ""
	}
}

// BloomManager handles all bloom filters centrally
type BloomManager struct {
	client  *redis.Client
	configs map[baseConfigs.BloomFilterKey]baseConfigs.BloomFilterConfig
}

// NewBloomManager creates a manager with predefined configurations
func NewBloomManager(client *redis.Client, configs []baseConfigs.BloomFilterConfig, logger *logrus.Logger) (*BloomManager, error) {
	if client == nil {
		return nil, errors.New("redis client is nil")
	}

	m := &BloomManager{
		client:  client,
		configs: make(map[baseConfigs.BloomFilterKey]baseConfigs.BloomFilterConfig),
	}

	// Validate and index configs
	for _, cfg := range configs {
		if cfg.Key == "" {
			return nil, errors.New("filter config missing key")
		}
		if cfg.Capacity <= 0 {
			return nil, fmt.Errorf("invalid capacity for %s: %d", cfg.Key, cfg.Capacity)
		}
		if cfg.FPRate <= 0 || cfg.FPRate >= 1 {
			return nil, fmt.Errorf("invalid fp rate for %s: %f", cfg.Key, cfg.FPRate)
		}
		m.configs[cfg.Key] = cfg
	}

	// Check if bloom filter exist in redis or nota
	logger.WithFields(logrus.Fields{
		"count": len(m.configs),
	}).Info("Checking bloom filters exist in Redis")

	for key := range m.configs {
		exists, err := client.Exists(context.Background(), string(key)).Result()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"filter_key": string(key),
			}).WithError(err).Error("Failed to check bloom filter key existence")
			continue
		} else if exists == 0 {
			logger.WithFields(logrus.Fields{
				"filter_key": string(key),
			}).Debug("Bloom filter key does not exist in Redis")
		}
	}

	return m, nil
}

// getConfig retrieves config for a key - no locks needed since configs are immutable
func (bm *BloomManager) getConfig(key baseConfigs.BloomFilterKey) (baseConfigs.BloomFilterConfig, bool) {
	cfg, exists := bm.configs[key]
	return cfg, exists
}

// Add inserts an item into the specified bloom filter
func (bm *BloomManager) Add(ctx context.Context, bloomFilterKey baseConfigs.BloomFilterKey, item string) error {
	if _, exists := bm.getConfig(bloomFilterKey); !exists {
		return fmt.Errorf("unknown filter key: %s", bloomFilterKey)
	}

	if _, err := bm.client.BFAdd(ctx, string(bloomFilterKey), item).Result(); err != nil {
		return err
	}
	return nil
}

// MightContain checks if item might exist in the filter
func (bm *BloomManager) MightContain(ctx context.Context, bloomFilterKey baseConfigs.BloomFilterKey, item string) (bool, error) {
	if _, exists := bm.getConfig(bloomFilterKey); !exists {
		return false, fmt.Errorf("unknown filter key: %s", bloomFilterKey)
	}

	exists, err := bm.client.BFExists(ctx, string(bloomFilterKey), item).Result()
	bm.recordMetricStandalone(string(bloomFilterKey), exists, err) // record metrics

	if err != nil {
		return false, err
	}

	return exists, nil
}

// AddMulti adds multiple items to a filter efficiently
func (bm *BloomManager) AddMulti(ctx context.Context, bloomFilterKey baseConfigs.BloomFilterKey, items []string) error {
	if len(items) == 0 {
		return nil
	}

	cfg, exists := bm.getConfig(bloomFilterKey)
	if !exists {
		return fmt.Errorf("unknown filter key: %s", bloomFilterKey)
	}

	interfaceItems := make([]interface{}, len(items))
	for i, item := range items {
		interfaceItems[i] = item
	}

	opts := &redis.BFInsertOptions{
		Capacity:   cfg.Capacity,
		Error:      cfg.FPRate,
		NonScaling: false,
	}

	if err := bm.client.BFInsert(ctx, string(bloomFilterKey), opts, interfaceItems...).Err(); err != nil {
		return err
	}
	return nil
}

// CheckMulti checks multiple items at once
func (bm *BloomManager) CheckMulti(ctx context.Context, bloomFilterKey baseConfigs.BloomFilterKey, items []string) ([]bool, error) {
	if len(items) == 0 {
		return []bool{}, nil
	}

	if _, exists := bm.getConfig(bloomFilterKey); !exists {
		return nil, fmt.Errorf("unknown filter key: %s", bloomFilterKey)
	}

	interfaceItems := make([]interface{}, len(items))
	for i, item := range items {
		interfaceItems[i] = item
	}

	results, err := bm.client.BFMExists(ctx, string(bloomFilterKey), interfaceItems...).Result()
	bm.recordMetricMulti(string(bloomFilterKey), results, err) // record metrics

	if err != nil {
		return nil, err
	}
	return results, nil
}

// Stats returns approximate item count in a filter
func (bm *BloomManager) Stats(ctx context.Context, bloomFilterKey baseConfigs.BloomFilterKey) (int64, error) {
	if _, exists := bm.getConfig(bloomFilterKey); !exists {
		return 0, fmt.Errorf("unknown filter key: %s", bloomFilterKey)
	}

	count, err := bm.client.BFCard(ctx, string(bloomFilterKey)).Result()
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Clear removes a bloom filter entirely
func (bm *BloomManager) Clear(ctx context.Context, bloomFilterKey baseConfigs.BloomFilterKey) error {
	if _, exists := bm.getConfig(bloomFilterKey); !exists {
		return fmt.Errorf("unknown filter key: %s", bloomFilterKey)
	}
	return bm.client.Del(ctx, string(bloomFilterKey)).Err()
}

// ListFilters returns all registered filter keys
func (bm *BloomManager) ListFilters() []baseConfigs.BloomFilterKey {
	keys := make([]baseConfigs.BloomFilterKey, 0, len(bm.configs))
	for k := range bm.configs {
		keys = append(keys, k)
	}
	return keys
}

// Builds and executes a pipeline for bulk bloom filter operations
// using a fetcher function to stream items with offset tracking.
// The caller is responsible for managing offset/pagination.
func (bm *BloomManager) BloomPipelineBuilder(
	ctx context.Context,
	bloomFilterKey baseConfigs.BloomFilterKey,
	fetcher func(ctx context.Context, offset, limit int) (*[]string, error),
	batchSize int,
	flushSize int,
) error {
	if batchSize == 0 {
		batchSize = 10000
	}
	if flushSize == 0 {
		flushSize = 10000
	}

	cfg, exists := bm.getConfig(bloomFilterKey)
	if !exists {
		return fmt.Errorf("unknown filter key: %s", bloomFilterKey)
	}

	bfTempKey := string(bloomFilterKey) + ":rebuild"

	// Clean up any stale temp key from previous run
	_ = bm.client.Del(ctx, bfTempKey).Err()

	// Reserve the bloom filter upfront with proper capacity and error rate
	err := bm.client.Do(ctx, "BF.RESERVE", bfTempKey, cfg.FPRate, cfg.Capacity).Err()
	if err != nil {
		return err
	}

	offset := 0
	totalSet := 0

	// helper to flush pipeline
	flushPipeline := func(pipe redis.Pipeliner, queued int) (int, error) {
		if queued == 0 {
			return 0, nil
		}
		_, err := pipe.Exec(ctx)
		if err != nil {
			_ = bm.client.Del(ctx, bfTempKey).Err()
			return 0, fmt.Errorf("pipeline flush failed at offset %d for %s: %w", offset, bloomFilterKey, err)
		}
		totalSet += queued
		return queued, nil
	}

	for {
		// Fetch next batch using the provided fetcher (caller manages offset)
		items, err := fetcher(ctx, offset, flushSize)
		if err != nil {
			_ = bm.client.Del(ctx, bfTempKey).Err()
			return fmt.Errorf("fetch failed at offset %d for %s: %w", offset, bloomFilterKey, err)
		}

		if len(*items) == 0 {
			break // No more items
		}

		// Flush items in chunks using pipeline
		queued := 0
		pipe := bm.client.Pipeline()

		for _, item := range *items {
			pipe.Do(ctx, "BF.ADD", bfTempKey, item)
			queued++

			// Flush pipeline in chunks so we don't queue too many commands in memory
			if queued >= flushSize {
				_, err := flushPipeline(pipe, queued)
				if err != nil {
					return err
				}
				pipe = bm.client.Pipeline()
				queued = 0
			}
		}

		// Flush remaining items in the batch
		if queued > 0 {
			_, err := flushPipeline(pipe, queued)
			if err != nil {
				return err
			}
		}

		offset += batchSize
	}

	// Atomic swap: delete old, rename temp to live
	pipe := bm.client.Pipeline()
	pipe.Del(ctx, string(bloomFilterKey))
	pipe.Rename(ctx, bfTempKey, string(bloomFilterKey))

	_, err = pipe.Exec(ctx)
	if err != nil {
		_ = bm.client.Del(ctx, bfTempKey).Err()
		return fmt.Errorf("atomic swap failed for %s: %w", bloomFilterKey, err)
	}

	return nil
}

// internal helper to update promethues metrics
func (bm *BloomManager) recordMetric(metricKey string, bloomState BloomState, count float64) {
	if metricKey == "" || count == 0 {
		return
	}

	stateStr := bloomState.String()

	// Increment the unified operation counter
	baseMetrics.BloomOperations.WithLabelValues(metricKey, stateStr).Add(count)
}

// internal helper to update promethues metrics for standalone operations
func (bm *BloomManager) recordMetricStandalone(metricKey string, exists bool, err error) {
	if metricKey == "" {
		return
	}

	if err != nil {
		bm.recordMetric(metricKey, bloomError, 1)
	} else if exists {
		bm.recordMetric(metricKey, bloomMightExist, 1)
	} else {
		bm.recordMetric(metricKey, bloomDefinitelyNotExists, 1)
	}
}

// internal helper to update promethues metrics for multi operations
func (bm *BloomManager) recordMetricMulti(metricKey string, results []bool, err error) {
	if metricKey == "" {
		return
	}

	if err != nil {
		bm.recordMetric(metricKey, bloomError, 1)
		return
	}

	var mightExistCount float64
	var definitelyNotExistsCount float64

	for _, exists := range results {
		if exists {
			mightExistCount++
		} else {
			definitelyNotExistsCount++
		}
	}

	// Batch record the states
	bm.recordMetric(metricKey, bloomMightExist, mightExistCount)
	bm.recordMetric(metricKey, bloomDefinitelyNotExists, definitelyNotExistsCount)
}
