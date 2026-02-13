package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/message-streaming-app/internal/storage"
)

func main() {
	port := getEnv("METRICS_PORT", "8080")
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getEnv("MONGO_DB", "message_streaming")
	collection := getEnv("MONGO_COLLECTION", "metrics")
	defaultLimit := 100
	if v := os.Getenv("DEFAULT_PAGE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			defaultLimit = n
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	store, err := storage.NewMongoStore(ctx, mongoURI, dbName, collection)
	cancel()
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = store.Close(ctx)
	}()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(loggerMiddleware())
	r.Use(authMiddleware())

	api := r.Group("/api/v1")
	{
		api.GET("/gpus", func(c *gin.Context) {
			// pagination & sort
			limit := parseQueryInt(c, "limit", defaultLimit)
			page := parseQueryInt(c, "page", 1)
			sort := strings.ToLower(c.DefaultQuery("sort", "asc"))
			sortAsc := sort != "desc"

			ctx := c.Request.Context()
			ids, err := store.ListGPUs(ctx)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// sort
			if sortAsc {
				// ascending alphabetical
				sortStrings(ids)
			} else {
				sortStringsDesc(ids)
			}

			total := len(ids)
			start := (page - 1) * limit
			end := start + limit
			if start > total {
				start = total
			}
			if end > total {
				end = total
			}
			paged := ids[start:end]

			c.JSON(http.StatusOK, gin.H{
				"total": total,
				"page":  page,
				"limit": limit,
				"items": paged,
			})
		})

		api.GET("/gpus/:id/telemetry", func(c *gin.Context) {
			id := c.Param("id")
			// parse times
			var startT, endT *time.Time
			if s := c.Query("start_time"); s != "" {
				if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
					startT = &t
				} else if t, err := time.Parse(time.RFC3339, s); err == nil {
					startT = &t
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format"})
					return
				}
			}
			if s := c.Query("end_time"); s != "" {
				if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
					endT = &t
				} else if t, err := time.Parse(time.RFC3339, s); err == nil {
					endT = &t
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format"})
					return
				}
			}

			limit := parseQueryInt(c, "limit", defaultLimit)
			page := parseQueryInt(c, "page", 1)
			sort := strings.ToLower(c.DefaultQuery("sort", "asc"))
			sortAsc := sort != "desc"

			ctx := c.Request.Context()
			items, total, err := store.QueryTelemetry(ctx, id, startT, endT, limit, page, sortAsc)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"total": total,
				"page":  page,
				"limit": limit,
				"items": items,
			})
		})
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("starting metrics service on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func parseQueryInt(c *gin.Context, key string, def int) int {
	s := c.Query(key)
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		return n
	}
	return def
}

func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		log.Printf("%s %s %d %s", c.Request.Method, c.Request.URL.Path, status, latency)
	}
}

func authMiddleware() gin.HandlerFunc {
	token := os.Getenv("AUTH_TOKEN")
	return func(c *gin.Context) {
		if token == "" {
			c.Next()
			return
		}
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
			return
		}
		// expect Bearer <token>
		parts := strings.Fields(auth)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] == token {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
	}
}

// simple alphabetical sorts
func sortStrings(s []string) {
	// simple insertion sort for small lists
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
func sortStringsDesc(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] < s[j]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
