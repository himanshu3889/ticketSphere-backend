package bloomFilter

import (
	"context"
	"errors"
	"fmt"

	baseConfigs "github.com/himanshu3889/ticketSphere-backend/configs"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

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
			}).Warn("Bloom filter key does not exist in Redis")
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
func (bm *BloomManager) Add(ctx context.Context, filterKey baseConfigs.BloomFilterKey, item string) error {
	if _, exists := bm.getConfig(filterKey); !exists {
		return fmt.Errorf("unknown filter key: %s", filterKey)
	}

	if _, err := bm.client.BFAdd(ctx, string(filterKey), item).Result(); err != nil {
		return fmt.Errorf("bfadd failed for %s: %w", filterKey, err)
	}
	return nil
}

// MightContain checks if item might exist in the filter
func (bm *BloomManager) MightContain(ctx context.Context, filterKey baseConfigs.BloomFilterKey, item string) (bool, error) {
	if _, exists := bm.getConfig(filterKey); !exists {
		return false, fmt.Errorf("unknown filter key: %s", filterKey)
	}

	exists, err := bm.client.BFExists(ctx, string(filterKey), item).Result()
	if err != nil {
		return false, fmt.Errorf("bfexists failed for %s: %w", filterKey, err)
	}
	return exists, nil
}

// AddMulti adds multiple items to a filter efficiently
func (bm *BloomManager) AddMulti(ctx context.Context, filterKey baseConfigs.BloomFilterKey, items []string) error {
	if len(items) == 0 {
		return nil
	}

	cfg, exists := bm.getConfig(filterKey)
	if !exists {
		return fmt.Errorf("unknown filter key: %s", filterKey)
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

	if err := bm.client.BFInsert(ctx, string(filterKey), opts, interfaceItems...).Err(); err != nil {
		return fmt.Errorf("bfinsert failed for %s: %w", filterKey, err)
	}
	return nil
}

// CheckMulti checks multiple items at once
func (bm *BloomManager) CheckMulti(ctx context.Context, filterKey baseConfigs.BloomFilterKey, items []string) ([]bool, error) {
	if len(items) == 0 {
		return []bool{}, nil
	}

	if _, exists := bm.getConfig(filterKey); !exists {
		return nil, fmt.Errorf("unknown filter key: %s", filterKey)
	}

	interfaceItems := make([]interface{}, len(items))
	for i, item := range items {
		interfaceItems[i] = item
	}

	results, err := bm.client.BFMExists(ctx, string(filterKey), interfaceItems...).Result()
	if err != nil {
		return nil, fmt.Errorf("bfmexists failed for %s: %w", filterKey, err)
	}
	return results, nil
}

// Stats returns approximate item count in a filter
func (bm *BloomManager) Stats(ctx context.Context, filterKey baseConfigs.BloomFilterKey) (int64, error) {
	if _, exists := bm.getConfig(filterKey); !exists {
		return 0, fmt.Errorf("unknown filter key: %s", filterKey)
	}

	count, err := bm.client.BFCard(ctx, string(filterKey)).Result()
	if err != nil {
		return 0, fmt.Errorf("bfcard failed for %s: %w", filterKey, err)
	}
	return count, nil
}

// Clear removes a bloom filter entirely
func (bm *BloomManager) Clear(ctx context.Context, filterKey baseConfigs.BloomFilterKey) error {
	if _, exists := bm.getConfig(filterKey); !exists {
		return fmt.Errorf("unknown filter key: %s", filterKey)
	}
	return bm.client.Del(ctx, string(filterKey)).Err()
}

// ListFilters returns all registered filter keys
func (bm *BloomManager) ListFilters() []baseConfigs.BloomFilterKey {
	keys := make([]baseConfigs.BloomFilterKey, 0, len(bm.configs))
	for k := range bm.configs {
		keys = append(keys, k)
	}
	return keys
}
