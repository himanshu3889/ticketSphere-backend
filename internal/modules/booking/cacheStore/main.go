package bookingCacheStore

import (
	"errors"

	redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	bookingRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/rmq"
	bookingStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/store"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

type CacheStore struct {
	Store        *bookingStore.Store
	CacheManager *redisDatabase.CacheManager
	BloomManager *bloomFilter.BloomManager
	EventBus     *bookingRMQ.EventBus
	logger       *logrus.Logger
	Tracer       trace.Tracer
}

func New(store *bookingStore.Store,
	cacheManager *redisDatabase.CacheManager,
	bloomManager *bloomFilter.BloomManager,
	eventBus *bookingRMQ.EventBus,
	logger *logrus.Logger,
	traceProvider trace.TracerProvider,
) (*CacheStore, error) {
	if store == nil {
		return nil, errors.New("Store is nil")
	}
	if cacheManager == nil {
		return nil, errors.New("Cache manager is nil")
	}
	if bloomManager == nil {
		return nil, errors.New("Bloom manager is nil")
	}
	if logger == nil {
		return nil, errors.New("Logger is nil")
	}
	if eventBus == nil {
		return nil, errors.New("Event bus is nil")
	}

	tracer := traceProvider.Tracer("ticketSphere.booking.cacheStore")

	return &CacheStore{
		Store:        store,
		CacheManager: cacheManager,
		BloomManager: bloomManager,
		EventBus:     eventBus,
		logger:       logger,
		Tracer:       tracer,
	}, nil
}
