package bookingApi

import (
	"errors"

	"github.com/himanshu3889/ticketSphere-backend/configs"
	bookingCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/cacheStore"
	bookingStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/store"
	"github.com/sirupsen/logrus"
)

type ApiHandler struct {
	authConfigs *configs.AuthConfigurations
	cacheStore  *bookingCacheStore.CacheStore
	store       *bookingStore.Store // from the cacheStore so consistency remains
	logger      *logrus.Logger
}

func New(
	authConfigs *configs.AuthConfigurations,
	cacheStore *bookingCacheStore.CacheStore,
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
