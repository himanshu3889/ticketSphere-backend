package redisDatabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/base/constants"
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

// CacheItem is one key-value pair to write back to cache.
type CacheItem struct {
	Key   string
	Value interface{}
	TTL   time.Duration
}

// state carried through the singleflight call
type fetchResult struct {
	data         []byte
	isBloomBlock bool
}

// CacheManager wraps Redis reads/writes with singleflight, NULL markers, and optional bloom checks.
type CacheManager struct {
	client       *redis.Client
	bloomMgr     *bloomFilter.BloomManager
	requestGroup singleflight.Group
	logger       *logrus.Logger
	tracer       trace.Tracer
}

// NewCacheManager wires up the cache manager. All deps are required, bloomMgr included.
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

	tracer := traceProvider.Tracer(constants.TracerCacheManager)

	return &CacheManager{
		client:   client,
		bloomMgr: bloomMgr,
		logger:   logger,
		tracer:   tracer,
	}, nil
}

// MarshalForCache JSON-encodes data for Redis; nil and typed-nil become NULL_Value.
// isNotNil=false tells callers to skip the bloom add.
func (cm *CacheManager) MarshalForCache(data interface{}) (redisValue interface{}, isNotNil bool, err error) {
	if data == nil {
		return NULL_Value, false, nil
	}

	// typed nil: nil ptr inside an interface
	v := reflect.ValueOf(data)
	// Kind check first, IsNil panics on non-pointers
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return NULL_Value, false, nil
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// Set writes one key (singleflighted). Nil data stores NULL_Value and skips bloom.
func (cm *CacheManager) Set(ctx context.Context, cacheKey string, bloomKey *configs.BloomFilterKey, data interface{}, bloomItem *string, ttl time.Duration) error {
	_, err, _ := cm.requestGroup.Do("set:"+cacheKey, func() (interface{}, error) {
		redisValue, isNotNil, err := cm.MarshalForCache(data)
		if err != nil {
			return nil, fmt.Errorf("cache marshal failed: %w", err)
		}

		if err := cm.client.Set(ctx, cacheKey, redisValue, ttl).Err(); err != nil {
			return nil, fmt.Errorf("cache set failed: %w", err)
		}

		// don't add null value to bloom filter
		if !isNotNil {
			return nil, nil
		}

		// async bloom add; never blocks the set
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

// SetMulti pipelines many writes. No bloom filter.
// One marshal failure aborts the whole batch before Redis sees anything.
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
			// drop queued commands before bailing
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

// Get reads one key and returns raw bytes; caller unmarshals.
func (cm *CacheManager) Get(ctx context.Context, boundedKey string, cacheKey string, bloomKey *configs.BloomFilterKey, bloomItem *string) ([]byte, CacheState, error) {
	start := time.Now()

	rawResult, err, _ := cm.requestGroup.Do("get:"+cacheKey, func() (interface{}, error) {
		// bloom first: definite no means skip the redis read
		if cm.bloomMgr != nil && bloomKey != nil && *bloomKey != "" && bloomItem != nil && *bloomItem != "" {
			mightExist, err := cm.bloomMgr.MightContain(ctx, *bloomKey, *bloomItem)
			if err == nil && !mightExist {
				return fetchResult{data: nil, isBloomBlock: true}, nil
			}

		}
		val, err := cm.client.Get(ctx, cacheKey).Bytes()
		return fetchResult{data: val, isBloomBlock: false}, err
	})

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

	// negative-cache marker: null block, not a miss
	if resultBytes == nil || bytes.Equal(resultBytes, []byte(NULL_Value)) {
		cm.recordMetric(boundedKey, CacheNullBlock, time.Since(start))
		return nil, CacheNullBlock, nil
	}

	return resultBytes, CacheHit, nil
}

// GetInto unmarshals a cache hit into dest. dest must be a pointer.
func (cm *CacheManager) GetInto(ctx context.Context, boundedKey string, cacheKey string, bloomKey *configs.BloomFilterKey, bloomItem *string, dest any) (CacheState, error) {
	data, state, err := cm.Get(ctx, boundedKey, cacheKey, bloomKey, bloomItem)
	if err != nil {
		return 0, err // System error
	}

	// Miss or a Penetration Block, DO NOT try to unmarshal.
	if state != CacheHit {
		return state, nil
	}

	// guaranteed hit here, safe to unmarshal
	if err := json.Unmarshal(data, dest); err != nil {
		return CacheHit, fmt.Errorf("failed to unmarshal cache data for key %s: %w", cacheKey, err)
	}

	return CacheHit, nil
}

// Delete drops the key.
func (cm *CacheManager) Delete(ctx context.Context, cacheKey string) error {
	return cm.client.Del(ctx, cacheKey).Err()
}

// parseMGetVals normalizes a raw MGET reply: hit bytes per key with NULL_Value
// markers as nil entries; true misses left absent from the map entirely.
func parseMGetVals(cacheKeys []string, vals []interface{}) (result map[string][]byte, hits int, misses int) {
	hits, misses = 0, 0
	result = make(map[string][]byte, len(cacheKeys))
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
	return result, hits, misses
}

// MGet fetches many keys in one round trip. NULL_Value markers come back as nil
// entries; true misses are absent from the map entirely.
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

	result, hits, misses := parseMGetVals(cacheKeys, vals)

	cm.recordMultiGetMetric(boundedKey, hits, misses, duration)

	return result, nil
}

// unmarshalMGetInto decodes each hit into its aligned dest and reports the
// per-key CacheState (hit / miss / null block / unmarshal error). The first
// unmarshal error is remembered; all keys are still processed.
func unmarshalMGetInto(cacheKeys []string, dests []any, rawMap map[string][]byte) (map[string]CacheState, error) {
	states := make(map[string]CacheState, len(cacheKeys))
	var unmarshalErr error

	for i, key := range cacheKeys {
		raw, ok := rawMap[key]
		if !ok {
			// Key not in result = true cache miss (redis returned nil)
			states[key] = CacheMiss
			continue
		}

		// negative-cache marker: MGet turned NULL_Value into nil, but catch the raw marker too
		if raw == nil || bytes.Equal(raw, []byte(NULL_Value)) {
			states[key] = CacheNullBlock
			continue
		}

		// guaranteed hit here, safe to unmarshal
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

// MGetInto unmarshals hits into dests
// The per-key CacheState says hit, miss, null block, or unmarshal error.
func (cm *CacheManager) MGetInto(ctx context.Context, boundedKey string, cacheKeys []string, dests []any) (map[string]CacheState, error) {
	if len(cacheKeys) != len(dests) {
		return nil, fmt.Errorf("cacheKeys and dests length mismatch: %d vs %d", len(cacheKeys), len(dests))
	}
	if len(cacheKeys) == 0 {
		return map[string]CacheState{}, nil
	}

	rawMap, err := cm.MGet(ctx, boundedKey, cacheKeys)
	if err != nil {
		return nil, err
	}

	return unmarshalMGetInto(cacheKeys, dests, rawMap)
}

// MGetIntoWithBloom folds the bloom existence sweep (BF.MEXISTS) and the value
// fetch into a single pipelined round trip on the same connection.
func (cm *CacheManager) MGetIntoWithBloom(ctx context.Context, bloomFilterKey configs.BloomFilterKey, bloomItems []string, boundedKey string, cacheKeys []string, dests []any) ([]bool, map[string]CacheState, error) {
	if len(cacheKeys) != len(dests) {
		return nil, nil, fmt.Errorf("cacheKeys and dests length mismatch: %d vs %d", len(cacheKeys), len(dests))
	}
	if len(bloomItems) != len(cacheKeys) {
		return nil, nil, fmt.Errorf("bloomItems and cacheKeys length mismatch: %d vs %d", len(bloomItems), len(cacheKeys))
	}
	if len(cacheKeys) == 0 {
		return []bool{}, map[string]CacheState{}, nil
	}

	start := time.Now()

	// queue the bloom sweep and the value fetch, flush once = one round trip
	pipe := cm.client.Pipeline()

	bloomArgs := make([]interface{}, len(bloomItems))
	for i, item := range bloomItems {
		bloomArgs[i] = item
	}
	bloomCmd := pipe.BFMExists(ctx, string(bloomFilterKey), bloomArgs...)
	mgetCmd := pipe.MGet(ctx, cacheKeys...)

	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		cm.recordMetric(boundedKey, CacheError, time.Since(start))
		baseMetrics.BloomOperations.WithLabelValues(string(bloomFilterKey), "error").Inc()
		return nil, nil, fmt.Errorf("cache bloom+mget pipeline failed: %w", err)
	}

	duration := time.Since(start)

	matches, err := bloomCmd.Result()
	if err != nil {
		cm.recordMetric(boundedKey, CacheError, duration)
		baseMetrics.BloomOperations.WithLabelValues(string(bloomFilterKey), "error").Inc()
		return nil, nil, fmt.Errorf("cache bloom check failed: %w", err)
	}

	// same bloom metric series as bloomFilter.CheckMulti
	var mightExist, definitelyNotExists float64
	for _, m := range matches {
		if m {
			mightExist++
		} else {
			definitelyNotExists++
		}
	}
	if mightExist > 0 {
		baseMetrics.BloomOperations.WithLabelValues(string(bloomFilterKey), "might_exist").Add(mightExist)
	}
	if definitelyNotExists > 0 {
		baseMetrics.BloomOperations.WithLabelValues(string(bloomFilterKey), "definitely_not_exists").Add(definitelyNotExists)
	}

	vals, err := mgetCmd.Result()
	if err != nil {
		cm.recordMetric(boundedKey, CacheError, duration)
		return nil, nil, fmt.Errorf("cache MGet failed: %w", err)
	}

	result, hits, misses := parseMGetVals(cacheKeys, vals)

	cm.recordMultiGetMetric(boundedKey, hits, misses, duration)

	states, unmarshalErr := unmarshalMGetInto(cacheKeys, dests, result)

	return matches, states, unmarshalErr
}

// HGet retrieves a specific field from a Redis hash with optional bloom filter check
func (cm *CacheManager) HGet(ctx context.Context, boundedKey string, cacheKey string, hashField string, bloomKey *configs.BloomFilterKey, bloomItem *string) ([]byte, CacheState, error) {
	start := time.Now()

	// Include hashField in the singleflight key to prevent cross-field collisions
	singleflightKey := "hget:" + cacheKey + ":" + hashField

	rawResult, err, _ := cm.requestGroup.Do(singleflightKey, func() (interface{}, error) {
		// Check bloom filter first if configured
		if cm.bloomMgr != nil && bloomKey != nil && *bloomKey != "" && bloomItem != nil && *bloomItem != "" {
			mightExist, err := cm.bloomMgr.MightContain(ctx, *bloomKey, *bloomItem)
			if err == nil && !mightExist {
				// Bloom filter says it definitely doesn't exist
				return fetchResult{data: nil, isBloomBlock: true}, nil
			}
		}

		// Get from cache using HGet instead of Get
		val, err := cm.client.HGet(ctx, cacheKey, hashField).Bytes()
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

	// Bloom block or cache negative cache
	if resultBytes == nil || bytes.Equal(resultBytes, []byte(NULL_Value)) {
		cm.recordMetric(boundedKey, CacheNullBlock, time.Since(start))
		return nil, CacheNullBlock, nil
	}

	// Success
	return resultBytes, CacheHit, nil
}

// HGetInto retrieves a specific hash field and unmarshals it into the provided dest pointer.
func (cm *CacheManager) HGetInto(ctx context.Context, boundedKey string, cacheKey string, hashField string, bloomKey *configs.BloomFilterKey, bloomItem *string, dest any) (CacheState, error) {
	data, state, err := cm.HGet(ctx, boundedKey, cacheKey, hashField, bloomKey, bloomItem)
	if err != nil {
		return 0, err
	}

	// Miss or a Penetration Block, DO NOT try to unmarshal.
	if state != CacheHit {
		return state, nil
	}

	// guaranteed CacheHit, Unmarshal the data.
	if err := json.Unmarshal(data, dest); err != nil {
		return CacheHit, fmt.Errorf("failed to unmarshal hash data for key %s, field %s: %w", cacheKey, hashField, err)
	}

	// Success
	return CacheHit, nil
}

// scripts must run independently for every caller.
func (cm *CacheManager) RunScript(ctx context.Context, boundedKey string, script *redis.Script, keys []string, args ...interface{}) (result interface{}, err error) {
	ctx, span := cm.tracer.Start(ctx, constants.SpanRedisRunScript)
	defer utils.EndSpan(span, err)

	span.SetAttributes(attribute.String("redis.script.name", boundedKey))

	start := time.Now()

	// script.Run automatically handles EVAL vs EVALSHA caching in go-redis
	result, err = script.Run(ctx, cm.client, keys, args...).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		cm.recordMetric(boundedKey, CacheError, time.Since(start))
		return nil, err
	}

	cm.recordMetric(boundedKey, CacheHit, time.Since(start))
	return result, nil
}

// KeyValPipelineBuilder rebuilds cache entries from offset-paginated fetcher batches.
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

// hit/miss counters plus mget latency
func (cm *CacheManager) recordMultiGetMetric(metricKey string, hits int, misses int, duration time.Duration) {
	if metricKey == "" {
		return
	}
	baseMetrics.CacheHits.WithLabelValues(metricKey).Add(float64(hits))
	baseMetrics.CacheMisses.WithLabelValues(metricKey).Add(float64(misses))
	baseMetrics.CacheLatency.WithLabelValues(metricKey, "mget").Observe(duration.Seconds())
}

// per-op state counter and latency
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

	baseMetrics.CacheOperations.WithLabelValues(metricKey, stateStr).Inc()

	baseMetrics.CacheLatency.WithLabelValues(metricKey, stateStr).Observe(duration.Seconds())
}
