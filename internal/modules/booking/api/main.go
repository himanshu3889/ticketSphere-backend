package bookingApi

import (
	"errors"

	bookingCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/cacheStore"
	bookingStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/store"
	"github.com/sirupsen/logrus"
)

type ApiHandler struct {
	cacheStore *bookingCacheStore.CacheStore
	store      *bookingStore.Store // from the cacheStore so cositency remains
	logger     *logrus.Logger
}

func New(cacheStore *bookingCacheStore.CacheStore, logger *logrus.Logger) (*ApiHandler, error) {
	if cacheStore == nil {
		return nil, errors.New("Cache store is nil")
	}
	store := cacheStore.Store
	if store == nil {
		return nil, errors.New("Store is nil from cache store")
	}
	if logger == nil {
		return nil, errors.New("Logger is nil")
	}
	return &ApiHandler{cacheStore: cacheStore, store: store, logger: logger}, nil
}
