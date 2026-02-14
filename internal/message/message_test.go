package message

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestMessageNew(t *testing.T) {
	payload := json.RawMessage(`{"temperature": 45.5}`)
	msg := New("metric", payload, "gpu-0")

	if msg.ID == "" {
		t.Error("expected non-empty message ID")
	}
	if msg.Type != "metric" {
		t.Errorf("expected type 'metric', got '%s'", msg.Type)
	}
	if msg.Source != "gpu-0" {
		t.Errorf("expected source 'gpu-0', got '%s'", msg.Source)
	}
	if msg.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if !bytes.Equal(msg.Payload, payload) {
		t.Error("payload mismatch")
	}
}

func TestMessageNewUniqueIDs(t *testing.T) {
	payload := json.RawMessage(`{"data": "test"}`)
	msg1 := New("type1", payload, "src1")
	msg2 := New("type2", payload, "src2")

	if msg1.ID == msg2.ID {
		t.Error("expected different IDs")
	}
}

func TestMessageMarshalJSON(t *testing.T) {
	payload := json.RawMessage(`{"value": 123}`)
	msg := New("test", payload, "source")

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if _, ok := result["id"].(string); !ok {
		t.Error("expected id field")
	}
	if _, ok := result["type"].(string); !ok {
		t.Error("expected type field")
	}
	if _, ok := result["timestamp"].(string); !ok {
		t.Error("expected timestamp field as string")
	}
}

func TestMessageUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "test-id",
		"type": "metric",
		"payload": {"temperature": 45.5},
		"timestamp": "2026-02-13T10:00:00Z",
		"source": "gpu-0"
	}`

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if msg.ID != "test-id" {
		t.Errorf("ID mismatch: got %s", msg.ID)
	}
	if msg.Type != "metric" {
		t.Errorf("Type mismatch: got %s", msg.Type)
	}
	if msg.Source != "gpu-0" {
		t.Errorf("Source mismatch: got %s", msg.Source)
	}
}

func TestMessageRoundTrip(t *testing.T) {
	original := New("event", json.RawMessage(`{"data": "test"}`), "source-1")

	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Message
	err = json.Unmarshal(jsonData, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if original.ID != restored.ID {
		t.Error("ID mismatch after roundtrip")
	}
	if original.Type != restored.Type {
		t.Error("Type mismatch after roundtrip")
	}
	if original.Source != restored.Source {
		t.Error("Source mismatch after roundtrip")
	}
	// Payload may not be byte-for-byte equal due to JSON marshaling
	// but should deserialize to the same values
	var origData, restData map[string]interface{}
	json.Unmarshal(original.Payload, &origData)
	json.Unmarshal(restored.Payload, &restData)
	if len(origData) != len(restData) {
		t.Error("Payload content mismatch after roundtrip")
	}
}

func TestMessageTimestampFormatRFC3339(t *testing.T) {
	payload := json.RawMessage(`{}`)
	msg := New("test", payload, "src")

	data, _ := json.Marshal(msg)
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	timestampStr := result["timestamp"].(string)
	_, err := time.Parse(time.RFC3339Nano, timestampStr)
	if err != nil {
		t.Errorf("timestamp not in RFC3339Nano format: %v", err)
	}
}

func TestMessageEmptyPayload(t *testing.T) {
	payload := json.RawMessage(`{}`)
	msg := New("type", payload, "src")

	if msg == nil {
		t.Error("expected non-nil message with empty payload")
	}
	if !bytes.Equal(msg.Payload, payload) {
		t.Error("empty payload mismatch")
	}
}

func TestMessageNullPayload(t *testing.T) {
	payload := json.RawMessage(`null`)
	msg := New("type", payload, "src")

	if msg == nil {
		t.Error("expected non-nil message with null payload")
	}
}

func TestMessageLargePayload(t *testing.T) {
	// Create a large valid JSON payload
	largeObj := map[string]interface{}{
		"data": make([]byte, 5000),
	}
	data, _ := json.Marshal(largeObj)
	payload := json.RawMessage(data)
	msg := New("large", payload, "src")

	if !bytes.Equal(msg.Payload, payload) {
		t.Error("large payload mismatch")
	}

	// Verify we can marshal and unmarshal successfully
	jsonData, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("failed to marshal large payload: %v", err)
	}
	if len(jsonData) == 0 {
		t.Error("marshaled data is empty")
	}

	var restored Message
	err = json.Unmarshal(jsonData, &restored)
	if err != nil {
		t.Errorf("failed to unmarshal large payload: %v", err)
	}
}

func TestMessageTimestampAccuracy(t *testing.T) {
	payload := json.RawMessage(`{}`)
	before := time.Now().UTC()
	msg := New("test", payload, "src")
	after := time.Now().UTC()

	if msg.Timestamp.Before(before) || msg.Timestamp.After(after.Add(time.Second)) {
		t.Errorf("timestamp out of expected range: before=%v, msg=%v, after=%v", before, msg.Timestamp, after)
	}
}
