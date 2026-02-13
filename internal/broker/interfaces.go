package broker

import "log/slog"

// Logger defines the logging interface used by the broker
type Logger = *slog.Logger

// DeliveryMode defines how messages are distributed to multiple consumers.
type DeliveryMode int

const (
	// Broadcast: every consumer receives the same message (fan-out).
	Broadcast DeliveryMode = iota
	// Queue: each message is delivered to exactly one consumer (competing consumers, round-robin).
	Queue
)

func (m DeliveryMode) String() string {
	switch m {
	case Broadcast:
		return "broadcast"
	case Queue:
		return "queue"
	default:
		return "unknown"
	}
}

// ParseDeliveryMode returns Broadcast for "broadcast", Queue for "queue", and Broadcast for any other value.
func ParseDeliveryMode(s string) DeliveryMode {
	switch s {
	case "queue":
		return Queue
	case "broadcast":
		return Broadcast
	default:
		return Broadcast
	}
}

// MessageQueue defines the interface for a message queue
type MessageQueue interface {
	// Enqueue adds a message to the queue
	Enqueue(msg []byte) error

	// Dequeue retrieves a message from the queue
	Dequeue() ([]byte, error)

	// IsFull checks if the queue is at capacity
	IsFull() bool

	// Len returns the current number of messages in the queue
	Len() int

	// Close closes the queue
	Close() error
}

// ConsumerRegistry defines the interface for managing consumers
type ConsumerRegistry interface {
	// RegisterConsumer adds a channel for a consumer
	RegisterConsumer(ch chan []byte) string

	// UnregisterConsumer removes a consumer by ID
	UnregisterConsumer(id string)

	// BroadcastMessage sends a message to all registered consumers
	BroadcastMessage(msg []byte)

	// GetConsumerCount returns the number of registered consumers
	GetConsumerCount() int
}

// FrameReader defines the interface for reading frames from a connection
type FrameReader interface {
	ReadFrame(buf []byte) ([]byte, error)
}

// FrameWriter defines the interface for writing frames to a connection
type FrameWriter interface {
	WriteFrame(data []byte) error
}

// ConnectionHandler defines the interface for handling a single connection
type ConnectionHandler interface {
	// Handle processes messages from the connection based on its role
	Handle(role string, reader FrameReader, writer FrameWriter) error
}
