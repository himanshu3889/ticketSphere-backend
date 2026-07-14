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
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/configs"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const NULL_Value = "__NULL__"

type CacheState int

const (
	// CacheHit means real data was found.
	CacheHit CacheState = iota

	// CacheMiss means the key wasn't there; caller should query the DB.
	CacheMiss

	// CacheNullBlock means we queried Redis and found the NULL_Value marker.
	CacheNullBlock

	// CacheBloomBlock means the local Bloom filter rejected the key (Zero Redis I/O).
	CacheBloomBlock

	// CacheError means error checking cache
	CacheError
)

func (c CacheState) String() string {
	switch c {
	case CacheHit:
		return "cache_hit"
	case CacheMiss:
		return "cache_miss"
	case CacheBloomBlock:
		return "cache_bloom_block"
	case CacheNullBlock:
		return "cache_null_block"
	case CacheError:
		return "cache_error"
	default:
		return ""
	}
}

// CacheItem represents a single key-value pair to rebuild in cache
type CacheItem struct {
	Key   string
	Value interface{}
	TTL   time.Duration
}

// Create a small internal struct to carry the state through singleflight
type fetchResult struct {
	data         []byte
	isBloomBlock bool
}

// CacheManager handles all cache operations with optional Bloom filter integration
type CacheManager struct {
	client       *redis.Client
	bloomMgr     *bloomFilter.BloomManager
	requestGroup singleflight.Group
	logger       *logrus.Logger
	tracer       trace.Tracer
}

// NewCacheManager creates a new cache manager
// bloomMgr is optional - pass nil to disable bloom filter
func NewCacheManager(client *redis.Client, bloomMgr *bloomFilter.BloomManager, logger *logrus.Logger, traceProvider trace.TracerProvider) (*CacheManager, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	if bloomMgr == nil {
		return nil, fmt.Errorf("bloom manager is nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is nil")
	}
	if traceProvider == nil {
		return nil, fmt.Errorf("trace provider is nil")
	}

	tracer := traceProvider.Tracer("cacheManager")

	return &CacheManager{
		client:   client,
		bloomMgr: bloomMgr,
		logger:   logger,
		tracer:   tracer,
	}, nil
}

// converts interface{} to Redis value
// Redis value: NULL_Value if nil or typed nil, else JSON marshaled bytes
func (cm *CacheManager) MarshalForCache(data interface{}) (redisValue interface{}, isNotNil bool, err error) {
	if data == nil {
		return NULL_Value, false, nil
	}

	// Check for "Typed" Nil (nil pointer passed inside interface)
	v := reflect.ValueOf(data)
	// We must check the Kind first to avoid panics on non-pointer types
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return NULL_Value, false, nil
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// Set stores data in cache and optionally adds to bloom filter
// data can be any type - will be JSON marshaled
func (cm *CacheManager) Set(ctx context.Context, cacheKey string, bloomKey *configs.BloomFilterKey, data interface{}, bloomItem *string, ttl time.Duration) error {
	_, err, _ := cm.requestGroup.Do("set:"+cacheKey, func() (interface{}, error) {
		// Marshal data
		redisValue, isNotNil, err := cm.MarshalForCache(data)
		if err != nil {
			return nil, fmt.Errorf("cache marshal failed: %w", err)
		}

		// Write to cache
		if err := cm.client.Set(ctx, cacheKey, redisValue, ttl).Err(); err != nil {
			return nil, fmt.Errorf("cache set failed: %w", err)
		}

		// don't add null value to bloom filter
		if !isNotNil {
			return nil, nil
		}

		// Add to bloom filter if configured
		if cm.bloomMgr != nil && bloomKey != nil && *bloomKey != "" && bloomItem != nil && *bloomItem != "" {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = cm.bloomMgr.Add(ctx, *bloomKey, *bloomItem)
			}()
		}

		return nil, nil
	})
	return err
}

// SetMulti stores multiple items in cache using a pipeline.
// Each item is JSON marshaled. No bloom filter is used.
// If any marshal fails, the entire operation fails before anything hits Redis.
func (cm *CacheManager) SetMulti(ctx context.Context, cacheKeys []string, data []interface{}, ttl time.Duration) error {
	if len(cacheKeys) != len(data) {
		return fmt.Errorf("cacheKeys and data length mismatch: %d vs %d", len(cacheKeys), len(data))
	}
	if len(cacheKeys) == 0 {
		return nil
	}

	pipe := cm.client.Pipeline()

	for i, key := range cacheKeys {
		redisValue, _, err := cm.MarshalForCache(data[i])
		if err != nil {
			// Clean up the pipeline so we don't leak queued commands
			pipe.Discard()
			return fmt.Errorf("cache marshal failed for key %s: %w", key, err)
		}

		pipe.Set(ctx, key, redisValue, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("cache setmulti failed: %w", err)
	}

	return nil
}

// Get retrieves data from cache with optional bloom filter check
// Returns raw bytes or nil (if not exists) - caller handles unmarshaling
func (cm *CacheManager) Get(ctx context.Context, boundedKey string, cacheKey string, bloomKey *configs.BloomFilterKey, bloomItem *string) ([]byte, CacheState, error) {
	start := time.Now()

	rawResult, err, _ := cm.requestGroup.Do("get:"+cacheKey, func() (interface{}, error) {
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

	if res.isBloomBlock {
		cm.recordMetric(boundedKey, CacheBloomBlock, time.Since(start))
		return nil, CacheBloomBlock, nil
	}

	if err != nil {
		if err == redis.Nil {
			// True Miss - Go check DB
			return nil, CacheMiss, nil
		}
		// Actual system failure
		cm.recordMetric(boundedKey, CacheError, time.Since(start))
		return nil, CacheError, err
	}

	resultBytes := res.data

	// Penetration Marker: Bloom block or cache negative cache
	if resultBytes == nil || bytes.Equal(resultBytes, []byte(NULL_Value)) {
		cm.recordMetric(boundedKey, CacheNullBlock, time.Since(start))
		return nil, CacheNullBlock, nil
	}

	// Success
	return resultBytes, CacheHit, nil
}

// Retrieves data from cache and unmarshals it into the provided dest pointer.
// dest MUST be a pointer to a struct, map, or slice.
func (cm *CacheManager) GetInto(ctx context.Context, boundedKey string, cacheKey string, bloomKey *configs.BloomFilterKey, bloomItem *string, dest any) (CacheState, error) {
	data, state, err := cm.Get(ctx, boundedKey, cacheKey, bloomKey, bloomItem)
	if err != nil {
		return 0, err // System error
	}

	// Miss or a Penetration Block, DO NOT try to unmarshal.
	if state != CacheHit {
		return state, nil
	}

	// guaranteed CacheHit, Unmarshal the data.
	if err := json.Unmarshal(data, dest); err != nil {
		return CacheHit, fmt.Errorf("failed to unmarshal cache data for key %s: %w", cacheKey, err)
	}

	// Success
	return CacheHit, nil
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
		cm.recordMetric(boundedKey, CacheError, duration)
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

// MGetInto retrieves multiple keys from cache and unmarshals hits into the provided dest pointers.
// cacheKeys and dests must be the same length, and each dest[i] MUST be a pointer to a struct, map, or slice.
// Returns a map of cacheKey -> CacheState so you know which ones were hits, misses, null blocks, or unmarshal errors.
func (cm *CacheManager) MGetInto(ctx context.Context, boundedKey string, cacheKeys []string, dests []any) (map[string]CacheState, error) {
	if len(cacheKeys) != len(dests) {
		return nil, fmt.Errorf("cacheKeys and dests length mismatch: %d vs %d", len(cacheKeys), len(dests))
	}
	if len(cacheKeys) == 0 {
		return map[string]CacheState{}, nil
	}

	// Fetch raw bytes for all keys in one round trip
	rawMap, err := cm.MGet(ctx, boundedKey, cacheKeys)
	if err != nil {
		return nil, err
	}

	states := make(map[string]CacheState, len(cacheKeys))
	var unmarshalErr error

	for i, key := range cacheKeys {
		raw, ok := rawMap[key]
		if !ok {
			// Key not in result = true cache miss (redis returned nil)
			states[key] = CacheMiss
			continue
		}

		// Penetration marker: either explicit nil from NULL_Value or the raw marker itself
		if raw == nil || bytes.Equal(raw, []byte(NULL_Value)) {
			states[key] = CacheNullBlock
			continue
		}

		// Guaranteed hit - unmarshal into the provided pointer
		if err := json.Unmarshal(raw, dests[i]); err != nil {
			if unmarshalErr == nil {
				unmarshalErr = fmt.Errorf("failed to unmarshal cache data for key %s: %w", key, err)
			}
			states[key] = CacheError
			continue
		}

		states[key] = CacheHit
	}

	return states, unmarshalErr
}

// scripts must run independently for every caller.
func (cm *CacheManager) RunScript(ctx context.Context, boundedKey string, script *redis.Script, keys []string, args ...interface{}) (result interface{}, err error) {
	ctx, span := cm.tracer.Start(ctx, "Redis.RunScript")
	defer utils.EndSpan(span, err)

	// Attach the script name
	span.SetAttributes(attribute.String("redis.script.name", boundedKey))

	start := time.Now()

	// script.Run automatically handles EVAL vs EVALSHA caching in go-redis
	result, err = script.Run(ctx, cm.client, keys, args...).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		// Record the metrics too
		cm.recordMetric(boundedKey, CacheError, time.Since(start))
		return nil, err
	}

	cm.recordMetric(boundedKey, CacheHit, time.Since(start))
	return result, nil
}

// Rebuilds cache entries from any data source using a fetcher function with offset-based pagination.
func (cm *CacheManager) KeyValPipelineBuilder(
	ctx context.Context,
	fetcher func(ctx context.Context, offset, limit int) (*[]CacheItem, error),
	batchSize int,
	flushSize int,
	defaultTTL time.Duration,
) error {
	if batchSize == 0 {
		batchSize = 10000
	}
	if flushSize == 0 {
		flushSize = 10000
	}

	pipe := cm.client.Pipeline()
	offset := 0
	totalSet := 0
	queued := 0

	// helper to flush pipeline
	flushPipeline := func(pipe redis.Pipeliner, queued int) (int, error) {
		if queued == 0 {
			return 0, nil
		}
		_, err := pipe.Exec(ctx)
		if err != nil {
			pipe.Discard()
			return 0, err
		}
		totalSet += queued
		return queued, nil
	}

	for {
		// Fetch next batch
		items, err := fetcher(ctx, offset, batchSize)
		if err != nil {
			return fmt.Errorf("fetch failed at offset %d: %w", offset, err)
		}

		if len(*items) == 0 {
			break // Done
		}

		for _, item := range *items {
			key := item.Key

			ttl := item.TTL
			if ttl == 0 {
				ttl = defaultTTL
			}

			val, _, err := cm.MarshalForCache(item.Value)
			if err != nil {
				// One bad row should not kill a 2-hour rebuild. Log and skip.
				cm.logger.WithFields(logrus.Fields{
					"key": key,
					"err": err,
				}).Debug("Skipping key")
				continue
			}

			pipe.Set(ctx, key, val, ttl)
			queued++

			// Flush pipeline in chunks so we don't queue 10,000 commands in memory
			if queued >= flushSize {
				_, err := flushPipeline(pipe, queued)
				if err != nil {
					return fmt.Errorf("pipeline flush failed at offset %d: %w", offset, err)
				}
				pipe = cm.client.Pipeline()
				queued = 0
			}
		}

		// Flush whatever is left in the pipe
		if queued > 0 {
			_, err := flushPipeline(pipe, queued)
			if err != nil {
				return fmt.Errorf("pipeline flush failed at offset %d: %w", offset, err)
			}
		}

		offset += batchSize
	}

	return nil
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

// internal helper to update promethues metrics
func (cm *CacheManager) recordMetric(metricKey string, cacheState CacheState, duration time.Duration) {
	if metricKey == "" {
		return
	}

	switch cacheState {
	case CacheHit:
		baseMetrics.CacheMisses.WithLabelValues(metricKey).Inc()
	case CacheMiss:
		baseMetrics.CacheMisses.WithLabelValues(metricKey).Inc()
	}

	stateStr := cacheState.String()

	// Increment the unified operation counter
	baseMetrics.CacheOperations.WithLabelValues(metricKey, stateStr).Inc()

	// Observe the latency
	baseMetrics.CacheLatency.WithLabelValues(metricKey, stateStr).Observe(duration.Seconds())
}
