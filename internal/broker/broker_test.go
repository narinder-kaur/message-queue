package broker

import (
	"fmt"
	"log/slog"
	"os"
	"testing"
)

func TestBroadcastRegistryRegister(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := NewBroadcastRegistry(logger)

	id1 := registry.RegisterConsumer(make(chan []byte, 10))
	id2 := registry.RegisterConsumer(make(chan []byte, 10))

	if registry.GetConsumerCount() != 2 {
		t.Errorf("expected 2 consumers, got %d", registry.GetConsumerCount())
	}

	if id1 == id2 {
		t.Error("consumer IDs should be unique")
	}
}

func TestMemoryMessageQueue_Enqueue_Dequeue(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	queue := NewMemoryMessageQueue(10, logger)
	defer queue.Close()

	msg := []byte("test message")
	if err := queue.Enqueue(msg); err != nil {
		t.Errorf("failed to enqueue: %v", err)
	}

	if queue.Len() != 1 {
		t.Errorf("expected queue length 1, got %d", queue.Len())
	}

	received, err := queue.Dequeue()
	if err != nil {
		t.Errorf("failed to dequeue: %v", err)
	}

	if string(received) != string(msg) {
		t.Errorf("received wrong message: expected %s, got %s", string(msg), string(received))
	}
}

func TestMemoryMessageQueue_Full(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	queue := NewMemoryMessageQueue(2, logger)
	defer queue.Close()

	queue.Enqueue([]byte("msg1"))
	queue.Enqueue([]byte("msg2"))

	if !queue.IsFull() {
		t.Error("expected queue to be full")
	}

	err := queue.Enqueue([]byte("msg3"))
	if err == nil {
		t.Error("expected error when enqueueing to full queue")
	}
}

func TestMemoryMessageQueue_ManyMessages(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	queue := NewMemoryMessageQueue(100, logger)
	defer queue.Close()

	numMessages := 50
	for i := 0; i < numMessages; i++ {
		msg := []byte(fmt.Sprintf("message %d", i))
		if err := queue.Enqueue(msg); err != nil {
			t.Errorf("failed to enqueue message %d: %v", i, err)
		}
	}

	if queue.Len() != numMessages {
		t.Errorf("expected %d messages, got %d", numMessages, queue.Len())
	}

	for i := 0; i < numMessages; i++ {
		_, err := queue.Dequeue()
		if err != nil {
			t.Errorf("failed to dequeue message %d: %v", i, err)
		}
	}

	if queue.Len() != 0 {
		t.Errorf("expected empty queue, got %d messages", queue.Len())
	}
}
