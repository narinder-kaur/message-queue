package producer

import (
	"os"

	"log/slog"
)

// SimpleEnvironmentReader reads environment variables
type SimpleEnvironmentReader struct{}

// NewSimpleEnvironmentReader creates a new environment reader
func NewSimpleEnvironmentReader() *SimpleEnvironmentReader {
	return &SimpleEnvironmentReader{}
}

// Get retrieves an environment variable or returns a default value
func (r *SimpleEnvironmentReader) Get(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// NewLogger creates a new logger wrapped in LoggerI for production use
func NewLogger() (*slog.Logger, error) {
	logger := slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{},
	))
	return logger, nil
}
