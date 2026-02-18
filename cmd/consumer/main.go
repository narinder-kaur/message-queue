package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/message-streaming-app/internal/common"
	"github.com/message-streaming-app/internal/message"
	"github.com/message-streaming-app/internal/protocol"
	"github.com/message-streaming-app/internal/storage"
)

func main() {
	logger := common.GetLogger()
	addr := common.GetEnv("BROKER_ADDR", "localhost:9080")

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		logger.Error("dial(%v): %v", "addr", addr, "error", err)
	}
	defer conn.Close()

	// Identify as consumer
	if _, err := conn.Write([]byte("CONSUMER\n")); err != nil {
		logger.Error(fmt.Sprintf("write role: %v", err.Error()))
		panic("failed to identify as consumer: " + err.Error())
	}

	logger.Info(fmt.Sprintf("Connected as consumer to %s", addr))

	// Initialize MongoDB storage
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	mongoURI := common.GetEnv("MONGODB_URI", "mongodb://localhost:27017")
	dbName := common.GetEnv("MONGODB_DATABASE", "message_streaming")
	collectionName := common.GetEnv("MONGO_COLLECTION", "metrics")

	mongoStore, err := storage.NewMongoStore(ctx, mongoURI, dbName, collectionName)
	cancel()
	if err != nil {
		logger.Error("initialize MongoDB store: %v", "error", err)
		panic("Error initiaizing MongoDB store")
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoStore.Close(ctx); err != nil {
			logger.Error("close MongoDB: %v", "error", err)
		}
	}()

	var buf []byte
	for {
		body, err := protocol.ReadFrame(conn, buf)
		if err != nil {
			logger.Error("read: %v", "error", err)
			return
		}
		buf = body

		var msg message.Message
		if err := json.Unmarshal(body, &msg); err != nil {
			logger.Error("invalid JSON: %v", "error", err)
			continue
		}
		logger.Debug("[notification] id=%s type=%s ts=%s payload=%s", msg.ID, msg.Type, msg.Timestamp.Format("15:04:05"), string(msg.Payload))

		// Store message in MongoDB (unmarshal handled by store)
		if err := mongoStore.StoreMessage(msg); err != nil {
			logger.Error("failed to store message in MongoDB: %v", "error", err)
			// Continue processing even if MongoDB store fails
			continue
		}
	}
}
