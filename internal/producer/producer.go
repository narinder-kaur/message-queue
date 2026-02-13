package producer

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/message-streaming-app/internal/message"
	"github.com/message-streaming-app/internal/protocol"
)

// Producer handles the production of messages to a message broker
type Producer struct {
	conn   net.Conn
	logger *slog.Logger
}

// NewProducer creates a new Producer instance
func NewProducer(conn net.Conn, logger *slog.Logger) *Producer {
	return &Producer{
		conn:   conn,
		logger: logger,
	}
}

// Start initializes the producer by sending the role identifier to the broker
func (p *Producer) Start() error {
	if _, err := p.conn.Write([]byte("PRODUCER\n")); err != nil {
		p.logger.Error(fmt.Sprintf("failed to identify as producer: %v", err))
		return fmt.Errorf("failed to identify as producer: %v", err)
	}
	p.logger.Info("successfully identified as producer")
	return nil
}

// StreamCSVMetrics reads a CSV file and streams metrics to the broker
func (p *Producer) StreamCSVMetrics(csvPath string, tsColumnName, labelsColumnName string) (int, error) {
	// Open CSV reader
	csvReader, err := NewMetricsCSVReader(csvPath, *p.logger)
	if err != nil {
		p.logger.Error(fmt.Sprintf("failed to create CSV reader: %v", err))
		return 0, fmt.Errorf("failed to create CSV reader: %v", err)
	}
	defer csvReader.Close()

	// Create transformer
	transformer := NewRowTransformer(csvReader, tsColumnName, labelsColumnName)

	rowCount := 0
	for csvReader.HasNextRow() {
		row, err := csvReader.GetNextRow()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			p.logger.Warn(fmt.Sprintf("skipping row %d due to read error: %v", rowCount+1, err))
			return rowCount, fmt.Errorf("failed to read row: %v", err)
		}

		// Transform row to payload
		payload := transformer.TransformRow(row)

		// Marshal to JSON
		body, err := json.Marshal(payload)
		if err != nil {
			p.logger.Warn(fmt.Sprintf("failed to marshal payload at row %d: %v", rowCount+1, err))
			continue
		}

		// Create message
		msg := message.New("metric", body, "csv-producer")

		// Write message
		if err := p.Stream(msg); err != nil {
			p.logger.Warn(fmt.Sprintf("failed to write message at row %d: %v", rowCount+1, err))
			return rowCount, fmt.Errorf("failed to write message at row %d: %w", rowCount+1, err)
		}

		rowCount++
	}

	p.logger.Info(fmt.Sprintf("metrics streamed successfully: %d rows", rowCount))
	return rowCount, nil

}

// Close gracefully closes the connection to the broker
func (p *Producer) Close() error {
	if p.conn != nil {
		// Try to flush any remaining data
		_ = p.conn.SetDeadline(time.Now())
		if err := p.conn.Close(); err != nil && err != io.EOF {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}
	return nil
}

func (p *Producer) Stream(msg *message.Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := protocol.WriteFrame(p.conn, body); err != nil {
		return fmt.Errorf("failed to write message to broker: %w", err)
	}

	return nil
}
