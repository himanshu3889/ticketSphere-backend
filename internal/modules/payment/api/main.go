package paymentApi

import (
	"errors"

	"github.com/himanshu3889/ticketSphere-backend/configs"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	paymentRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/rmq"
	paymentStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/store"
	"github.com/sirupsen/logrus"
)

// ApiHandler holds the payment API dependencies.
type ApiHandler struct {
	authConfigs           *configs.AuthConfig
	paymentGatewayConfigs *configs.PaymentGatewayConfig
	cacheStore            *paymentCacheStore.CacheStore
	store                 *paymentStore.Store // same store the cache layer uses
	eventBus              *paymentRMQ.EventBus
	logger                *logrus.Logger
}

// New wires the handler, rejecting nil deps.
func New(
	authConfigs *configs.AuthConfig,
	paymentGatewayConfigs *configs.PaymentGatewayConfig,
	cacheStore *paymentCacheStore.CacheStore,
	eventBus *paymentRMQ.EventBus,
	logger *logrus.Logger) (*ApiHandler, error) {
	if authConfigs == nil {
		return nil, errors.New("Auth configs is nil")
	}
	if paymentGatewayConfigs == nil {
		return nil, errors.New("Payment gateway configs is nil")
	}
	if cacheStore == nil {
		return nil, errors.New("Cache store is nil")
	}
	store := cacheStore.Store
	if store == nil {
		return nil, errors.New("Store is nil from cache store")
	}
	if eventBus == nil {
		return nil, errors.New("Event bus is nil")
	}
	if logger == nil {
		return nil, errors.New("Logger is nil")
	}
	return &ApiHandler{
		authConfigs:           authConfigs,
		paymentGatewayConfigs: paymentGatewayConfigs,
		cacheStore:            cacheStore,
		store:                 store,
		eventBus:              eventBus,
		logger:                logger,
	}, nil
}
