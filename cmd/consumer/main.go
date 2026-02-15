package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"time"

	"github.com/message-streaming-app/internal/message"
	"github.com/message-streaming-app/internal/protocol"
	"github.com/message-streaming-app/internal/storage"
)

func main() {
	addr := getEnv("BROKER_ADDR", "localhost:9000")

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("dial(%v): %v", addr, err)
	}
	defer conn.Close()

	// Identify as consumer
	if _, err := conn.Write([]byte("CONSUMER\n")); err != nil {
		log.Fatalf("write role: %v", err)
	}

	log.Printf("Connected as consumer to %s", addr)

	// Initialize MongoDB storage
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	mongoURI := getEnv("MONGODB_URI", "mongodb://localhost:27017")
	dbName := getEnv("MONGODB_DATABASE", "message_streaming")
	collectionName := getEnv("MONGO_COLLECTION", "metrics")

	mongoStore, err := storage.NewMongoStore(ctx, mongoURI, dbName, collectionName)
	cancel()
	if err != nil {
		log.Fatalf("initialize MongoDB store: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoStore.Close(ctx); err != nil {
			log.Printf("close MongoDB: %v", err)
		}
	}()

	var buf []byte
	for {
		body, err := protocol.ReadFrame(conn, buf)
		if err != nil {
			log.Printf("read: %v", err)
			return
		}
		buf = body

		var msg message.Message
		if err := json.Unmarshal(body, &msg); err != nil {
			log.Printf("invalid JSON: %v", err)
			continue
		}
		log.Printf("[notification] id=%s type=%s ts=%s payload=%s", msg.ID, msg.Type, msg.Timestamp.Format("15:04:05"), string(msg.Payload))

		// Store message in MongoDB (unmarshal handled by store)
		if err := mongoStore.StoreMessage(msg); err != nil {
			log.Printf("failed to store message in MongoDB: %v", err)
			// Continue processing even if MongoDB store fails
			continue
		}
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
