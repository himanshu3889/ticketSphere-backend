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
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	database "github.com/himanshu3889/ticketSphere-backend/base/databases"
	redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/ticketSphere-backend/base/middlewares"
	"github.com/himanshu3889/ticketSphere-backend/base/otelTrace"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/configs"
	inventoryApi "github.com/himanshu3889/ticketSphere-backend/internal/modules/inventory/api"
	inventoryCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/inventory/cacheStore"
	inventoryStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/inventory/store"
)

type Server struct {
	addr   string
	server *http.Server

	// Dependencies this server owns and must close on shutdown
	postgresDB   *sqlx.DB
	redisClient  *redis.Client
	cacheManager *redisDatabase.CacheManager
	bloomManager *bloomFilter.BloomManager
	logger       *logrus.Logger
}

func NewServer(cfg *configs.InventoryConfig, logger *logrus.Logger) (*Server, error) {
	logger.Info("Creating new server...")

	srvr := &Server{
		addr:   cfg.Server.Address(),
		logger: logger,
	}

	// Tracing providers
	inventoryTp, err := otelTrace.InitTracer(context.Background(), &cfg.OTEL)
	if err != nil {
		return nil, err
	}

	// Init Postgres
	srvr.postgresDB, err = database.InitPostgresDB(&cfg.Postgres, srvr.logger)
	if err != nil {
		return nil, err
	}

	// Init Redis
	err, srvr.redisClient, srvr.cacheManager, srvr.bloomManager = redisDatabase.InitRedis(&cfg.Redis, srvr.logger, inventoryTp)
	if err != nil {
		return nil, err
	}

	snowflakeNode, err := utils.InitSnowflake(cfg.Server.MachineID, srvr.logger)
	if err != nil {
		return nil, err
	}

	// Build inventory module bottom-up
	baseStore, err := inventoryStore.New(srvr.postgresDB, snowflakeNode, srvr.logger)
	if err != nil {
		return nil, err
	}
	cachedStore, err := inventoryCacheStore.New(baseStore, srvr.cacheManager, srvr.bloomManager, srvr.logger)
	if err != nil {
		return nil, err
	}
	apiHandler, err := inventoryApi.New(&cfg.Auth, cachedStore, srvr.logger)
	if err != nil {
		return nil, err
	}

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

	inventoryApi.SetupRoutes(baseApiGroup, apiHandler)

	srvr.server = &http.Server{
		Addr:    srvr.addr,
		Handler: router,
	}

	return srvr, nil
}

func (srvr *Server) start() error {
	srvr.logger.Infof("Inventory server starting on %s", srvr.addr)
	listener, err := net.Listen("tcp", srvr.addr)
	if err != nil {
		return err
	}
	srvr.logger.Infof("Inventory server started successfully on %s", srvr.addr)
	return srvr.server.Serve(listener)
}

func (srvr *Server) Shutdown(ctx context.Context) error {
	return srvr.server.Shutdown(ctx)
}

func (srvr *Server) Run() {
	go func() {
		if err := srvr.start(); err != nil && err != http.ErrServerClosed {
			srvr.logger.Fatalf("Inventory server error: %v", err)
		}
		srvr.logger.Info("Inventory server started successfully")
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	srvr.logger.Info("Shutting down inventory server...")

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

	srvr.logger.Info("Inventory server exited")
}
