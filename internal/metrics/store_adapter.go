package metrics

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// StoreAdapter wraps a MongoDB store implementation to match the Store interface
// This allows us to adapt external implementations to our interface boundaries
type StoreAdapter struct {
	queryFunc    func(context.Context, string, *time.Time, *time.Time, int, int, bool) ([]bson.M, int64, error)
	listGPUsFunc func(context.Context) ([]string, error)
	closeFunc    func(context.Context) error
}

// NewStoreAdapter creates a new store adapter
func NewStoreAdapter(
	queryFunc func(context.Context, string, *time.Time, *time.Time, int, int, bool) ([]bson.M, int64, error),
	listGPUsFunc func(context.Context) ([]string, error),
	closeFunc func(context.Context) error,
) *StoreAdapter {
	return &StoreAdapter{
		queryFunc:    queryFunc,
		listGPUsFunc: listGPUsFunc,
		closeFunc:    closeFunc,
	}
}

// ListGPUs returns all GPU IDs
func (a *StoreAdapter) ListGPUs(ctx context.Context) ([]string, error) {
	return a.listGPUsFunc(ctx)
}

// QueryTelemetry retrieves telemetry data for a specific GPU
func (a *StoreAdapter) QueryTelemetry(
	ctx context.Context,
	gpuID string,
	startTime, endTime *time.Time,
	limit, page int,
	sortAsc bool,
) (interface{}, int, error) {
	items, total, err := a.queryFunc(ctx, gpuID, startTime, endTime, limit, page, sortAsc)
	// Convert []bson.M to interface{} and int64 to int
	return items, int(total), err
}

// Close closes the store connection
func (a *StoreAdapter) Close(ctx context.Context) error {
	return a.closeFunc(ctx)
}
