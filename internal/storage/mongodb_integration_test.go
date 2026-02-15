//go:build integration
// +build integration

package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/message-streaming-app/internal/message"
)

// Integration tests for MongoStore. These run only when the 'integration' build tag
// is used or when explicitly enabled by running `go test -tags=integration ./internal/storage`.

func TestMongoStoreIntegrationSmoke(t *testing.T) {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		t.Skip("MONGO_URI not set; skipping integration test")
	}

	ctx := context.Background()
	coll := "test_metrics_integration"
	ms, err := NewMongoStore(ctx, mongoURI, "message_streaming_test", coll)
	if err != nil {
		t.Fatalf("NewMongoStore failed: %v", err)
	}
	defer ms.Close(ctx)

	// store a message
	payload := map[string]interface{}{"uuid": "integration-test", "value": 1}
	b, _ := json.Marshal(payload)
	msg := message.Message{Payload: b, Timestamp: time.Now().UTC()}
	if err := ms.StoreMessage(msg); err != nil {
		t.Fatalf("StoreMessage failed: %v", err)
	}

	// ListGPUs should at least return our uuid
	ids, err := ms.ListGPUs(ctx)
	if err != nil {
		t.Fatalf("ListGPUs failed: %v", err)
	}
	if len(ids) == 0 {
		t.Fatalf("expected at least one GPU id from ListGPUs")
	}
}
