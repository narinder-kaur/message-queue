package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/message-streaming-app/internal/message"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mockCursor implements CursorAPI for tests
type mockCursor struct {
	docs []bson.M
	idx  int
	err  error
}

func (m *mockCursor) Close(ctx context.Context) error { return nil }
func (m *mockCursor) Next(ctx context.Context) bool {
	return m.idx < len(m.docs)
}
func (m *mockCursor) Decode(v interface{}) error {
	if m.idx >= len(m.docs) {
		return errors.New("no more docs")
	}
	b, _ := json.Marshal(m.docs[m.idx])
	m.idx++
	return json.Unmarshal(b, v)
}
func (m *mockCursor) Err() error { return m.err }

// mockCollection implements CollectionAPI for tests
type mockCollection struct {
	insertErr     error
	lastInserted  interface{}
	distinctUUID  []interface{}
	distinctGPUID []interface{}
	distinctErr   error
	countRes      int64
	countErr      error
	findCursor    CursorAPI
	findErr       error
	lastFilter    interface{}
	lastFindOpts  *options.FindOptions
}

func (m *mockCollection) InsertOne(ctx context.Context, doc interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	if m.insertErr != nil {
		return nil, m.insertErr
	}
	m.lastInserted = doc
	return &mongo.InsertOneResult{InsertedID: "mockid"}, nil
}
func (m *mockCollection) Distinct(ctx context.Context, field string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error) {
	if m.distinctErr != nil {
		return nil, m.distinctErr
	}
	if field == "uuid" {
		return m.distinctUUID, nil
	}
	return m.distinctGPUID, nil
}
func (m *mockCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	m.lastFilter = filter
	return m.countRes, m.countErr
}
func (m *mockCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CursorAPI, error) {
	m.lastFilter = filter
	if len(opts) > 0 && opts[0] != nil {
		m.lastFindOpts = opts[0]
	}
	return m.findCursor, m.findErr
}

func TestStoreMetricAddsTimestampAndInserts(t *testing.T) {
	ms := &MongoStore{collection: &mockCollection{}}

	payload := map[string]interface{}{"value": 1}
	if err := ms.StoreMetric(payload); err != nil {
		t.Fatalf("StoreMetric failed: %v", err)
	}
	if _, ok := payload["created_at"]; !ok {
		t.Fatalf("expected created_at to be set on metric")
	}
}

func TestStoreMessageUnmarshalError(t *testing.T) {
	ms := &MongoStore{collection: &mockCollection{}}
	bad := message.Message{Payload: []byte("not json"), Timestamp: time.Time{}}
	// make invalid JSON by passing raw text that's invalid
	if err := ms.StoreMessage(bad); err == nil {
		t.Fatalf("expected unmarshal error, got nil")
	}
}

func TestListGPUsUsesUUIDThenFallback(t *testing.T) {
	// Case 1: UUIDs present
	mc1 := &mockCollection{distinctUUID: []interface{}{"uuid1", "uuid2"}}
	ms1 := &MongoStore{collection: mc1}
	ids, err := ms1.ListGPUs(context.Background())
	if err != nil {
		t.Fatalf("ListGPUs failed: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}

	// Case 2: No uuids -> fallback to gpu_id including numeric values
	// Case 2: No uuids -> fallback to gpu_id including numeric values
	mc3 := &mockCollection{distinctUUID: []interface{}{}, distinctGPUID: []interface{}{"123", int32(456)}}
	ms3 := &MongoStore{collection: mc3}
	ids2, err := ms3.ListGPUs(context.Background())
	if err != nil {
		t.Fatalf("ListGPUs fallback failed: %v", err)
	}
	if len(ids2) != 2 {
		t.Fatalf("expected 2 ids from fallback, got %d", len(ids2))
	}
}

func TestListGPUsEmptyResults(t *testing.T) {
	mc := &mockCollection{distinctUUID: []interface{}{}, distinctGPUID: []interface{}{}}
	ms := &MongoStore{collection: mc}
	ids, err := ms.ListGPUs(context.Background())
	if err != nil {
		t.Fatalf("ListGPUs failed: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected 0 ids, got %d", len(ids))
	}
}

func TestQueryTelemetryReturnsDocsAndTotal(t *testing.T) {
	docs := []bson.M{{"gpu_id": "g1"}, {"gpu_id": "g1"}}
	mc := &mockCollection{countRes: 2, findCursor: &mockCursor{docs: docs}}
	ms := &MongoStore{collection: mc}
	res, total, err := ms.QueryTelemetry(context.Background(), "g1", nil, nil, 10, 1, false)
	if err != nil {
		t.Fatalf("QueryTelemetry failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
}

func TestQueryTelemetryTimeRangeAndPagination(t *testing.T) {
	// prepare 5 docs
	docs := []bson.M{{"gpu_id": "g1"}, {"gpu_id": "g1"}, {"gpu_id": "g1"}, {"gpu_id": "g1"}, {"gpu_id": "g1"}}
	mc := &mockCollection{countRes: 5, findCursor: &mockCursor{docs: docs}}
	ms := &MongoStore{collection: mc}

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	// request page 2 with limit 2 -> should set skip=(2-1)*2=2, limit=2
	res, total, err := ms.QueryTelemetry(context.Background(), "g1", &start, &end, 2, 2, true)
	if err != nil {
		t.Fatalf("QueryTelemetry failed: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total 5, got %d", total)
	}
	if len(res) != 5 { // mockCursor returns all docs; we assert that Find options were set correctly instead
		// still acceptable; ensure we captured opts
	}

	// Assert the mock captured the filter and options
	if mc.lastFilter == nil {
		t.Fatalf("expected lastFilter to be set")
	}
	// ensure timestamp filter present
	if ff, ok := mc.lastFilter.(bson.M); ok {
		if _, ok := ff["timestamp"]; !ok {
			t.Fatalf("expected timestamp filter in lastFilter")
		}
	} else {
		t.Fatalf("unexpected filter type: %T", mc.lastFilter)
	}

	if mc.lastFindOpts == nil {
		t.Fatalf("expected lastFindOpts to be set")
	}
	if mc.lastFindOpts.Limit == nil || *mc.lastFindOpts.Limit != int64(2) {
		t.Fatalf("expected Limit 2, got %v", mc.lastFindOpts.Limit)
	}
	if mc.lastFindOpts.Skip == nil || *mc.lastFindOpts.Skip != int64(2) {
		t.Fatalf("expected Skip 2, got %v", mc.lastFindOpts.Skip)
	}
}

func TestQueryTelemetryCountError(t *testing.T) {
	mc := &mockCollection{countErr: fmt.Errorf("boom")}
	ms := &MongoStore{collection: mc}
	_, _, err := ms.QueryTelemetry(context.Background(), "g1", nil, nil, 10, 1, false)
	if err == nil {
		t.Fatalf("expected count error, got nil")
	}
}

func TestStoreMetricInsertError(t *testing.T) {
	mc := &mockCollection{insertErr: fmt.Errorf("insert fail")}
	ms := &MongoStore{collection: mc}
	payload := map[string]interface{}{"v": 1}
	if err := ms.StoreMetric(payload); err == nil {
		t.Fatalf("expected insert error, got nil")
	}
}

func TestStoreMessageUsesMessageTimestamp(t *testing.T) {
	mc := &mockCollection{}
	ms := &MongoStore{collection: mc}
	ts := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	payload := map[string]interface{}{"a": "b"}
	b, _ := json.Marshal(payload)
	msg := message.Message{Payload: b, Timestamp: ts}
	if err := ms.StoreMessage(msg); err != nil {
		t.Fatalf("StoreMessage failed: %v", err)
	}
	// check lastInserted captured by mock
	doc, ok := mc.lastInserted.(map[string]interface{})
	if !ok {
		t.Fatalf("expected lastInserted to be a map, got %T", mc.lastInserted)
	}
	if v, ok := doc["timestamp"]; !ok {
		t.Fatalf("expected timestamp in stored doc")
	} else {
		if tv, ok := v.(time.Time); ok {
			if !tv.Equal(ts) {
				t.Fatalf("expected timestamp %v, got %v", ts, tv)
			}
		}
	}
}

// badCursor simulates a cursor whose Decode returns an error
type badCursor struct{}

func (b *badCursor) Close(ctx context.Context) error { return nil }
func (b *badCursor) Next(ctx context.Context) bool   { return true }
func (b *badCursor) Decode(v interface{}) error      { return fmt.Errorf("decode fail") }
func (b *badCursor) Err() error                      { return nil }

func TestQueryTelemetryFindErrorAndDecodeError(t *testing.T) {
	// Find error
	mc1 := &mockCollection{countRes: 1, findErr: fmt.Errorf("find boom")}
	ms1 := &MongoStore{collection: mc1}
	if _, _, err := ms1.QueryTelemetry(context.Background(), "g1", nil, nil, 10, 1, false); err == nil {
		t.Fatalf("expected find error, got nil")
	}

	// Decode error from cursor
	mc2 := &mockCollection{countRes: 1, findCursor: &badCursor{}}
	ms2 := &MongoStore{collection: mc2}
	if _, _, err := ms2.QueryTelemetry(context.Background(), "g1", nil, nil, 10, 1, false); err == nil {
		t.Fatalf("expected decode error, got nil")
	}
}
