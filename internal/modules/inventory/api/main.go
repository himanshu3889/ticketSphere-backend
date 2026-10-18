package inventoryApi

import (
	"errors"

	"github.com/himanshu3889/ticketSphere-backend/configs"
	inventoryCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/inventory/cacheStore"
	inventoryStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/inventory/store"
	"github.com/sirupsen/logrus"
)

type ApiHandler struct {
	authConfigs *configs.AuthConfig
	cacheStore  *inventoryCacheStore.CacheStore
	store       *inventoryStore.Store // borrowed from the cache store so both share one
	logger      *logrus.Logger
}

func New(
	authConfigs *configs.AuthConfig,
	cacheStore *inventoryCacheStore.CacheStore,
	logger *logrus.Logger) (*ApiHandler, error) {
	if authConfigs == nil {
		return nil, errors.New("Auth configs is nil")
	}
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
	return &ApiHandler{authConfigs: authConfigs, cacheStore: cacheStore, store: store, logger: logger}, nil
}
