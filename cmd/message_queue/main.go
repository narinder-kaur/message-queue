package main

import (
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/message-streaming-app/internal/broker"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Get configuration from environment
	addr := getEnv("BROKER_ADDR", ":9000")
	deliveryModeStr := strings.ToLower(getEnv("DELIVERY_MODE", "broadcast"))
	deliveryMode := broker.ParseDeliveryMode(deliveryModeStr)

	// Create broker
	srv := broker.NewBroker(deliveryMode, logger)

	// Start listening
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", "addr", addr, "error", err)
		os.Exit(1)
	}
	defer ln.Close()

	logger.Info("broker started", "addr", addr, "delivery_mode", deliveryMode.String())

	// Accept connections
	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.Error("failed to accept connection", "error", err)
			continue
		}
		logger.Info("accepted connection", "remote_addr", conn.RemoteAddr())
		go srv.HandleConn(conn)
	}
}
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}