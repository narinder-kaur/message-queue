package metrics

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterGinRoutes registers Gin routes and middleware adapting the existing
// Handler methods (which use http.ResponseWriter and *http.Request) so we don't
// need to rework the core handler logic in one go.
func RegisterGinRoutes(engine *gin.Engine, handler *Handler, logger Logger, authenticator Authenticator) {
	// CORS middleware allows cross-origin requests and handles preflight
	cors := func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept,Origin")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}

	// Register CORS globally so preflight requests are handled before auth
	engine.Use(cors)

	// Gin logging middleware adapted to existing logger
	logging := func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		logger.Info(
			"http request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
		)
	}

	// Gin auth middleware adapted to existing Authenticator
	auth := func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		var token string
		if authHeader != "" {
			parts := strings.Fields(authHeader)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				token = parts[1]
			}
		}

		if !authenticator.Authenticate(token) {
			logger.Warn("authentication failed", "path", c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		c.Next()
	}

	// Group routes under /api/v1 with our middlewares
	v1 := engine.Group("/api/v1")
	v1.Use(logging, auth)

	// register package-level handler pointer used by documented functions
	registeredHandler = handler

	// Adapt existing handler methods by using documented wrapper functions
	v1.GET("/gpus", ListGPUs)

	// Keep the same path shape as before; the existing handler extracts GPU ID from path
	v1.GET("/gpus/:id/telemetry", QueryGPUTelemetry)

	// Health endpoints without auth
	engine.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	engine.GET("/ready", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	// Serve OpenAPI file from working directory and Swagger UI
	engine.GET("/openapi.yaml", func(c *gin.Context) {
		pwd, err := os.Getwd()
		if err != nil {
			logger.Error("failed to get working directory", "error", err)
			c.Status(http.StatusInternalServerError)
			return
		}
		logger.Info("serving OpenAPI spec", "path", pwd+"/docs/openapi.yaml")
		data, err := os.ReadFile(pwd + "/docs/openapi.yaml")
		if err != nil {
			logger.Error("failed to read OpenAPI file", "error", err)
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "application/x-yaml", data)
	})

	// Swagger UI that points to the /openapi.yaml endpoint
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/openapi.yaml")))
}

var registeredHandler *Handler

func ListGPUs(c *gin.Context) {
	if registeredHandler == nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	registeredHandler.ListGPUs(c.Writer, c.Request)
}

func QueryGPUTelemetry(c *gin.Context) {
	if registeredHandler == nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	registeredHandler.QueryGPUTelemetry(c.Writer, c.Request)
}
