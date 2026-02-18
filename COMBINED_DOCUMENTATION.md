# Message Streaming App — Combined Documentation

Version: 1.0
Date: 2026-02-19

---

## Table of Contents

- Overview
- Services
- High-level Architecture (diagram)
- Message Queue Service — Detailed Architecture
- Metrics API — Detailed Documentation
- Environment Variables (per service)
- User Flow
- OpenAPI / Swagger
- How to run

---

## Overview

Message Streaming App is a lightweight message streaming example consisting of:

- a TCP `message-queue` broker that accepts producers and consumers,
- `producer` that streams CSV metrics into the broker,
- `consumer` that receives messages and persists telemetry into MongoDB,
- `metrics` HTTP service (Gin) to query telemetry and expose API docs.

The system emphasizes clarity: simple framing protocol for reliable message boundaries, flexible delivery modes (broadcast or queue), and a REST API for querying persisted telemetry.

## Services

- `message-queue` — TCP broker (entrypoint: `cmd/message_queue`).
- `producer` — CSV producer (entrypoint: `cmd/producer`).
- `consumer` — Consumer that writes to MongoDB (entrypoint: `cmd/consumer`).
- `metrics` — Gin-based HTTP metrics API (entrypoint: `cmd/metrics`).
- `mongodb` — External datastore for telemetry (not included in repo).

## High-level Architecture

```mermaid
flowchart LR
  Producer[Producer (CSV)] -->|TCP frames| Broker[Message Broker]
  Broker -->|broadcast/queue| Consumer[Consumer]
  Consumer -->|store| MongoDB[(MongoDB)]
  Metrics[Metrics HTTP Service] -->|query| MongoDB
```

---

## Message Queue Service — Detailed Architecture

Location: `internal/broker`

### Purpose

The broker accepts TCP connections from producers and consumers. The first line sent by a client must be either `PRODUCER` or `CONSUMER` (followed by `\n`). Producers stream framed messages; consumers receive messages according to broker delivery mode.

### Key Components

- `Broker` (`broker.go`): accepts connections, reads client role, delegates to producer/consumer handlers.
- `BroadcastRegistry` (`consumer_registry.go`): manages a map of consumers (channel per consumer) and broadcasts messages to all registered consumers.
- `MemoryMessageQueue` (`memory_queue.go`): buffered in-memory FIFO queue used in `queue` delivery mode.
- `protocol` package (`internal/protocol`): implements length-prefixed framing (reader/writer helpers) to ensure message boundaries.
- `FrameReader` / `FrameWriter`: adapters for reading/writing frames over network connections.

### Delivery Modes

- `broadcast`: Each registered consumer receives every message. Consumer channels have a buffer (`CONSUMER_CHANNEL_BUFFER_SIZE`). If a consumer channel is full, messages may be dropped with a warning.
- `queue`: Messages are enqueued in a buffered in-memory queue; consumers dequeue messages. When the queue is full, `Enqueue` fails and the message is dropped.

### Configuration (env vars)

- `DELIVERY_MODE` — `broadcast` or `queue` (default: `broadcast`).
- `TCP_PORT` — TCP listener port (default: `9080`).
- `HTTP_PORT` — HTTP port for health checks (default: `8080`).
- `MAX_CONSUMERS` — size hint for registry (default: `10`).
- `CONSUMER_CHANNEL_BUFFER_SIZE` — per-consumer buffer size (default: `10000`).

### Reliability & Scaling Notes

- Current broker is in-memory and single-node. For high reliability, use a durable message broker (Kafka/RabbitMQ) or add persistence (disk-backed queue).
- To scale horizontally, run multiple broker instances behind a load balancer or migrate to a distributed message system.

---

## Metrics API — Detailed Documentation

Base path: `/api/v1`

### Authentication

- Optional bearer token via `Authorization: Bearer <token>`. The token value is configured with `AUTH_TOKEN` environment variable in the `metrics` service. If `AUTH_TOKEN` is empty, auth is disabled.

### Endpoints

1) GET `/api/v1/gpus`

- Summary: Returns a paginated list of GPU IDs.
- Query params:
  - `limit` (int) — default `DEFAULT_PAGE_SIZE`.
  - `page` (int) — default `1`.
  - `sort` (string) — `asc` or `desc`.
- Response 200: `ListResponse` — JSON with `total`, `page`, `limit`, `items` (array of GPU IDs).

2) GET `/api/v1/gpus/{id}/telemetry`

- Summary: Returns paginated telemetry records for GPU `id`.
- Path param: `id` (string).
- Query params:
  - `start_time` (RFC3339 string, optional)
  - `end_time` (RFC3339 string, optional)
  - `limit`, `page`, `sort` same as above.
- Response 200: `QueryResponse` — JSON with `total`, `page`, `limit`, `items` (array of telemetry objects).

### Health endpoints

- GET `/healthz` — 200 OK (liveness).
- GET `/ready` — 200 OK (readiness).

### Examples

List GPUs:

```sh
curl -H "Authorization: Bearer ${AUTH_TOKEN}" "http://localhost:8080/api/v1/gpus?limit=20&page=1"
```

Query telemetry:

```sh
curl -H "Authorization: Bearer ${AUTH_TOKEN}" "http://localhost:8080/api/v1/gpus/gpu-1/telemetry?start_time=2025-07-18T13:00:00Z&end_time=2025-07-18T14:00:00Z"
```

### Schema summary

- `ListResponse`:
  - `total` (int)
  - `page` (int)
  - `limit` (int)
  - `items` ([]string)

- `QueryResponse`:
  - `total` (int)
  - `page` (int)
  - `limit` (int)
  - `items` ([]object)

---

## Environment Variables (per service)

Content is taken from `.env.example`.

### message-queue

- `DELIVERY_MODE` (broadcast|queue) — default `broadcast`
- `TCP_PORT` — default `9080`
- `HTTP_PORT` — default `8080`
- `MAX_CONSUMERS` — default `10`
- `CONSUMER_CHANNEL_BUFFER_SIZE` — default `10000`

### producer

- `BROKER_ADDR` — `host:port` (default `localhost:9080`)
- `CSV_PATH` — path to CSV file

### consumer

- `BROKER_ADDR` — broker address
- `MONGODB_URI` — MongoDB connection string
- `MONGODB_DATABASE` — DB name (default `message_streaming`)
- `MONGO_COLLECTION` — collection name (default `metrics`)

### metrics

- `METRICS_PORT` — HTTP port (default `8080`)
- `MONGODB_URI`, `MONGODB_DATABASE`, `MONGO_COLLECTION` — MongoDB settings
- `AUTH_TOKEN` — optional bearer token; empty disables auth
- `DEFAULT_PAGE_SIZE` — default pagination limit (default `100`)

---

## User Flow

1. Producer reads CSV and connects to broker (`BROKER_ADDR`) and identifies as `PRODUCER`.
2. Producer streams framed messages (length-prefixed) over TCP.
3. Broker receives frames and, depending on `DELIVERY_MODE`:
   - `broadcast`: forwards message to all registered consumers via per-consumer channel.
   - `queue`: enqueues message in a FIFO queue for consumers to dequeue.
4. Consumer(s) connect to broker as `CONSUMER` and receive messages; the consumer persists messages into MongoDB.
5. Metrics service queries MongoDB to return lists of GPUs and telemetry via REST endpoints.

---

## OpenAPI / Swagger

- The repo includes a Makefile target `generate-openapi-swag` which runs `swag` to generate API docs. The metrics service serves the generated `openapi.yaml` at `/openapi.yaml` and exposes Swagger UI at `/swagger/index.html`.

Regenerate OpenAPI with:

```sh
make generate-openapi-swag
```

Swagger UI (when running metrics on default port 8080):

```
http://localhost:8080/swagger/index.html
```

---

## How to run (quick)

```sh
# build binaries
make build

# run broker
./bin/message-queue

# run metrics (example)
METRICS_PORT=8080 MONGODB_URI=mongodb://localhost:27017 ./bin/metrics

# run producer
BROKER_ADDR=localhost:9080 CSV_PATH=internal/data/dcgm_metrics_20250718_134233.csv ./bin/producer

# run consumer (stores to MongoDB)
MONGODB_URI=mongodb://localhost:27017 ./bin/consumer
```

---

If you want this combined doc exported to a different format or split into a single HTML page, I can add a script or Makefile target to generate that. 
