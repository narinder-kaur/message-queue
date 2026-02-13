package message

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Message is the JSON format used by producers and consumers.
type Message struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source,omitempty"`
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// New creates a message with generated ID and current timestamp.
func New(typ string, payload json.RawMessage, source string) *Message {
	return &Message{
		ID:        newID(),
		Type:      typ,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
		Source:    source,
	}
}

// MarshalJSON implements json.Marshaler so Timestamp is RFC3339.
func (m Message) MarshalJSON() ([]byte, error) {
	type alias Message
	return json.Marshal(struct {
		alias
		Timestamp string `json:"timestamp"`
	}{
		alias:     alias(m),
		Timestamp: m.Timestamp.Format(time.RFC3339Nano),
	})
}

// UnmarshalJSON implements json.Unmarshaler for RFC3339 timestamp.
func (m *Message) UnmarshalJSON(data []byte) error {
	type alias Message
	aux := struct {
		*alias
		Timestamp string `json:"timestamp"`
	}{alias: (*alias)(m)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Timestamp != "" {
		m.Timestamp, _ = time.Parse(time.RFC3339Nano, aux.Timestamp)
	}
	return nil
}
