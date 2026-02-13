package main

import (
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"github.com/message-streaming-app/internal/producer"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Initialize environment reader
	envReader := producer.NewSimpleEnvironmentReader()

	// Get configuration from environment
	brokerAddr := envReader.Get("BROKER_ADDR", "localhost:9000")
	csvPath := envReader.Get("CSV_PATH", filepath.Join("../../", "internal", "data", "dcgm_metrics_20250718_134233.csv"))

	// Establish connection to broker
	conn, err := net.Dial("tcp", brokerAddr)
	if err != nil {
		logger.Error("failed to connect to broker", "addr", brokerAddr, "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Resolve CSV path
	absCSVPath, err := filepath.Abs(csvPath)
	if err != nil {
		logger.Error("failed to resolve CSV path", "error", err)
		os.Exit(1)
	}

	// Create producer
	prod := producer.NewProducer(conn, logger)

	// Start producer
	if err := prod.Start(); err != nil {
		logger.Error("failed to start producer", "error", err)
		os.Exit(1)
	}

	// Stream CSV metrics
	// Specify column names for transformation: "timestamp" and "labels_raw"
	rowCount, err := prod.StreamCSVMetrics(absCSVPath, "timestamp", "labels_raw")
	if err != nil {
		logger.Error("failed to stream CSV metrics", "error", err)
		os.Exit(1)
	}

	// Graceful shutdown
	if err := prod.Close(); err != nil {
		logger.Warn("error during shutdown", "error", err)
	}

	logger.Info("Producer shutdown complete", "rows_processed", rowCount)
}
