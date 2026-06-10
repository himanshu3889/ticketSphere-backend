package main

import (
	"context"
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

	"github.com/himanshu3889/ticketSphere-backend/base/databases"
	redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
	"github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/ticketSphere-backend/base/middlewares"
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

func NewServer(cfg *configs.Configs, logger *logrus.Logger) (*Server, error) {
	logger.Info("Creating new server...")

	srvr := &Server{
		addr:   cfg.Server.Addr,
		logger: logger,
	}

	// Init Postgres
	var err error
	srvr.postgresDB, err = database.InitPostgresDB(cfg.App, srvr.logger)
	if err != nil {
		return nil, err
	}

	// Init Redis
	err, srvr.redisClient, srvr.cacheManager, srvr.bloomManager = redisDatabase.InitRedis(cfg.Redis, srvr.logger)
	if err != nil {
		return nil, err
	}

	// Init Snowflake
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
	apiHandler, err := inventoryApi.New(&cfg.App.Auth, cachedStore, srvr.logger)
	if err != nil {
		return nil, err
	}

	// Wire routes
	router := gin.New()
	router.Use(middlewares.GatewayTrafficMiddleware(cfg.App.Auth.GatewaySecret))
	router.Use(middlewares.RequestIDMiddleware())
	router.Use(gin.Recovery())

	// Prometheus Metrics, a protected group for Prometheus
	metricsGroup := router.Group("/", gin.BasicAuth(gin.Accounts{
		cfg.App.Prometheus.User: cfg.App.Prometheus.Password,
	}))
	metricsGroup.GET("/metrics", gin.WrapH(promhttp.Handler()))

	inventoryApi.SetupRoutes(router, apiHandler)

	srvr.server = &http.Server{
		Addr:    srvr.addr,
		Handler: router,
	}

	return srvr, nil
}

func (srvr *Server) Start() error {
	srvr.logger.Infof("Inventory server starting on %s", srvr.addr)
	return srvr.server.ListenAndServe()
}

func (srvr *Server) Shutdown(ctx context.Context) error {
	return srvr.server.Shutdown(ctx)
}

func (srvr *Server) Run() {
	// Start HTTP server
	go func() {
		if err := srvr.Start(); err != nil && err != http.ErrServerClosed {
			srvr.logger.Fatalf("Inventory server error: %v", err)
		}
	}()

	// Wait for signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	srvr.logger.Info("Shutting down inventory server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srvr.Shutdown(ctx); err != nil {
		srvr.logger.Errorf("Shutdown error: %v", err)
	}

	// Close all connections
	if err := database.ClosePostgresConnection(srvr.postgresDB); err != nil {
		srvr.logger.Errorf("Postgres close error: %v", err)
	}
	if err := redisDatabase.CloseRedisConnection(srvr.redisClient); err != nil {
		srvr.logger.Errorf("Redis close error: %v", err)
	}

	srvr.logger.Info("Inventory server exited")
}
