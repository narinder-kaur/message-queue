package metrics

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// Simple mocks
type failingStore struct{ err error }

func (f *failingStore) ListGPUs(ctx context.Context) ([]string, error) { return nil, f.err }
func (f *failingStore) QueryTelemetry(ctx context.Context, gpuID string, startTime, endTime *time.Time, limit, page int, sortAsc bool) (interface{}, int, error) {
	return nil, 0, f.err
}
func (f *failingStore) Close(ctx context.Context) error { return nil }

type okStore struct {
	ids   []string
	items interface{}
	total int
}

func (o *okStore) ListGPUs(ctx context.Context) ([]string, error) { return o.ids, nil }
func (o *okStore) QueryTelemetry(ctx context.Context, gpuID string, startTime, endTime *time.Time, limit, page int, sortAsc bool) (interface{}, int, error) {
	return o.items, o.total, nil
}
func (o *okStore) Close(ctx context.Context) error { return nil }

type passthroughAuth bool

func (p passthroughAuth) Authenticate(token string) bool { return bool(p) }

type nopSorter struct{}

func (n nopSorter) SortStrings(items []string, order SortOrder) {}

func TestExtractGPUIDFromPath(t *testing.T) {
	cases := map[string]string{
		"/api/v1/gpus/gpu-123/telemetry": "gpu-123",
		"/gpus/abc/telemetry":            "abc",
		"/gpus/":                         "",
		"/no/gpus/here":                  "here",
	}
	for path, want := range cases {
		if got := extractGPUIDFromPath(path); got != want {
			t.Fatalf("extractGPUIDFromPath(%s) = %s, want %s", path, got, want)
		}
	}
}

func TestListGPUsHandlerSuccessAndErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	sorter := nopSorter{}
	auth := passthroughAuth(true)

	// success
	store := &okStore{ids: []string{"a", "b", "c", "d"}}
	h := NewHandler(store, logger, auth, sorter, 2)
	req := httptest.NewRequest(http.MethodGet, "/gpus?limit=2&page=2", nil)
	rr := httptest.NewRecorder()
	h.ListGPUs(rr, req)
	res := rr.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body ListResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Total != 4 || body.Page != 2 || body.Limit != 2 {
		t.Fatalf("unexpected body: %+v", body)
	}

	// limit=0 falls back to default via parser; expect OK
	req2 := httptest.NewRequest(http.MethodGet, "/gpus?limit=0", nil)
	rr2 := httptest.NewRecorder()
	h.ListGPUs(rr2, req2)
	if rr2.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for limit=0 fallback, got %d", rr2.Result().StatusCode)
	}

	// store error
	storeErr := &failingStore{err: io.ErrUnexpectedEOF}
	h2 := NewHandler(storeErr, logger, auth, sorter, 10)
	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/gpus", nil)
	h2.ListGPUs(rr3, req3)
	if rr3.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 for store error")
	}
}

func TestQueryGPUTelemetryHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	sorter := nopSorter{}
	auth := passthroughAuth(true)

	// missing GPU ID
	h := NewHandler(&okStore{}, logger, auth, sorter, 10)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/gpus//telemetry", nil)
	h.QueryGPUTelemetry(rr, req)
	if rr.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing gpu id")
	}

	// invalid start_time
	req2 := httptest.NewRequest(http.MethodGet, "/gpus/g1/telemetry?start_time=bad", nil)
	rr2 := httptest.NewRecorder()
	h.QueryGPUTelemetry(rr2, req2)
	if rr2.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid time")
	}

	// store error
	hErr := NewHandler(&failingStore{err: io.ErrUnexpectedEOF}, logger, auth, sorter, 10)
	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/gpus/g1/telemetry", nil)
	hErr.QueryGPUTelemetry(rr3, req3)
	if rr3.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 for store error")
	}

	// success
	items := []map[string]interface{}{{"x": 1}}
	store := &okStore{items: items, total: 1}
	hs := NewHandler(store, logger, auth, sorter, 5)
	rr4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodGet, "/api/gpus/g1/telemetry?limit=5&page=1", nil)
	hs.QueryGPUTelemetry(rr4, req4)
	if rr4.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for success")
	}
	var body QueryResponse
	if err := json.NewDecoder(rr4.Result().Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Total != 1 || body.Page != 1 {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestMiddlewares(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	// LoggingMiddleware should pass through and allow status capture
	mw := LoggingMiddleware(logger)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("ok"))
	})
	rr := httptest.NewRecorder()
	mw(handler).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Result().StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Result().StatusCode)
	}

	// AuthMiddleware should reject unauthorized
	auth := passthroughAuth(false)
	amw := AuthMiddleware(auth, logger)
	rr2 := httptest.NewRecorder()
	amw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})).ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr2.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr2.Result().StatusCode)
	}
}
