package producer

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
)

func TestNewMetricsCSVReaderEmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "empty_*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	// leave file empty
	tmpFile.Close()

	logger := *slog.New(slog.NewTextHandler(os.Stderr, nil))
	_, err = NewMetricsCSVReader(tmpFile.Name(), logger)
	if err == nil {
		t.Fatalf("expected error for empty CSV header, got nil")
	}
}

func TestGetNextRowPartialRecord(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "partial_*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// header has 3 cols; record provides empty value for c
	tmpFile.WriteString("a,b,c\n1,2,\n")
	tmpFile.Close()

	logger := *slog.New(slog.NewTextHandler(os.Stderr, nil))
	reader, err := NewMetricsCSVReader(tmpFile.Name(), logger)
	if err != nil {
		t.Fatalf("NewMetricsCSVReader failed: %v", err)
	}
	defer reader.Close()

	row, err := reader.GetNextRow()
	if err != nil {
		t.Fatalf("GetNextRow failed: %v", err)
	}
	if row["a"] != "1" || row["b"] != "2" {
		t.Fatalf("unexpected values: %v", row)
	}
	if row["c"] != "" {
		t.Fatalf("expected empty string for missing column c, got %q", row["c"])
	}
}

func TestGetRawRowEOFAndHasNextRow(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "eof_*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("h1,h2\n")
	tmpFile.Close()

	logger := *slog.New(slog.NewTextHandler(os.Stderr, nil))
	reader, err := NewMetricsCSVReader(tmpFile.Name(), logger)
	if err != nil {
		t.Fatalf("NewMetricsCSVReader failed: %v", err)
	}
	defer reader.Close()

	_, err = reader.GetRawRow()
	if err == nil {
		t.Fatalf("expected EOF error for GetRawRow on header-only file")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}

	if reader.HasNextRow() {
		t.Fatalf("expected HasNextRow to be false after EOF")
	}
}

func TestCloseWhenFileNil(t *testing.T) {
	r := &MetricsCSVReader{}
	if err := r.Close(); err != nil {
		t.Fatalf("Close returned error for nil file: %v", err)
	}
}

func TestGetNextRowErrorState(t *testing.T) {
	logger := *slog.New(slog.NewTextHandler(os.Stderr, nil))
	r := &MetricsCSVReader{hasError: true, lastError: errors.New("boom"), Logger: logger}
	_, err := r.GetNextRow()
	if err == nil {
		t.Fatalf("expected error when reader in error state")
	}
}
