package main

import (
	"log"
	"net"
	"os"
	"strings"

	"github.com/message-streaming-app/internal/broker"
)

func main() {
	addr := getEnv("BROKER_ADDR", ":9000")
	// DELIVERY_MODE: "broadcast" = every consumer gets every message; "queue" = each message to one consumer
	deliveryMode := broker.ParseDeliveryMode(strings.ToLower(getEnv("DELIVERY_MODE", "broadcast")))
	srv := broker.New(deliveryMode)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	log.Printf("Broker listening on %s (delivery=%s)", addr, deliveryMode)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		log.Printf("INFO:accepted connection from %s", conn.RemoteAddr())
		go srv.HandleConn(conn)
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
