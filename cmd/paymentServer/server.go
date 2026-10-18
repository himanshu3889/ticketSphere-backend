package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	database "github.com/himanshu3889/ticketSphere-backend/base/databases"
	baseRabbitMQ "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/rabbitmq"
	redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/ticketSphere-backend/base/middlewares"
	"github.com/himanshu3889/ticketSphere-backend/base/otelTrace"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/configs"
	paymentApi "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/api"
	paymentCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/cacheStore"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/paymentGateway"
	paymentRMQ "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/rmq"
	paymentConsumers "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/rmq/consumers"
	paymentStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/store"
)

type Server struct {
	addr   string
	server *http.Server

	// Dependencies this server owns and must close on shutdown
	postgresDB         *sqlx.DB
	redisClient        *redis.Client
	cacheManager       *redisDatabase.CacheManager
	bloomManager       *bloomFilter.BloomManager
	rabbitMQConnection *amqp091.Connection
	logger             *logrus.Logger
}

func NewServer(cfg *configs.PaymentConfig, logger *logrus.Logger) (*Server, error) {
	logger.Info("Creating new server...")

	srvr := &Server{
		addr:   cfg.Server.Address(),
		logger: logger,
	}

	paymentTp, err := otelTrace.InitTracer(context.Background(), &cfg.OTEL)
	if err != nil {
		return nil, err
	}

	srvr.postgresDB, err = database.InitPostgresDB(&cfg.Postgres, srvr.logger)
	if err != nil {
		return nil, err
	}

	err, srvr.redisClient, srvr.cacheManager, srvr.bloomManager = redisDatabase.InitRedis(&cfg.Redis, srvr.logger, paymentTp)
	if err != nil {
		return nil, err
	}

	snowflakeNode, err := utils.InitSnowflake(cfg.Server.MachineID, srvr.logger)
	if err != nil {
		return nil, err
	}

	conn, err := baseRabbitMQ.InitRabbitMQ(&cfg.RMQ, srvr.logger)
	if err != nil {
		return nil, err
	}
	srvr.rabbitMQConnection = conn

	eventBus := paymentRMQ.GetEventBus(conn, srvr.logger, paymentTp)

	// Build payment module bottom-up
	baseStore, err := paymentStore.New(srvr.postgresDB, snowflakeNode, srvr.logger)
	if err != nil {
		return nil, err
	}

	paymentGatewayTimeout := time.Duration(cfg.PaymentGateway.TimeoutSec) * time.Second
	paymentGatewayConfig := paymentGateway.Config{
		BaseURL:    cfg.PaymentGateway.BaseURL,
		Timeout:    paymentGatewayTimeout,
		HTTPClient: &http.Client{Timeout: paymentGatewayTimeout},
		Logger:     srvr.logger,
		Tracer:     paymentTp.Tracer(constants.TracerPaymentGateway),
	}

	paymentGateway, err := paymentGateway.New(paymentGatewayConfig)
	if err != nil {
		return nil, err
	}

	cachedStore, err := paymentCacheStore.New(baseStore, srvr.cacheManager, srvr.bloomManager, eventBus, paymentGateway, &cfg.PaymentGateway, srvr.logger, paymentTp)
	if err != nil {
		return nil, err
	}
	apiHandler, err := paymentApi.New(&cfg.Auth, &cfg.PaymentGateway, cachedStore, eventBus, srvr.logger)
	if err != nil {
		return nil, err
	}

	paymentConsumers.StartConsumers(srvr.rabbitMQConnection, cachedStore, srvr.logger, paymentTp)

	// Wire routes
	router := gin.New()

	// /metrics behind basic auth
	metricsGroup := router.Group("/", gin.BasicAuth(gin.Accounts{
		cfg.Prometheus.User: cfg.Prometheus.Password,
	}))
	metricsGroup.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes guarded by the gateway secret
	baseApiGroup := router.Group("")
	baseApiGroup.Use(middlewares.GatewayTrafficMiddleware(cfg.Auth.GatewaySecret))
	baseApiGroup.Use(middlewares.RequestIDMiddleware())
	baseApiGroup.Use(gin.Recovery())

	paymentApi.SetupRoutes(baseApiGroup, apiHandler)

	srvr.server = &http.Server{
		Addr:    srvr.addr,
		Handler: router,
	}

	return srvr, nil
}

func (srvr *Server) start() error {
	srvr.logger.Infof("Payment server starting on %s", srvr.addr)
	listener, err := net.Listen("tcp", srvr.addr)
	if err != nil {
		return err
	}
	srvr.logger.Infof("Payment server started successfully on %s", srvr.addr)
	return srvr.server.Serve(listener)
}

func (srvr *Server) Shutdown(ctx context.Context) error {
	return srvr.server.Shutdown(ctx)
}

func (srvr *Server) Run() {
	go func() {
		if err := srvr.start(); err != nil && err != http.ErrServerClosed {
			srvr.logger.Fatalf("Payment server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	srvr.logger.Info("Shutting down payment server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srvr.Shutdown(ctx); err != nil {
		srvr.logger.Errorf("Shutdown error: %v", err)
	}

	if err := database.ClosePostgresConnection(srvr.postgresDB); err != nil {
		srvr.logger.Errorf("Postgres close error: %v", err)
	}
	if err := redisDatabase.CloseRedisConnection(srvr.redisClient); err != nil {
		srvr.logger.Errorf("Redis close error: %v", err)
	}
	if err := srvr.rabbitMQConnection.Close(); err != nil {
		srvr.logger.Errorf("RabbitMQ close error: %v", err)
	}

	srvr.logger.Info("Payment server exited")
}
