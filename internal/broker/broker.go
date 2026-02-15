package broker

import (
	"bufio"
	"io"
	"net"

	"github.com/message-streaming-app/internal/protocol"
)

const (
	roleProducer = "PRODUCER"
	roleConsumer = "CONSUMER"
)

// Broker accepts producer and consumer connections and distributes messages by mode
type Broker struct {
	mode     DeliveryMode
	logger   Logger
	registry ConsumerRegistry
	queue    MessageQueue
}

// NewBroker creates a new Broker instance
func NewBroker(mode DeliveryMode, logger Logger) *Broker {
	b := &Broker{
		mode:     mode,
		logger:   logger,
		registry: NewBroadcastRegistry(logger),
		queue:    NewMemoryMessageQueue(10000, logger),
	}
	return b
}

// Close shuts down broker internals (registry and queue)
func (b *Broker) Close() error {
	if b.queue != nil {
		_ = b.queue.Close()
	}
	if b.registry != nil {
		_ = b.registry.Close()
	}
	return nil
}

// HandleConn handles a single TCP connection
// The first line sent must be either "PRODUCER" or "CONSUMER"
func (b *Broker) HandleConn(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)

	// Read the role identifier (PRODUCER or CONSUMER)
	line, err := br.ReadString('\n')
	if err != nil {
		b.logger.Error("failed to read role identifier", "error", err)
		return
	}

	role := trimLine(line)
	b.logger.Info("connection received", "remote_addr", conn.RemoteAddr(), "role", role)

	// Create frame reader and writer
	frameReader := NewBufferedFrameReader(br)
	frameWriter := NewConnectionFrameWriter(conn)

	switch role {
	case roleProducer:
		b.handleProducer(frameReader)
	case roleConsumer:
		b.handleConsumer(frameWriter)
	default:
		b.logger.Error("unknown role received", "role", role)
	}
}

// handleProducer reads messages from a producer and enqueues them
func (b *Broker) handleProducer(reader FrameReader) {
	defer b.logger.Info("producer connection closed")

	buf := make([]byte, 0, 64*1024)
	for {
		body, err := reader.ReadFrame(buf)
		if err != nil {
			if err != io.EOF {
				b.logger.Error("producer read error", "error", err)
			}
			return
		}

		// Enqueue the message based on delivery mode
		switch b.mode {
		case Broadcast:
			b.registry.BroadcastMessage(body)
		case Queue:
			if err := b.queue.Enqueue(body); err != nil {
				b.logger.Warn("failed to enqueue message", "error", err)
			}
		}
	}
}

// handleConsumer delivers messages to a consumer
func (b *Broker) handleConsumer(writer FrameWriter) {
	defer b.logger.Info("consumer connection closed")

	switch b.mode {
	case Broadcast:
		b.handleConsumerBroadcast(writer)
	case Queue:
		b.handleConsumerQueue(writer)
	}
}

// handleConsumerBroadcast handles a consumer in broadcast mode
func (b *Broker) handleConsumerBroadcast(writer FrameWriter) {
	// Create a channel for this consumer
	ch := make(chan []byte, 64)

	// Register the consumer
	consumerID := b.registry.RegisterConsumer(ch)
	defer b.registry.UnregisterConsumer(consumerID)

	// Send messages as they arrive
	for msg := range ch {
		if err := writer.WriteFrame(msg); err != nil {
			b.logger.Error("consumer write error", "consumer_id", consumerID, "error", err)
			return
		}
	}
}

// handleConsumerQueue handles a consumer in queue mode
func (b *Broker) handleConsumerQueue(writer FrameWriter) {
	for {
		msg, err := b.queue.Dequeue()
		if err != nil {
			// Queue might be closed or empty, wait a bit and retry
			// In production, this should use a blocking dequeue operation
			b.logger.Error("queue dequeue error", "error", err)
			return
		}

		if err := writer.WriteFrame(msg); err != nil {
			b.logger.Error("consumer write error", "error", err)
			return
		}
	}
}

// trimLine removes trailing line endings from a string
func trimLine(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\r' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}

// BufferedFrameReader wraps a buffered reader to read frames
type BufferedFrameReader struct {
	reader *bufio.Reader
}

// NewBufferedFrameReader creates a new buffered frame reader
func NewBufferedFrameReader(r *bufio.Reader) *BufferedFrameReader {
	return &BufferedFrameReader{reader: r}
}

// ReadFrame reads a frame from the buffered reader
func (f *BufferedFrameReader) ReadFrame(buf []byte) ([]byte, error) {
	return protocol.ReadFrame(f.reader, buf)
}

// ConnectionFrameWriter writes frames to a connection
type ConnectionFrameWriter struct {
	conn net.Conn
}

// NewConnectionFrameWriter creates a new connection frame writer
func NewConnectionFrameWriter(conn net.Conn) *ConnectionFrameWriter {
	return &ConnectionFrameWriter{conn: conn}
}

// WriteFrame writes a frame to the connection
func (f *ConnectionFrameWriter) WriteFrame(data []byte) error {
	return protocol.WriteFrame(f.conn, data)
}
