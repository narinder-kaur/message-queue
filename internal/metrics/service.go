package metrics

import (
    "context"
    "time"
)

// Service contains business logic for metrics operations separated from HTTP concerns
type Service struct {
    store        Store
    sorter       Sorter
    defaultLimit int
}

// NewService creates a new Service
func NewService(store Store, sorter Sorter, defaultLimit int) *Service {
    if defaultLimit <= 0 {
        defaultLimit = 100
    }
    return &Service{store: store, sorter: sorter, defaultLimit: defaultLimit}
}

// ListGPUs returns a paginated, sorted list of GPU IDs
func (s *Service) ListGPUs(ctx context.Context, page, limit int, order SortOrder) (ListResponse, error) {
    // Validate pagination
    params := PaginationParams{Limit: limit, Page: page}
    if err := params.Validate(); err != nil {
        return ListResponse{}, err
    }

    // Get GPU IDs from store
    gpuIDs, err := s.store.ListGPUs(ctx)
    if err != nil {
        return ListResponse{}, err
    }

    // Sort and paginate
    s.sorter.SortStrings(gpuIDs, order)
    offset := CalculateOffset(page, limit)
    paginated := PaginateSlice(gpuIDs, offset, limit)

    resp := ListResponse{Total: len(gpuIDs), Page: page, Limit: limit, Items: paginated}
    return resp, nil
}

// QueryTelemetry queries telemetry entries for a GPU and returns a paginated response
func (s *Service) QueryTelemetry(ctx context.Context, gpuID string, startTime, endTime *time.Time, limit, page int, order SortOrder) (QueryResponse, error) {
    // Validate pagination
    params := PaginationParams{Limit: limit, Page: page}
    if err := params.Validate(); err != nil {
        return QueryResponse{}, err
    }

    items, total, err := s.store.QueryTelemetry(ctx, gpuID, startTime, endTime, limit, page, order == SortAscending)
    if err != nil {
        return QueryResponse{}, err
    }

    resp := QueryResponse{Total: total, Page: page, Limit: limit, Items: items}
    return resp, nil
}
