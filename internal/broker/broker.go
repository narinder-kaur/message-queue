package broker

import (
	"bufio"
	"io"
	"log"
	"net"
	"sync"

	"github.com/message-streaming-app/internal/protocol"
)

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

const (
	roleProducer = "PRODUCER"
	roleConsumer = "CONSUMER"

	queueBufferSize = 10000
)

// Broker accepts producer and consumer connections and distributes JSON messages by mode.
type Broker struct {
	mode DeliveryMode

	// broadcast mode: each consumer has a channel, every message is sent to all
	mu        sync.RWMutex
	consumers map[chan []byte]struct{}

	// queue mode: single channel, each message is read by exactly one consumer
	queue chan []byte
}

// New returns a new Broker with the given delivery mode.
func New(mode DeliveryMode) *Broker {
	b := &Broker{
		mode:      mode,
		consumers: make(map[chan []byte]struct{}),
	}
	if mode == Queue {
		b.queue = make(chan []byte, queueBufferSize)
	}
	return b
}

// RegisterConsumer adds a channel for broadcast mode (receives a copy of every message).
func (b *Broker) RegisterConsumer(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.consumers[ch] = struct{}{}
}

// UnregisterConsumer removes the channel in broadcast mode.
func (b *Broker) UnregisterConsumer(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.consumers, ch)
	close(ch)
}

// broadcast sends a copy of the message to every registered consumer.
func (b *Broker) broadcast(msg []byte) {
	b.mu.RLock()
	chans := make([]chan []byte, 0, len(b.consumers))
	for ch := range b.consumers {
		chans = append(chans, ch)
	}
	b.mu.RUnlock()

	for _, ch := range chans {
		msgCopy := append([]byte(nil), msg...)
		select {
		case ch <- msgCopy:
		default:
		}
	}
}

// enqueue delivers the message according to the broker mode.
func (b *Broker) enqueue(msg []byte) {
	switch b.mode {
	case Broadcast:
		b.broadcast(msg)
	case Queue:
		msgCopy := append([]byte(nil), msg...)
		select {
		case b.queue <- msgCopy:
		default:
			log.Printf("queue full, dropping message")
		}
	}
}

// HandleConn handles a single TCP connection. First line must be "PRODUCER" or "CONSUMER".
func (b *Broker) HandleConn(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)

	line, err := br.ReadString('\n')
	if err != nil {
		log.Printf("read role: %v", err)
		return
	}
	role := trimLine(line)

	switch role {
	case roleProducer:
		b.handleProducer(conn, br)
	case roleConsumer:
		b.handleConsumer(conn, br)
	default:
		log.Printf("unknown role: %q", role)
	}
}

func trimLine(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\r' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}

func (b *Broker) handleProducer(conn net.Conn, r *bufio.Reader) {
	buf := make([]byte, 0, 64*1024)
	for {
		body, err := protocol.ReadFrame(r, buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("producer read: %v", err)
			}
			return
		}
		b.enqueue(body)
	}
}

func (b *Broker) handleConsumer(conn net.Conn, _ *bufio.Reader) {
	switch b.mode {
	case Broadcast:
		b.handleConsumerBroadcast(conn)
	case Queue:
		b.handleConsumerQueue(conn)
	}
}

// handleConsumerBroadcast: each consumer has its own channel, receives every message.
func (b *Broker) handleConsumerBroadcast(conn net.Conn) {
	ch := make(chan []byte, 64)
	b.RegisterConsumer(ch)
	defer b.UnregisterConsumer(ch)

	for msg := range ch {
		if err := protocol.WriteFrame(conn, msg); err != nil {
			log.Printf("consumer write: %v", err)
			return
		}
	}
}

// handleConsumerQueue: all consumers read from the same queue; each message goes to one consumer.
func (b *Broker) handleConsumerQueue(conn net.Conn) {
	for {
		msg, ok := <-b.queue
		if !ok {
			return
		}
		if err := protocol.WriteFrame(conn, msg); err != nil {
			log.Printf("consumer write: %v", err)
			return
		}
	}
}
