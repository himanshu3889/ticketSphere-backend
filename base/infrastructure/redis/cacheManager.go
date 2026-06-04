package redisDatabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	baseMetrics "github.com/himanshu3889/ticketSphere-backend/base/metric"
	"github.com/himanshu3889/ticketSphere-backend/configs"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const NULL_Value = "__NULL__"

// Create a small internal struct to carry the state through singleflight
type fetchResult struct {
	data         []byte
	isBloomBlock bool
}

// CacheManager handles all cache operations with optional Bloom filter integration
type CacheManager struct {
	client   *redis.Client
	bloomMgr *bloomFilter.BloomManager
	group    singleflight.Group
}

// NewCacheManager creates a new cache manager
// bloomMgr is optional - pass nil to disable bloom filter
func NewCacheManager(client *redis.Client, bloomMgr *bloomFilter.BloomManager) *CacheManager {
	return &CacheManager{
		client:   client,
		bloomMgr: bloomMgr,
	}
}

// Set stores data in cache and optionally adds to bloom filter
// data can be any type - will be JSON marshaled
func (cm *CacheManager) Set(ctx context.Context, cacheKey string, bloomKey *configs.BloomFilterKey, data interface{}, bloomItem *string, ttl time.Duration) error {
	// Marshal data

	isNilData := data == nil
	// Check for "Typed" Nil (nil pointer passed inside interface)
	if !isNilData {
		val := reflect.ValueOf(data)
		// We must check the Kind first to avoid panics on non-pointer types
		if val.Kind() == reflect.Ptr && val.IsNil() {
			isNilData = true
		}
	}
	if isNilData {
		// Write NULL to cache only
		if err := cm.client.Set(ctx, cacheKey, NULL_Value, ttl).Err(); err != nil {
			return fmt.Errorf("cache set failed: %w", err)
		}
		return nil
	}

	byteData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("cache marshal failed: %w", err)
	}

	// Write to cache
	if err := cm.client.Set(ctx, cacheKey, byteData, ttl).Err(); err != nil {
		return fmt.Errorf("cache set failed: %w", err)
	}

	// Add to bloom filter if configured
	if cm.bloomMgr != nil && bloomKey != nil && *bloomKey != "" && bloomItem != nil && *bloomItem != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = cm.bloomMgr.Add(ctx, *bloomKey, *bloomItem)
		}()
	}

	return nil
}

// Get retrieves data from cache with optional bloom filter check
// Returns raw bytes or nil (if not exists) - caller handles unmarshaling
func (cm *CacheManager) Get(ctx context.Context, boundedKey string, cacheKey string, bloomKey *configs.BloomFilterKey, bloomItem *string) ([]byte, error) {
	start := time.Now()

	rawResult, err, _ := cm.group.Do(cacheKey, func() (interface{}, error) {
		// Check bloom filter first if configured
		if cm.bloomMgr != nil && bloomKey != nil && *bloomKey != "" && bloomItem != nil && *bloomItem != "" {
			mightExist, err := cm.bloomMgr.MightContain(ctx, *bloomKey, *bloomItem)
			if err == nil && !mightExist {
				// Bloom filter says it definitely doesn't exist
				return fetchResult{data: nil, isBloomBlock: true}, nil
			}

		}
		// Get from cache
		val, err := cm.client.Get(ctx, cacheKey).Bytes()
		return fetchResult{data: val, isBloomBlock: false}, err
	})

	// Extract values safely
	var res fetchResult
	if rawResult != nil {
		res = rawResult.(fetchResult)
	}

	// for EVERY caller in the singleflight group
	cm.recordMetric(boundedKey, err, res.isBloomBlock, time.Since(start))

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	resultBytes := res.data

	if resultBytes == nil || bytes.Equal(resultBytes, []byte(NULL_Value)) {
		return nil, nil
	}

	return resultBytes, nil
}

// Delete removes from cache
func (cm *CacheManager) Delete(ctx context.Context, cacheKey string) error {
	return cm.client.Del(ctx, cacheKey).Err()
}

// MGet retrieves multiple keys from Redis cache
// Returns a map of cacheKey -> raw value (string or []byte)
func (cm *CacheManager) MGet(ctx context.Context, boundedKey string, cacheKeys []string) (map[string][]byte, error) {
	if len(cacheKeys) == 0 {
		return map[string][]byte{}, nil
	}

	start := time.Now()

	vals, err := cm.client.MGet(ctx, cacheKeys...).Result()

	duration := time.Since(start)

	if err != nil {
		cm.recordMetric(boundedKey, err, false, duration)
		return nil, fmt.Errorf("cache MGet failed: %w", err)
	}

	hits, misses := 0, 0
	result := make(map[string][]byte, len(cacheKeys))
	for i, v := range vals {
		if v == nil {
			misses++
			continue
		}

		var resultBytes []byte
		switch t := v.(type) {
		case string:
			resultBytes = []byte(t)
		case []byte:
			resultBytes = t
		default:
			continue
		}

		if bytes.Equal(resultBytes, []byte(NULL_Value)) {
			result[cacheKeys[i]] = nil
		} else {
			result[cacheKeys[i]] = resultBytes
		}

		hits++

	}

	cm.recordMultiGetMetric(boundedKey, hits, misses, duration)

	return result, nil
}

// scripts must run independently for every caller.  // TODO: log errors for redis scripts
func (cm *CacheManager) RunScript(ctx context.Context, boundedKey string, script *redis.Script, keys []string, args ...interface{}) (interface{}, error) {
	start := time.Now()

	// script.Run automatically handles EVAL vs EVALSHA caching in go-redis
	result, err := script.Run(ctx, cm.client, keys, args...).Result()

	// Record the metric using the same pattern as Get
	cm.recordMetric(boundedKey, err, false, time.Since(start))

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	return result, nil
}

// recordMetric internal helper to update Prometheus
func (cm *CacheManager) recordMultiGetMetric(metricKey string, hits int, misses int, duration time.Duration) {
	if metricKey == "" {
		return
	}
	baseMetrics.CacheHits.WithLabelValues(metricKey).Add(float64(hits))
	baseMetrics.CacheMisses.WithLabelValues(metricKey).Add(float64(misses))
	baseMetrics.CacheLatency.WithLabelValues(metricKey, "mget").Observe(duration.Seconds())
}

// recordMetric internal helper to update Prometheus
func (cm *CacheManager) recordMetric(metricKey string, err error, isBloomBlock bool, duration time.Duration) {
	if metricKey == "" {
		return
	}

	if isBloomBlock {
		return
	}

	status := "hit"
	// It's a miss if redis returned Nil or if bloom filter returned nil result without error
	if err == redis.Nil {
		status = "miss"
		baseMetrics.CacheMisses.WithLabelValues(metricKey).Inc()
	} else if err == nil {
		baseMetrics.CacheHits.WithLabelValues(metricKey).Inc()
	} else {
		status = "error"
	}

	baseMetrics.CacheLatency.WithLabelValues(metricKey, status).Observe(duration.Seconds())
}
