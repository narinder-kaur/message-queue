// @title Metrics Service API
// @version 1.0
// @description API for querying GPU telemetry and listing GPUs.
// @contact.name Narinder Kaur
// @contact.email dummy@hotmail.com
// @license.name Apache 2.0
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/message-streaming-app/internal/common"
	"github.com/message-streaming-app/internal/metrics"
	"github.com/message-streaming-app/internal/storage"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Parse configuration from environment
	config := newConfigFromEnv()

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	store, err := storage.NewMongoStore(ctx, config.MongoURI, config.DBName, config.Collection)
	cancel()
	if err != nil {
		logger.Error("failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Close(ctx)
	}()

	// Create authenticator
	authenticator := metrics.NewTokenAuthenticator(config.AuthToken)

	// Create sorter
	sorter := metrics.NewStringSorter()

	// Create store adapter to wrap MongoStore to our Store interface
	storeAdapter := metrics.NewStoreAdapter(
		store.QueryTelemetry,
		store.ListGPUs,
		store.Close,
	)

	// Create handler
	handler := metrics.NewHandler(
		storeAdapter,
		logger,
		authenticator,
		sorter,
		config.DefaultLimit,
	)

	// Setup Gin router and register routes (adapter preserves existing handler logic)
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	metrics.RegisterGinRoutes(router, handler, logger, authenticator)

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	logger.Info("starting metrics service", "addr", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Run server in background
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	logger.Info("shutting down metrics server")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Error("metrics server shutdown error", "error", err)
	}
}

// config holds the application configuration
type config struct {
	Port         string
	MongoURI     string
	DBName       string
	Collection   string
	AuthToken    string
	DefaultLimit int
}

// newConfigFromEnv creates configuration from environment variables
func newConfigFromEnv() *config {
	defaultLimit := 100
	if v := os.Getenv("DEFAULT_PAGE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			defaultLimit = n
		}
	}

	return &config{
		Port:         common.GetEnv("METRICS_PORT", "8080"),
		MongoURI:     common.GetEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DBName:       common.GetEnv("MONGODB_DATABASE", "message_streaming"),
		Collection:   common.GetEnv("MONGO_COLLECTION", "metrics"),
		AuthToken:    os.Getenv("AUTH_TOKEN"),
		DefaultLimit: defaultLimit,
	}
}

// chainMiddleware chains multiple HTTP middleware functions
// chainMiddleware removed; Gin adapter is used instead to keep handlers unchanged
