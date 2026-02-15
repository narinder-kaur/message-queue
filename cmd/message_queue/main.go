package main

import (
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/message-streaming-app/internal/broker"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Get configuration from environment
	deliveryModeStr := strings.ToLower(getEnv("DELIVERY_MODE", "broadcast"))
	deliveryMode := broker.ParseDeliveryMode(deliveryModeStr)
	tcpAddr := getEnv("TCP_PORT", "9080")
	httpPort := getEnv("HTTP_PORT", "8080")

	// Create broker
	srv := broker.NewBroker(deliveryMode, logger)

	// Start listening
	ln, err := net.Listen("tcp", ":"+tcpAddr)
	if err != nil {
		logger.Error("failed to listen", "addr", tcpAddr, "error", err)
		os.Exit(1)
	}
	logger.Info("broker started successfully", "addr", tcpAddr, "delivery_mode", deliveryMode.String())
	srvHTTP := startHTTPServer(httpPort, logger)

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Accept loop runs in a goroutine so we can signal shutdown
	acceptErrCh := make(chan error, 1)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				// If listener closed, exit accept loop
				if errors.Is(err, net.ErrClosed) {
					acceptErrCh <- nil
					return
				}
				logger.Error("failed to accept connection", "error", err)
				continue
			}
			logger.Info("accepted connection", "remote_addr", conn.RemoteAddr())
			go srv.HandleConn(conn)
		}
	}()

	<-stop
	logger.Info("shutting down")

	// Stop accepting new connections
	_ = ln.Close()

	// Shutdown HTTP server gracefully
	shutdownHTTPServer(srvHTTP, logger)

	// Close broker internals
	_ = srv.Close()

	// wait briefly for ongoing handlers to finish
	time.Sleep(1 * time.Second)

	logger.Info("shutdown complete")
}
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
