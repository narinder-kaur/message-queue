package producer

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// MetricsCSVReader is a pure CSV file reader that provides row-by-row access
type MetricsCSVReader struct {
	file      *os.File
	reader    *csv.Reader
	header    []string
	rowNum    int
	hasError  bool
	lastError error
	Logger    slog.Logger
}

// NewMetricsCSVReader creates a new CSV reader for the specified file
func NewMetricsCSVReader(path string, logger slog.Logger) (*MetricsCSVReader, error) {
	f, err := os.Open(path)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to open CSV file: %v", err))
		return nil, err
	}

	csvReader := csv.NewReader(f)

	// Read header
	header, err := csvReader.Read()
	if err != nil {
		f.Close()
		logger.Error(fmt.Sprintf("failed to read CSV header: %v", err))
		return nil, fmt.Errorf("failed to read CSV header: %v", err)
	}

	// Validate header
	if len(header) == 0 {
		f.Close()
		logger.Error("CSV file has no columns")
		return nil, fmt.Errorf("CSV file has no columns")
	}

	return &MetricsCSVReader{
		file:   f,
		reader: csvReader,
		header: header,
		rowNum: 0,
		Logger: logger,
	}, nil
}

// GetHeader returns the CSV header row
func (r *MetricsCSVReader) GetHeader() []string {
	if r.header == nil {
		return []string{}
	}
	// Return a copy to prevent external modification
	headerCopy := make([]string, len(r.header))
	copy(headerCopy, r.header)
	return headerCopy
}

// HasNextRow checks if there is another row to read
func (r *MetricsCSVReader) HasNextRow() bool {
	if r.hasError {
		return false
	}
	return true
}

// GetNextRow returns the next row as a map with column names as keys
func (r *MetricsCSVReader) GetNextRow() (map[string]string, error) {
	if r.hasError {
		r.Logger.Error(fmt.Sprintf("reader is in error state: %v", r.lastError))
		return nil, fmt.Errorf("reader is in error state: %v", r.lastError)
	}

	record, err := r.reader.Read()
	if err != nil {
		if err == io.EOF {
			r.hasError = true
			return nil, err
		}
		r.hasError = true
		r.lastError = err
		r.Logger.Error(fmt.Sprintf("failed to read CSV row %d: %v", r.rowNum, err))
		return nil, fmt.Errorf("failed to read CSV row %d: %v", r.rowNum, err)
	}

	r.rowNum++

	// Create map from header and record
	row := make(map[string]string, len(r.header))
	for i, col := range r.header {
		if i < len(record) {
			row[col] = record[i]
		} else {
			row[col] = ""
		}
	}

	return row, nil
}

// GetRawRow returns the next row as a slice of strings
func (r *MetricsCSVReader) GetRawRow() ([]string, error) {
	if r.hasError {
		r.Logger.Error(fmt.Sprintf("reader is in error state: %v", r.lastError))
		return nil, fmt.Errorf("reader is in error state: %v", r.lastError)
	}

	record, err := r.reader.Read()
	if err != nil {
		if err == io.EOF {
			r.hasError = true
			return nil, err
		}
		r.hasError = true
		r.lastError = err
		r.Logger.Error(fmt.Sprintf("failed to read CSV row %d: %v", r.rowNum, err))
		return nil, fmt.Errorf("failed to read CSV row %d: %v", r.rowNum, err)
	}

	r.rowNum++
	return record, nil
}

// RowCount returns the number of rows read so far
func (r *MetricsCSVReader) RowCount() int {
	return r.rowNum
}

// IsValid checks if the CSV reader is in a valid state
func (r *MetricsCSVReader) IsValid() bool {
	return !r.hasError && r.file != nil && r.reader != nil
}

// FindColumnIndex finds the index of a column by name (case-insensitive)
func (r *MetricsCSVReader) FindColumnIndex(colName string) int {
	for i, col := range r.header {
		if strings.EqualFold(col, colName) {
			return i
		}
	}
	return -1
}

// Close closes the underlying file
func (r *MetricsCSVReader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

func (r *MetricsCSVReader) GetLogger() slog.Logger {
	return r.Logger
}
