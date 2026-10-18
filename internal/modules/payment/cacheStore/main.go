package paymentCacheStore

import (
	"errors"

	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/ticketSphere-backend/configs"
	paymentGateway "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/paymentGateway"
	paymentRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/rmq"
	paymentStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/store"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

// CacheStore wires the payment cache dependencies.
type CacheStore struct {
	Store                 *paymentStore.Store
	CacheManager          *redisDatabase.CacheManager
	BloomManager          *bloomFilter.BloomManager
	EventBus              *paymentRMQ.EventBus
	PaymentGateway        *paymentGateway.Client
	paymentGatewayConfigs *configs.PaymentGatewayConfig
	logger                *logrus.Logger
	Tracer                trace.Tracer
}

// New validates deps and builds the CacheStore.
func New(store *paymentStore.Store,
	cacheManager *redisDatabase.CacheManager,
	bloomManager *bloomFilter.BloomManager,
	eventBus *paymentRMQ.EventBus,
	paymentGateway *paymentGateway.Client,
	paymentGatewayConfigs *configs.PaymentGatewayConfig,
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
	if paymentGateway == nil {
		return nil, errors.New("Gateway is nil")
	}
	if paymentGatewayConfigs == nil {
		return nil, errors.New("Payment gateway configs is nil")
	}
	if paymentGatewayConfigs.WebhookURL == "" {
		return nil, errors.New("Payment gateway webhook url is empty")
	}
	if logger == nil {
		return nil, errors.New("Logger is nil")
	}
	if eventBus == nil {
		return nil, errors.New("Event bus is nil")
	}

	tracer := traceProvider.Tracer(constants.TracerPaymentCacheStore)

	return &CacheStore{
		Store:                 store,
		CacheManager:          cacheManager,
		BloomManager:          bloomManager,
		EventBus:              eventBus,
		PaymentGateway:        paymentGateway,
		paymentGatewayConfigs: paymentGatewayConfigs,
		logger:                logger,
		Tracer:                tracer,
	}, nil
}
