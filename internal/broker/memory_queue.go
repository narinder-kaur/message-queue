package broker

import (
	"fmt"
)

// MemoryMessageQueue is an in-memory queue implementation
type MemoryMessageQueue struct {
	queue chan []byte
	size  int
	logger Logger
}

// NewMemoryMessageQueue creates a new in-memory message queue
func NewMemoryMessageQueue(size int, logger Logger) *MemoryMessageQueue {
	if size <= 0 {
		size = 10000 // default size
	}
	return &MemoryMessageQueue{
		queue:  make(chan []byte, size),
		size:   size,
		logger: logger,
	}
}

// Enqueue adds a message to the queue
func (q *MemoryMessageQueue) Enqueue(msg []byte) error {
	if q.queue == nil {
		return fmt.Errorf("queue is closed")
	}

	msgCopy := append([]byte(nil), msg...)
	select {
	case q.queue <- msgCopy:
		return nil
	default:
		q.logger.Warn("queue full, message dropped", "queue_size", q.size, "pending_messages", len(q.queue))
		return fmt.Errorf("queue full")
	}
}

// Dequeue retrieves a message from the queue
func (q *MemoryMessageQueue) Dequeue() ([]byte, error) {
	if q.queue == nil {
		return nil, fmt.Errorf("queue is closed")
	}

	select {
	case msg, ok := <-q.queue:
		if !ok {
			return nil, fmt.Errorf("queue is closed")
		}
		return msg, nil
	default:
		return nil, fmt.Errorf("queue is empty")
	}
}

// IsFull checks if the queue is at capacity
func (q *MemoryMessageQueue) IsFull() bool {
	return len(q.queue) >= q.size
}

// Len returns the current number of messages in the queue
func (q *MemoryMessageQueue) Len() int {
	return len(q.queue)
}

// Close closes the queue
func (q *MemoryMessageQueue) Close() error {
	if q.queue != nil {
		close(q.queue)
		q.queue = nil
	}
	return nil
}
