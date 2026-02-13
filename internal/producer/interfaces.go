package producer

import (
	"log/slog"
)

// EnvironmentReader defines the interface for reading environment variables
type EnvironmentReader interface {
	Get(key, defaultVal string) string
}

// CSVReader defines the interface for reading and processing CSV files
type CSVReader interface {
	GetHeader() []string
	HasNextRow() bool
	GetNextRow() (map[string]string, error)
	GetRawRow() ([]string, error)
	RowCount() int
	IsValid() bool
	FindColumnIndex(colName string) int
	Close() error
	GetLogger() slog.Logger
}
