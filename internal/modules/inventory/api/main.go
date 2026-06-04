package inventoryApi

import (
	"errors"

	inventoryCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/inventory/cacheStore"
	inventoryStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/inventory/store"
	"github.com/sirupsen/logrus"
)

type ApiHandler struct {
	cacheStore *inventoryCacheStore.CacheStore
	store      *inventoryStore.Store // from the cacheStore so cositency remains
	logger     *logrus.Logger
}

func New(cacheStore *inventoryCacheStore.CacheStore, logger *logrus.Logger) (*ApiHandler, error) {
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
