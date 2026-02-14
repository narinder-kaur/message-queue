package producer

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/message-streaming-app/internal/message"
)

// mockNetConn is a simple net.Conn mock for tests
type mockNetConn struct {
	writeBuffer *bytes.Buffer
	writeErr    error
	closeErr    error
}

func (m *mockNetConn) Read(b []byte) (int, error) { return 0, io.EOF }
func (m *mockNetConn) Write(b []byte) (int, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.writeBuffer.Write(b)
}
func (m *mockNetConn) Close() error                       { return m.closeErr }
func (m *mockNetConn) LocalAddr() net.Addr                { return nil }
func (m *mockNetConn) RemoteAddr() net.Addr               { return nil }
func (m *mockNetConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockNetConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockNetConn) SetWriteDeadline(t time.Time) error { return nil }

func TestProducerStartSuccessAndFailure(t *testing.T) {
	buf := &bytes.Buffer{}
	mc := &mockNetConn{writeBuffer: buf}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	p := NewProducer(mc, logger)

	// success
	if err := p.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if buf.String() != "PRODUCER\n" {
		t.Fatalf("expected PRODUCER\n, got %q", buf.String())
	}

	// failure on write
	mc2 := &mockNetConn{writeBuffer: &bytes.Buffer{}, writeErr: errors.New("write fail")}
	p2 := NewProducer(mc2, logger)
	if err := p2.Start(); err == nil {
		t.Fatalf("expected Start to fail on write error")
	}
}

func TestProducerStreamAndClose(t *testing.T) {
	buf := &bytes.Buffer{}
	mc := &mockNetConn{writeBuffer: buf}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	p := NewProducer(mc, logger)

	msg := message.New("test", []byte(`"hello"`), "tester")
	if err := p.Stream(msg); err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	// protocol.WriteFrame writes 4-byte length prefix + body
	out := buf.Bytes()
	if len(out) < 4 {
		t.Fatalf("expected frame bytes, got %d", len(out))
	}

	// Close success
	mc.closeErr = nil
	if err := p.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Close returns error when underlying Close returns error
	mc2 := &mockNetConn{writeBuffer: &bytes.Buffer{}, closeErr: errors.New("close boom")}
	p2 := NewProducer(mc2, logger)
	if err := p2.Close(); err == nil {
		t.Fatalf("expected Close to return error when conn.Close fails")
	}
}

func TestStreamCSVMetricsSuccessAndCSVNotFound(t *testing.T) {
	// prepare temp CSV
	tmpFile, err := os.CreateTemp("", "prod_*.csv")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("col1,col2\nval1,val2\nval3,val4\n")
	tmpFile.Close()

	buf := &bytes.Buffer{}
	mc := &mockNetConn{writeBuffer: buf}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	p := NewProducer(mc, logger)

	n, err := p.StreamCSVMetrics(tmpFile.Name(), "", "")
	if err != nil {
		t.Fatalf("StreamCSVMetrics failed: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 rows streamed, got %d", n)
	}

	// Non-existent file should return error
	_, err = p.StreamCSVMetrics("/no/such/file.csv", "", "")
	if err == nil {
		t.Fatalf("expected error for missing CSV file")
	}
}

func TestParseLabelsAndTransformRowWithRaw(t *testing.T) {
	raw := `k1="v1",k2="v,2",k3="v\\"3"`
	labels := ParseLabels(raw)
	if labels["k1"] != "v1" {
		t.Fatalf("k1 mismatch: %v", labels)
	}
	if labels["k2"] != "v,2" {
		t.Fatalf("k2 mismatch: %v", labels)
	}
	if labels["k3"] == "" || labels["k3"][len(labels["k3"])-1] != '3' {
		t.Fatalf("k3 mismatch: %v", labels)
	}

	header := []string{"ts", "labels", "v"}
	record := []string{"oldts", `a="1",b="2"`, "42"}
	payload := TransformRowWithRaw(header, record, 0, 1)
	if _, ok := payload["ts"]; !ok {
		t.Fatalf("expected ts replaced")
	}
	if _, ok := payload["labels"]; !ok {
		t.Fatalf("expected labels parsed")
	}
	if payload["v"] != "42" {
		t.Fatalf("expected v=42, got %v", payload["v"])
	}
}

func TestNewMetricsCSVReader(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("col1,col2\nval1,val2\n")
	tmpFile.Close()

	logger := *slog.New(slog.NewTextHandler(os.Stderr, nil))
	reader, err := NewMetricsCSVReader(tmpFile.Name(), logger)
	if err != nil {
		t.Fatalf("NewMetricsCSVReader failed: %v", err)
	}
	defer reader.Close()

	if reader == nil {
		t.Error("expected non-nil reader")
	}
}

func TestNewMetricsCSVReaderFileNotFound(t *testing.T) {
	logger := *slog.New(slog.NewTextHandler(os.Stderr, nil))
	_, err := NewMetricsCSVReader("/nonexistent/path/file.csv", logger)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestCSVReaderGetHeader(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("timestamp,gpu_id,temperature\nval1,val2,val3\n")
	tmpFile.Close()

	logger := *slog.New(slog.NewTextHandler(os.Stderr, nil))
	reader, err := NewMetricsCSVReader(tmpFile.Name(), logger)
	if err != nil {
		t.Fatalf("NewMetricsCSVReader failed: %v", err)
	}
	defer reader.Close()

	header := reader.GetHeader()
	if len(header) != 3 {
		t.Errorf("expected 3 columns, got %d", len(header))
	}
	if header[0] != "timestamp" {
		t.Errorf("expected 'timestamp', got '%s'", header[0])
	}
}

func TestCSVReaderHasNextRow(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("a,b\n1,2\n3,4\n")
	tmpFile.Close()

	logger := *slog.New(slog.NewTextHandler(os.Stderr, nil))
	reader, err := NewMetricsCSVReader(tmpFile.Name(), logger)
	if err != nil {
		t.Fatalf("NewMetricsCSVReader failed: %v", err)
	}
	defer reader.Close()

	count := 0
	for reader.HasNextRow() {
		reader.GetNextRow()
		count++
	}

	if count != 3 {
		t.Errorf("expected 3 rows, got %d", count)
	}
}

func TestCSVReaderRowCount(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("h1,h2\nv1,v2\nv3,v4\nv5,v6\n")
	tmpFile.Close()

	logger := *slog.New(slog.NewTextHandler(os.Stderr, nil))
	reader, err := NewMetricsCSVReader(tmpFile.Name(), logger)
	if err != nil {
		t.Fatalf("NewMetricsCSVReader failed: %v", err)
	}
	defer reader.Close()

	for reader.HasNextRow() {
		reader.GetNextRow()
	}

	if reader.RowCount() != 3 {
		t.Errorf("expected 3 rows, got %d", reader.RowCount())
	}
}

func TestCSVReaderIsValid(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("col1,col2\nval1,val2\n")
	tmpFile.Close()

	logger := *slog.New(slog.NewTextHandler(os.Stderr, nil))
	reader, err := NewMetricsCSVReader(tmpFile.Name(), logger)
	if err != nil {
		t.Fatalf("NewMetricsCSVReader failed: %v", err)
	}
	defer reader.Close()

	if !reader.IsValid() {
		t.Error("expected reader to be valid")
	}
}

func TestCSVReaderFindColumnIndex(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("name,age,city\nAlice,30,NYC\n")
	tmpFile.Close()

	logger := *slog.New(slog.NewTextHandler(os.Stderr, nil))
	reader, err := NewMetricsCSVReader(tmpFile.Name(), logger)
	if err != nil {
		t.Fatalf("NewMetricsCSVReader failed: %v", err)
	}
	defer reader.Close()

	idx := reader.FindColumnIndex("age")
	if idx != 1 {
		t.Errorf("expected index 1, got %d", idx)
	}

	idx = reader.FindColumnIndex("missing")
	if idx != -1 {
		t.Errorf("expected index -1 for missing column, got %d", idx)
	}
}

func TestCSVReaderGetRawRow(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("name,value\ntest,123\n")
	tmpFile.Close()

	logger := *slog.New(slog.NewTextHandler(os.Stderr, nil))
	reader, err := NewMetricsCSVReader(tmpFile.Name(), logger)
	if err != nil {
		t.Fatalf("NewMetricsCSVReader failed: %v", err)
	}
	defer reader.Close()

	if reader.HasNextRow() {
		row, err := reader.GetRawRow()
		if err != nil {
			t.Fatalf("GetRawRow failed: %v", err)
		}
		if len(row) != 2 {
			t.Errorf("expected 2 columns, got %d", len(row))
		}
		if row[0] != "test" || row[1] != "123" {
			t.Errorf("unexpected row values: %v", row)
		}
	}
}
