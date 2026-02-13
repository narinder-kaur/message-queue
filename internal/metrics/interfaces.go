package metrics

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Logger type alias for slog.Logger
type Logger = *slog.Logger

// Error definitions for metrics package
var (
	ErrInvalidLimit = errors.New("limit must be greater than 0")
	ErrInvalidPage  = errors.New("page must be greater than 0")
	ErrInvalidGPUID = errors.New("GPU ID is required")
	ErrInvalidTime  = errors.New("invalid time format")
)

// Store defines the interface for metric data storage operations
type Store interface {
	// ListGPUs returns all GPU IDs
	ListGPUs(ctx context.Context) ([]string, error)

	// QueryTelemetry retrieves telemetry data for a specific GPU
	// Returns items (as interface{}), total count (as int), and error
	QueryTelemetry(
		ctx context.Context,
		gpuID string,
		startTime, endTime *time.Time,
		limit, page int,
		sortAsc bool,
	) (interface{}, int, error)

	// Close closes the store connection
	Close(ctx context.Context) error
}

// SortOrder defines the sort order direction
type SortOrder int

const (
	SortAscending SortOrder = iota
	SortDescending
)

// PaginationParams holds pagination request parameters
type PaginationParams struct {
	Limit int
	Page  int
}

// Validate checks if pagination parameters are valid
func (p PaginationParams) Validate() error {
	if p.Limit <= 0 {
		return ErrInvalidLimit
	}
	if p.Page <= 0 {
		return ErrInvalidPage
	}
	return nil
}

// TelemetryQuery holds all telemetry query parameters
type TelemetryQuery struct {
	GPUID      string
	StartTime  *time.Time
	EndTime    *time.Time
	Pagination PaginationParams
	SortOrder  SortOrder
}

// GPUListRequest holds GPU listing request parameters
type GPUListRequest struct {
	Pagination PaginationParams
	SortOrder  SortOrder
}

// ListResponse is a generic paginated response structure
type ListResponse struct {
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
	Items interface{} `json:"items"`
}

// QueryResponse is a generic query response structure
type QueryResponse struct {
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
	Items interface{} `json:"items"`
}

// Authenticator defines the interface for authentication
type Authenticator interface {
	// Authenticate checks if the token is valid
	Authenticate(token string) bool
}

// Sorter defines the interface for sorting operations
type Sorter interface {
	// SortStrings sorts a slice of strings according to the order
	SortStrings(items []string, order SortOrder)
}

// QueryParser defines the interface for parsing query parameters
type QueryParser interface {
	// ParseInt parses an integer from query parameters
	ParseInt(key string, defaultVal int) int

	// ParseSortOrder parses sort order from query parameters
	ParseSortOrder(key string, defaultVal SortOrder) SortOrder

	// ParseTimeParam parses a time parameter from query
	ParseTimeParam(key string) (*time.Time, error)
}

// HTTPHandler defines the HTTP handler interface
type HTTPHandler interface {
	// ListGPUs handles the GET /gpus endpoint
	ListGPUs(w http.ResponseWriter, r *http.Request)

	// QueryGPUTelemetry handles the GET /gpus/{id}/telemetry endpoint
	QueryGPUTelemetry(w http.ResponseWriter, r *http.Request)
}

// Middleware defines the interface for HTTP middleware
type Middleware interface {
	// Handler wraps the next HTTP handler
	Handler(next http.Handler) http.Handler
}
