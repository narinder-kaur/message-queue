package metrics

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Handler implements the HTTPHandler interface and manages metrics API requests
type Handler struct {
	store         Store
	logger        Logger
	authenticator Authenticator
	sorter        Sorter
	defaultLimit  int
	service       *Service
}

// NewHandler creates a new metrics handler
func NewHandler(
	store Store,
	logger Logger,
	authenticator Authenticator,
	sorter Sorter,
	defaultLimit int,
) *Handler {
	if defaultLimit <= 0 {
		defaultLimit = 100
	}
	svc := NewService(store, sorter, defaultLimit)
	return &Handler{
		store:         store,
		logger:        logger,
		authenticator: authenticator,
		sorter:        sorter,
		defaultLimit:  defaultLimit,
		service:       svc,
	}
}

// ListGPUs handles the GET /gpus endpoint
func (h *Handler) ListGPUs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	parser := NewDefaultQueryParser(r.URL.Query())

	limit := parser.ParseInt("limit", h.defaultLimit)
	page := parser.ParseInt("page", 1)
	sortOrder := parser.ParseSortOrder("sort", SortAscending)

	// Validate pagination
	params := PaginationParams{Limit: limit, Page: page}
	if err := params.Validate(); err != nil {
		h.logger.Error("invalid pagination parameters", "error", err)
		h.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Delegate business logic to service
	ctx := r.Context()
	response, err := h.service.ListGPUs(ctx, page, limit, sortOrder)
	if err != nil {
		h.logger.Error("failed to list GPUs", "error", err)
		h.jsonError(w, http.StatusInternalServerError, "failed to list GPUs")
		return
	}

	h.jsonOK(w, response)
	h.logger.Info("listed GPUs", "total", response.Total, "page", response.Page, "limit", response.Limit)
}

// QueryGPUTelemetry handles the GET /gpus/{id}/telemetry endpoint
func (h *Handler) QueryGPUTelemetry(w http.ResponseWriter, r *http.Request) {
	// Extract GPU ID from URL path
	gpuID := extractGPUIDFromPath(r.URL.Path)
	if gpuID == "" {
		h.logger.Warn("missing GPU ID in request")
		h.jsonError(w, http.StatusBadRequest, "GPU ID is required")
		return
	}

	// Parse query parameters
	parser := NewDefaultQueryParser(r.URL.Query())

	// Parse time parameters
	startTime, err := parser.ParseTimeParam("start_time")
	if err != nil {
		h.logger.Warn("invalid start_time parameter", "error", err)
		h.jsonError(w, http.StatusBadRequest, "invalid start_time format")
		return
	}

	endTime, err := parser.ParseTimeParam("end_time")
	if err != nil {
		h.logger.Warn("invalid end_time parameter", "error", err)
		h.jsonError(w, http.StatusBadRequest, "invalid end_time format")
		return
	}

	// Parse pagination and sort
	limit := parser.ParseInt("limit", h.defaultLimit)
	page := parser.ParseInt("page", 1)
	sortOrder := parser.ParseSortOrder("sort", SortAscending)

	// Validate pagination
	params := PaginationParams{Limit: limit, Page: page}
	if err := params.Validate(); err != nil {
		h.logger.Error("invalid pagination parameters", "error", err)
		h.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Delegate business logic to service
	ctx := r.Context()
	response, err := h.service.QueryTelemetry(ctx, gpuID, startTime, endTime, limit, page, sortOrder)
	if err != nil {
		h.logger.Error("failed to query telemetry", "gpu_id", gpuID, "error", err)
		h.jsonError(w, http.StatusInternalServerError, "failed to query telemetry")
		return
	}

	h.jsonOK(w, response)
	h.logger.Info("queried telemetry", "gpu_id", gpuID, "total", response.Total, "page", response.Page)
}

// jsonOK writes a successful JSON response
func (h *Handler) jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

// jsonError writes an error JSON response
func (h *Handler) jsonError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// extractGPUIDFromPath extracts the GPU ID from the URL path
// Example: /api/v1/gpus/gpu-123/telemetry -> gpu-123
func extractGPUIDFromPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "gpus" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			latency := time.Since(start)
			logger.Info(
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"latency_ms", latency.Milliseconds(),
			)
		})
	}
}

// AuthMiddleware checks authentication token from Authorization header
func AuthMiddleware(authenticator Authenticator, logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			// Parse Bearer token
			var token string
			if authHeader != "" {
				parts := strings.Fields(authHeader)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					token = parts[1]
				}
			}

			if !authenticator.Authenticate(token) {
				logger.Warn("authentication failed", "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (r *responseWriter) WriteHeader(code int) {
	if !r.written {
		r.statusCode = code
		r.written = true
		r.ResponseWriter.WriteHeader(code)
	}
}

func (r *responseWriter) Write(b []byte) (int, error) {
	if !r.written {
		r.written = true
	}
	return r.ResponseWriter.Write(b)
}
