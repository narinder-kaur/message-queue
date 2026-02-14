package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/message-streaming-app/internal/message"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoStore handles storing messages in MongoDB
type MongoStore struct {
	client     *mongo.Client
	db         *mongo.Database
	collection CollectionAPI
}

// CollectionAPI abstracts required mongo.Collection methods to allow testing.
type CollectionAPI interface {
	InsertOne(context.Context, interface{}, ...*options.InsertOneOptions) (*mongo.InsertOneResult, error)
	Distinct(context.Context, string, interface{}, ...*options.DistinctOptions) ([]interface{}, error)
	CountDocuments(context.Context, interface{}, ...*options.CountOptions) (int64, error)
	Find(context.Context, interface{}, ...*options.FindOptions) (CursorAPI, error)
}

// CursorAPI abstracts the mongo.Cursor methods used in QueryTelemetry.
type CursorAPI interface {
	Close(context.Context) error
	Next(context.Context) bool
	Decode(interface{}) error
	Err() error
}

// realCollection adapts *mongo.Collection to CollectionAPI
type realCollection struct{ c *mongo.Collection }

func (r realCollection) InsertOne(ctx context.Context, doc interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	return r.c.InsertOne(ctx, doc, opts...)
}
func (r realCollection) Distinct(ctx context.Context, field string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error) {
	return r.c.Distinct(ctx, field, filter, opts...)
}
func (r realCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return r.c.CountDocuments(ctx, filter, opts...)
}
func (r realCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CursorAPI, error) {
	cur, err := r.c.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	return realCursor{cur}, nil
}

// realCursor adapts *mongo.Cursor to CursorAPI
type realCursor struct{ c *mongo.Cursor }

func (rc realCursor) Close(ctx context.Context) error        { return rc.c.Close(ctx) }
func (rc realCursor) Next(ctx context.Context) bool           { return rc.c.Next(ctx) }
func (rc realCursor) Decode(v interface{}) error              { return rc.c.Decode(v) }
func (rc realCursor) Err() error                              { return rc.c.Err() }

// NewMongoStore creates a new MongoDB store connection
func NewMongoStore(ctx context.Context, mongoURI, dbName, collectionName string) (*MongoStore, error) {
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	if dbName == "" {
		dbName = "message_streaming"
	}
	if collectionName == "" {
		collectionName = "metrics"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("connect to MongoDB: %w", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping MongoDB: %w", err)
	}

	db := client.Database(dbName)
	collection := db.Collection(collectionName)

	return &MongoStore{
		client:     client,
		db:         db,
		collection: realCollection{collection},
	}, nil
}

// StoreMessage stores the payload from a message in MongoDB with proper timestamp handling
func (ms *MongoStore) StoreMetric(metric map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add timestamp to the payload document
	// Timestamp is stored as a BSON timestamp type
	metric["created_at"] = time.Now().UTC()
	result, err := ms.collection.InsertOne(ctx, metric)
	if err != nil {
		return fmt.Errorf("insert metric: %w", err)
	}

	fmt.Printf("Metric stored with ID: %v\n", result.InsertedID)
	return nil
}

// StoreMessage unmarshals the message payload and stores it as a metric document.
func (ms *MongoStore) StoreMessage(msg message.Message) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	// prefer message timestamp if set
	if !msg.Timestamp.IsZero() {
		payload["timestamp"] = msg.Timestamp
	} else {
		payload["timestamp"] = time.Now().UTC()
	}
	// store
	return ms.StoreMetric(payload)
}

// Close closes the MongoDB connection
func (ms *MongoStore) Close(ctx context.Context) error {
	return ms.client.Disconnect(ctx)
}

// ListGPUs returns distinct GPU identifiers (by UUID) from the metrics collection.
func (ms *MongoStore) ListGPUs(ctx context.Context) ([]string, error) {
	// distinct on `uuid` field; fall back to gpu_id if uuid missing in some documents
	uuids, err := ms.collection.Distinct(ctx, "uuid", bson.M{})
	if err != nil {
		return nil, fmt.Errorf("distinct uuid: %w", err)
	}

	// Convert results to []string
	ids := make([]string, 0, len(uuids))
	for _, v := range uuids {
		if s, ok := v.(string); ok && s != "" {
			ids = append(ids, s)
		}
	}

	// If no uuids found, try distinct on gpu_id
	if len(ids) == 0 {
		gpus, err := ms.collection.Distinct(ctx, "gpu_id", bson.M{})
		if err != nil {
			return nil, fmt.Errorf("distinct gpu_id: %w", err)
		}
		for _, v := range gpus {
			if s, ok := v.(string); ok && s != "" {
				ids = append(ids, s)
			} else if n, ok := v.(int32); ok {
				ids = append(ids, fmt.Sprintf("%d", n))
			}
		}
	}

	return ids, nil
}

// QueryTelemetry returns telemetry documents for a given GPU identifier (uuid or gpu_id).
// start/end are optional; if non-nil they are applied as inclusive time filters on `timestamp`.
func (ms *MongoStore) QueryTelemetry(ctx context.Context, gpuIdentifier string, start, end *time.Time, limit, page int, sortAsc bool) ([]bson.M, int64, error) {
	filter := bson.M{}
	if gpuIdentifier != "" {
		// match either uuid or gpu_id
		filter["$or"] = []bson.M{{"uuid": gpuIdentifier}, {"gpu_id": gpuIdentifier}}
	}
	// time filters operate on `timestamp` field
	if start != nil || end != nil {
		tr := bson.M{}
		if start != nil {
			tr["$gte"] = *start
		}
		if end != nil {
			tr["$lte"] = *end
		}
		filter["timestamp"] = tr
	}

	// Count total
	total, err := ms.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count documents: %w", err)
	}

	findOpts := options.Find()
	if limit <= 0 {
		limit = 100
	}
	findOpts.SetLimit(int64(limit))
	if page <= 1 {
		findOpts.SetSkip(0)
	} else {
		findOpts.SetSkip(int64((page - 1) * limit))
	}
	sortOrder := -1
	if sortAsc {
		sortOrder = 1
	}
	findOpts.SetSort(bson.D{{Key: "timestamp", Value: sortOrder}})

	cur, err := ms.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("find documents: %w", err)
	}
	defer cur.Close(ctx)

	var results []bson.M
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return nil, 0, fmt.Errorf("decode doc: %w", err)
		}
		results = append(results, doc)
	}
	if err := cur.Err(); err != nil {
		return nil, 0, fmt.Errorf("cursor error: %w", err)
	}

	return results, total, nil
}
