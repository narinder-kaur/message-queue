# Message Streaming App

A small message streaming system that demonstrates a simple TCP-based broker, CSV producer, consumers that persist telemetry to MongoDB, and a metrics HTTP service for querying telemetry.

## Services

- `message-queue` — TCP broker that accepts producer and consumer connections and distributes messages in either `broadcast` or `queue` delivery modes. Provides health HTTP endpoints.
- `producer` — Reads a CSV and streams messages to the broker over TCP.
- `consumer` — Connects to broker as a consumer and stores messages in MongoDB.
- `metrics` — HTTP service (Gin) exposing REST endpoints to list GPUs and query telemetry stored in MongoDB. Includes CORS and optional Swagger UI.
- `mongodb` — Persistent storage for telemetry data (external dependency).

## High-level architecture

```mermaid
flowchart LR
  Producer[Producer (CSV)] -->|TCP frames| Broker[Message Broker]
  Broker -->|broadcast/queue| Consumer[Consumer]
  Consumer -->|store| MongoDB[(MongoDB)]
  Metrics[Metrics HTTP Service] -->|query| MongoDB
```

## Project layout

- `cmd/*` — service entry points
- `internal/*` — application internal packages (broker, producer, consumer, metrics, storage, protocol)
- `dockerfiles/*` — Dockerfiles for each service
- `charts/*` — Helm chart for deployment

## Running locally

Prerequisites: Go 1.25+, MongoDB (or use a remote URI).

Build all services:

```sh
make build
```

Start the broker (default ports):

```sh
./bin/message-queue
```

Start a consumer (ensure MongoDB is reachable):

```sh
MONGODB_URI=mongodb://localhost:27017 ./bin/consumer
```

Start metrics service (default port `8080`):

```sh
METRICS_PORT=8080 MONGODB_URI=mongodb://localhost:27017 ./bin/metrics
```

Open Swagger UI for metrics (when metrics service is running):

```
http://localhost:8080/swagger/index.html
```

## Generating OpenAPI docs

This repo includes a `Makefile` target using `swag` to generate OpenAPI documentation.

```sh
make generate-openapi-swag
```

The generated `openapi.yaml` will be created at the repository root and served by the metrics service at `/openapi.yaml`.

## Environment

See `.env.example` for a complete list of environment variables per service.

## Contributing

Contributions are welcome — open issues or PRs.
