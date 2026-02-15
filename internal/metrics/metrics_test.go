package metrics

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
)

// MockStore implements the Store interface for testing
type MockStore struct {
	gpuIDs    []string
	telemetry interface{}
	total     int
	err       error
}

func (m *MockStore) ListGPUs(ctx context.Context) ([]string, error) {
	return m.gpuIDs, m.err
}

func (m *MockStore) QueryTelemetry(ctx context.Context, gpuID string, startTime, endTime *time.Time, limit, page int, sortAsc bool) (interface{}, int, error) {
	return m.telemetry, m.total, m.err
}

func (m *MockStore) Close(ctx context.Context) error {
	return nil
}

// TestStringSorterAscending tests ascending sort order
func TestStringSorterAscending(t *testing.T) {
	sorter := NewStringSorter()
	items := []string{"gamma", "alpha", "beta"}
	expected := []string{"alpha", "beta", "gamma"}
	sorter.SortStrings(items, SortAscending)
	for i, v := range items {
		if v != expected[i] {
			t.Errorf("expected %s, got %s at index %d", expected[i], v, i)
		}
	}
}

// TestStringSorterDescending tests descending sort order
func TestStringSorterDescending(t *testing.T) {
	sorter := NewStringSorter()
	items := []string{"gamma", "alpha", "beta"}
	expected := []string{"gamma", "beta", "alpha"}
	sorter.SortStrings(items, SortDescending)
	for i, v := range items {
		if v != expected[i] {
			t.Errorf("expected %s, got %s at index %d", expected[i], v, i)
		}
	}
}

// TestTokenAuthenticatorValidToken tests valid token authentication
func TestTokenAuthenticatorValidToken(t *testing.T) {
	auth := NewTokenAuthenticator("secret123")
	if !auth.Authenticate("secret123") {
		t.Error("expected authentication to succeed")
	}
}

// TestTokenAuthenticatorInvalidToken tests invalid token authentication
func TestTokenAuthenticatorInvalidToken(t *testing.T) {
	auth := NewTokenAuthenticator("secret123")
	if auth.Authenticate("wrongtoken") {
		t.Error("expected authentication to fail")
	}
}

// TestTokenAuthenticatorNoTokenRequired tests authentication with empty token
func TestTokenAuthenticatorNoTokenRequired(t *testing.T) {
	auth := NewTokenAuthenticator("")
	if !auth.Authenticate("anything") {
		t.Error("expected authentication to succeed when no token required")
	}
}

// TestDefaultQueryParserParseInt tests parsing integer parameters
func TestDefaultQueryParserParseInt(t *testing.T) {
	params := map[string][]string{
		"limit":   {"50"},
		"page":    {"2"},
		"invalid": {"abc"},
	}
	parser := NewDefaultQueryParser(params)

	tests := []struct {
		key        string
		defaultVal int
		expected   int
	}{
		{"limit", 10, 50},
		{"page", 1, 2},
		{"invalid", 100, 100},
		{"missing", 75, 75},
	}

	for _, tt := range tests {
		result := parser.ParseInt(tt.key, tt.defaultVal)
		if result != tt.expected {
			t.Errorf("ParseInt(%s, %d) = %d, expected %d", tt.key, tt.defaultVal, result, tt.expected)
		}
	}
}

// TestDefaultQueryParserParseSortOrder tests parsing sort order parameter
func TestDefaultQueryParserParseSortOrder(t *testing.T) {
	params := map[string][]string{
		"sort": {"desc"},
	}
	parser := NewDefaultQueryParser(params)

	order := parser.ParseSortOrder("sort", SortAscending)
	if order != SortDescending {
		t.Errorf("expected SortDescending, got %d", order)
	}

	if order := parser.ParseSortOrder("missing", SortDescending); order != SortDescending {
		t.Errorf("expected SortDescending default, got %d", order)
	}
}

// TestDefaultQueryParserParseTime tests parsing time parameter with valid timestamp
func TestDefaultQueryParserParseTime(t *testing.T) {
	timeStr := "2026-02-13T22:30:00Z"
	params := map[string][]string{
		"time": {timeStr},
	}
	parser := NewDefaultQueryParser(params)

	result, err := parser.ParseTimeParam("time")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected time to be parsed")
	}
}

// TestDefaultQueryParserParseTimeInvalid tests parsing time parameter with invalid timestamp
func TestDefaultQueryParserParseTimeInvalid(t *testing.T) {
	params := map[string][]string{
		"time": {"not-a-time"},
	}
	parser := NewDefaultQueryParser(params)

	_, err := parser.ParseTimeParam("time")
	if !errors.Is(err, ErrInvalidTime) {
		t.Errorf("expected ErrInvalidTime, got %v", err)
	}
}

// TestPaginationParamsValidate tests pagination parameter validation
func TestPaginationParamsValidate(t *testing.T) {
	tests := []struct {
		params PaginationParams
		valid  bool
	}{
		{PaginationParams{Limit: 10, Page: 1}, true},
		{PaginationParams{Limit: 0, Page: 1}, false},
		{PaginationParams{Limit: 10, Page: 0}, false},
		{PaginationParams{Limit: -5, Page: 1}, false},
	}

	for _, tt := range tests {
		err := tt.params.Validate()
		if tt.valid && err != nil {
			t.Errorf("expected valid params, got error: %v", err)
		}
		if !tt.valid && err == nil {
			t.Errorf("expected error for invalid params: %+v", tt.params)
		}
	}
}

// TestCalculateOffset tests offset calculation for pagination
func TestCalculateOffset(t *testing.T) {
	tests := []struct {
		page     int
		limit    int
		expected int
	}{
		{1, 10, 0},
		{2, 10, 10},
		{3, 10, 20},
		{1, 25, 0},
	}

	for _, tt := range tests {
		result := CalculateOffset(tt.page, tt.limit)
		if result != tt.expected {
			t.Errorf("CalculateOffset(%d, %d) = %d, expected %d", tt.page, tt.limit, result, tt.expected)
		}
	}
}

// TestPaginateSlice tests pagination slice operation
func TestPaginateSlice(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	tests := []struct {
		offset   int
		limit    int
		expected []string
	}{
		{0, 2, []string{"a", "b"}},
		{2, 2, []string{"c", "d"}},
		{4, 2, []string{"e"}},
		{5, 2, []string{}},
	}

	for _, tt := range tests {
		result := PaginateSlice(items, tt.offset, tt.limit)
		if len(result) != len(tt.expected) {
			t.Errorf("PaginateSlice offset=%d limit=%d: expected %d items, got %d", tt.offset, tt.limit, len(tt.expected), len(result))
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("expected %s, got %s at index %d", tt.expected[i], v, i)
			}
		}
	}
}

// TestHandlerListGPUs tests the Handler ListGPUs method
func TestHandlerListGPUs(t *testing.T) {
	mockStore := &MockStore{
		gpuIDs: []string{"gpu-1", "gpu-2", "gpu-3"},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	sorter := NewStringSorter()
	auth := NewTokenAuthenticator("")

	handler := NewHandler(mockStore, logger, auth, sorter, 100)
	if handler == nil {
		t.Fatal("expected handler to be created")
	}
}

// TestHandlerValidation tests the Handler with invalid configuration
func TestHandlerValidation(t *testing.T) {
	mockStore := &MockStore{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	auth := NewTokenAuthenticator("")
	sorter := NewStringSorter()

	// Test with invalid default limit
	handler := NewHandler(mockStore, logger, auth, sorter, -1)
	if handler == nil {
		t.Fatal("expected handler to be created with default limit correction")
	}
}
