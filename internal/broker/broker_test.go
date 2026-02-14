package broker

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// simpleConn implements net.Conn for tests
type simpleConn struct {
	readBuf  *bytes.Reader
	writeBuf *bytes.Buffer
	closed   bool
}

func (s *simpleConn) Read(b []byte) (int, error) {
	if s.readBuf == nil {
		return 0, io.EOF
	}
	return s.readBuf.Read(b)
}
func (s *simpleConn) Write(b []byte) (int, error) {
	if s.closed {
		return 0, io.EOF
	}
	return s.writeBuf.Write(b)
}
func (s *simpleConn) Close() error                       { s.closed = true; return nil }
func (s *simpleConn) LocalAddr() net.Addr                { return nil }
func (s *simpleConn) RemoteAddr() net.Addr               { return nil }
func (s *simpleConn) SetDeadline(t time.Time) error      { return nil }
func (s *simpleConn) SetReadDeadline(t time.Time) error  { return nil }
func (s *simpleConn) SetWriteDeadline(t time.Time) error { return nil }

func frameBytes(payload []byte) []byte {
	var h [4]byte
	binary.BigEndian.PutUint32(h[:], uint32(len(payload)))
	return append(h[:], payload...)
}

func TestParseDeliveryModeAndString(t *testing.T) {
	if Broadcast.String() != "broadcast" {
		t.Fatalf("unexpected Broadcast.String()")
	}
	if Queue.String() != "queue" {
		t.Fatalf("unexpected Queue.String()")
	}
	if ParseDeliveryMode("queue") != Queue {
		t.Fatalf("ParseDeliveryMode queue failed")
	}
	if ParseDeliveryMode("unknown") != Broadcast {
		t.Fatalf("ParseDeliveryMode default failed")
	}
}

func TestHandleConnProducerQueueEnqueues(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	b := NewBroker(Queue, logger)

	payload := []byte("hello-queue")
	data := append([]byte("PRODUCER\n"), frameBytes(payload)...)
	conn := &simpleConn{readBuf: bytes.NewReader(data), writeBuf: &bytes.Buffer{}}

	// call HandleConn synchronously
	b.HandleConn(conn)

	// Dequeue message from broker queue
	msg, err := b.queue.Dequeue()
	if err != nil {
		t.Fatalf("expected message in queue, got error: %v", err)
	}
	if !bytes.Equal(msg, payload) {
		t.Fatalf("expected payload %v, got %v", payload, msg)
	}
}

func TestHandleConnConsumerBroadcastReceivesMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	b := NewBroker(Broadcast, logger)

	// consumer conn reads only role
	conn := &simpleConn{readBuf: bytes.NewReader([]byte("CONSUMER\n")), writeBuf: &bytes.Buffer{}}

	var wg sync.WaitGroup
	wg.Go(func() {
		b.HandleConn(conn)
	})

	// wait a bit to ensure consumer registered
	time.Sleep(20 * time.Millisecond)

	// this produces a new message that should be broadcast to the consumer
	payload := []byte("broadcast-msg")
	b.registry.BroadcastMessage(payload)

	// allow write to occur
	time.Sleep(20 * time.Millisecond)

	// close connection to stop handler
	conn.Close()

	// read frame from conn.writeBuf
	wb := conn.writeBuf.Bytes()
	if len(wb) < 4 {
		t.Fatalf("no frame written to consumer")
	}
	n := binary.BigEndian.Uint32(wb[:4])
	body := wb[4 : 4+n]
	if !bytes.Equal(body, payload) {
		t.Fatalf("expected broadcast payload %v, got %v", payload, body)
	}
	assert.True(t, bytes.Equal(body, payload), "expected broadcast payload %v, got %v", payload, body)
}

func TestHandleConnConsumerQueueReceivesEnqueuedMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	b := NewBroker(Queue, logger)

	conn := &simpleConn{readBuf: bytes.NewReader([]byte("CONSUMER\n")), writeBuf: &bytes.Buffer{}}

	var wg sync.WaitGroup
	wg.Go(func() {
		b.HandleConn(conn)
	})

	// enqueue a message
	payload := []byte("queue-consumer-msg")
	if err := b.queue.Enqueue(payload); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	time.Sleep(20 * time.Millisecond)
	conn.Close()
	wg.Wait()

	wb := conn.writeBuf.Bytes()
	if len(wb) < 4 {
		t.Fatalf("expected frame written to queue consumer")
	}
	n := binary.BigEndian.Uint32(wb[:4])
	body := wb[4 : 4+n]
	if !bytes.Equal(body, payload) {
		t.Fatalf("expected payload %v, got %v", payload, body)
	}
}

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

func TestBroadcastRegistryUnregister(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := NewBroadcastRegistry(logger)

	id1 := registry.RegisterConsumer(make(chan []byte, 10))
	id2 := registry.RegisterConsumer(make(chan []byte, 10))

	if registry.GetConsumerCount() != 2 {
		t.Errorf("expected 2 consumers before unregister, got %d", registry.GetConsumerCount())
	}

	registry.UnregisterConsumer(id1)

	if registry.GetConsumerCount() != 1 {
		t.Errorf("expected 1 consumer after unregister, got %d", registry.GetConsumerCount())
	}

	registry.UnregisterConsumer(id2)

	if registry.GetConsumerCount() != 0 {
		t.Errorf("expected 0 consumers after unregistering both, got %d", registry.GetConsumerCount())
	}
}

func TestBroadcastRegistryBroadcastMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := NewBroadcastRegistry(logger)

	ch1 := make(chan []byte, 10)
	ch2 := make(chan []byte, 10)

	registry.RegisterConsumer(ch1)
	registry.RegisterConsumer(ch2)

	msg := []byte("broadcast message")
	registry.BroadcastMessage(msg)

	received1 := <-ch1
	received2 := <-ch2

	if string(received1) != string(msg) {
		t.Errorf("consumer 1: expected %s, got %s", string(msg), string(received1))
	}
	if string(received2) != string(msg) {
		t.Errorf("consumer 2: expected %s, got %s", string(msg), string(received2))
	}
}

func TestBroadcastRegistryGetConsumerCount(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	registry := NewBroadcastRegistry(logger)

	if registry.GetConsumerCount() != 0 {
		t.Errorf("expected 0 consumers initially, got %d", registry.GetConsumerCount())
	}

	registry.RegisterConsumer(make(chan []byte, 10))
	if registry.GetConsumerCount() != 1 {
		t.Errorf("expected 1 consumer after register, got %d", registry.GetConsumerCount())
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

func TestMemoryMessageQueue_IsEmpty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	queue := NewMemoryMessageQueue(10, logger)
	defer queue.Close()

	if queue.Len() != 0 {
		t.Errorf("expected empty queue initially, got %d messages", queue.Len())
	}

	queue.Enqueue([]byte("msg"))
	if queue.Len() != 1 {
		t.Errorf("expected 1 message after enqueue, got %d", queue.Len())
	}
}

func TestMemoryMessageQueue_Close(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	queue := NewMemoryMessageQueue(10, logger)

	queue.Enqueue([]byte("msg"))
	queue.Close()

	if queue.Len() != 0 {
		t.Errorf("expected empty queue after close, got %d messages", queue.Len())
	}
}

func TestMemoryMessageQueue_DequeueEmpty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	queue := NewMemoryMessageQueue(10, logger)
	defer queue.Close()

	_, err := queue.Dequeue()
	if err == nil {
		t.Error("expected error when dequeuing from empty queue")
	}
}

func TestTrimLine(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello\n", "hello"},
		{"hello\r\n", "hello"},
		{"hello", "hello"},
		{"hello\r", "hello"},
		{"\n", ""},
		{"\r\n", ""},
		{"test\n\r", "test"},
	}

	for _, tt := range tests {
		result := trimLine(tt.input)
		if result != tt.expected {
			t.Errorf("trimLine(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestNewBrokerBroadcastMode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	broker := NewBroker(Broadcast, logger)

	if broker == nil {
		t.Error("expected non-nil broker")
	}
	if broker.mode != Broadcast {
		t.Errorf("expected Broadcast mode, got %v", broker.mode)
	}
}

func TestNewBrokerQueueMode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	broker := NewBroker(Queue, logger)

	if broker == nil {
		t.Error("expected non-nil broker")
	}
	if broker.mode != Queue {
		t.Errorf("expected Queue mode, got %v", broker.mode)
	}
}

func TestBufferedFrameReader(t *testing.T) {
	data := []byte{0, 0, 0, 5, 'h', 'e', 'l', 'l', 'o'}
	reader := bufio.NewReader(bytes.NewReader(data))
	frameReader := NewBufferedFrameReader(reader)

	buf := make([]byte, 0, 100)
	frame, err := frameReader.ReadFrame(buf)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}

	if string(frame) != "hello" {
		t.Errorf("expected 'hello', got %s", string(frame))
	}
}

func TestConnectionFrameWriter(t *testing.T) {
	buffer := &bytes.Buffer{}
	writer := NewConnectionFrameWriter(&mockConn{
		write: buffer.Write,
	})

	data := []byte("test message")
	err := writer.WriteFrame(data)
	if err != nil {
		t.Fatalf("WriteFrame failed: %v", err)
	}

	if buffer.Len() == 0 {
		t.Error("expected data to be written")
	}
}

// mockConn implements a mock net.Conn for testing
type mockConn struct {
	write func([]byte) (int, error)
	read  func([]byte) (int, error)
}

func (m *mockConn) Read(b []byte) (int, error) {
	if m.read != nil {
		return m.read(b)
	}
	return 0, nil
}

func (m *mockConn) Write(b []byte) (int, error) {
	if m.write != nil {
		return m.write(b)
	}
	return len(b), nil
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return nil
}

func (m *mockConn) RemoteAddr() net.Addr {
	return nil
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}
