package producer

import (
	"sync"

	"github.com/message-streaming-app/internal/message"
)

// MockLogger is a logger implementation for testing
type MockLogger struct {
	Logs   []string
	Fatals []string
	mu     sync.Mutex
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		Logs:   []string{},
		Fatals: []string{},
	}
}

func (l *MockLogger) Logf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Logs = append(l.Logs, format)
}

func (l *MockLogger) Fatalf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Fatals = append(l.Fatals, format)
}

// MockMessageWriter is a message writer implementation for testing
type MockMessageWriter struct {
	Messages    []*message.Message
	mu          sync.Mutex
	shouldError bool
	errorMsg    string
}

func NewMockMessageWriter() *MockMessageWriter {
	return &MockMessageWriter{
		Messages: []*message.Message{},
	}
}

func (m *MockMessageWriter) Write(msg *message.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.shouldError {
		return NewError(m.errorMsg)
	}
	m.Messages = append(m.Messages, msg)
	return nil
}

func (m *MockMessageWriter) SetError(enabled bool, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = enabled
	m.errorMsg = msg
}

// SimpleError is a simple error implementation
type SimpleError struct {
	msg string
}

func NewError(msg string) error {
	return &SimpleError{msg: msg}
}

func (e *SimpleError) Error() string {
	return e.msg
}
