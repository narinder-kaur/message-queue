package broker

import (
	"sync"

	"github.com/google/uuid"
)

// BroadcastRegistry manages consumer channels for broadcast delivery mode
type BroadcastRegistry struct {
	mu        sync.RWMutex
	consumers map[string]chan []byte
	logger    Logger
}

// NewBroadcastRegistry creates a new consumer registry
func NewBroadcastRegistry(logger Logger) *BroadcastRegistry {
	return &BroadcastRegistry{
		consumers: make(map[string]chan []byte),
		logger:    logger,
	}
}

// RegisterConsumer adds a channel for a consumer and returns its unique ID
func (r *BroadcastRegistry) RegisterConsumer(ch chan []byte) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := uuid.New().String()
	r.consumers[id] = ch
	r.logger.Info("consumer registered", "consumer_id", id, "total_consumers", len(r.consumers))
	return id
}

// UnregisterConsumer removes a consumer by ID
func (r *BroadcastRegistry) UnregisterConsumer(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ch, ok := r.consumers[id]; ok {
		close(ch)
		delete(r.consumers, id)
		r.logger.Info("consumer unregistered", "consumer_id", id, "remaining_consumers", len(r.consumers))
	}
}

// BroadcastMessage sends a message to all registered consumers
func (r *BroadcastRegistry) BroadcastMessage(msg []byte) {
	r.mu.RLock()
	chans := make([]chan []byte, 0, len(r.consumers))
	for _, ch := range r.consumers {
		chans = append(chans, ch)
	}
	r.mu.RUnlock()

	for _, ch := range chans {
		msgCopy := append([]byte(nil), msg...)
		select {
		case ch <- msgCopy:
		default:
			r.logger.Warn("consumer channel full, dropping message")
		}
	}
}

// GetConsumerCount returns the number of registered consumers
func (r *BroadcastRegistry) GetConsumerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.consumers)
}

// Close closes all consumer channels and clears the registry
func (r *BroadcastRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, ch := range r.consumers {
		close(ch)
		delete(r.consumers, id)
	}
	return nil
}
