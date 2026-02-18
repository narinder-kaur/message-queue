# Message Streaming App - Build targets
BINDIR ?= bin
# compute short git hash for tagging images; allow override via TAG env
TAG := $(shell git rev-parse --short HEAD 2>/dev/null || echo latest)

# per-service image names
CONSUMER_IMAGE ?= $(REGISTRY)/consumer
MESSAGE_QUEUE_IMAGE ?= $(REGISTRY)/message-queue
METRICS_IMAGE ?= $(REGISTRY)/metrics
PRODUCER_IMAGE ?= $(REGISTRY)/producer

.PHONY: docker-build-all docker-push-all docker-build docker-push check_login. check_docker_registry check_docker_credentials

.DEFAULT_GOAL := all

.PHONY: all build clean message-queue producer consumer metrics build-spec test coverage coverage-check

all: build

build: message-queue producer consumer metrics

message-queue:
	@mkdir -p $(BINDIR)
	go build -o $(BINDIR)/message-queue ./cmd/message_queue

producer:
	@mkdir -p $(BINDIR)
	go build -o $(BINDIR)/producer ./cmd/producer

consumer:
	@mkdir -p $(BINDIR)
	go build -o $(BINDIR)/consumer ./cmd/consumer

metrics:
	@mkdir -p $(BINDIR)
	go build -o $(BINDIR)/metrics ./cmd/metrics

clean:
	rm -rf $(BINDIR)

# Docker image helpers

docker-build-consumer: check_docker_registry
	docker build -f dockerfiles/consumer/Dockerfile -t $(CONSUMER_IMAGE):$(TAG) .

docker-build-message-queue: check_docker_registry
	docker build -f dockerfiles/message_queue/Dockerfile -t $(MESSAGE_QUEUE_IMAGE):$(TAG) .

docker-build-metrics: check_docker_registry
	docker build -f dockerfiles/metrics/Dockerfile -t $(METRICS_IMAGE):$(TAG) .

docker-build-producer: check_docker_registry
	docker build -f dockerfiles/producer/Dockerfile -t $(PRODUCER_IMAGE):$(TAG) .

docker-build-all: docker-build-consumer docker-build-message-queue docker-build-metrics docker-build-producer


docker-push-consumer: check_login
	@echo "--- Pushing consumer image to registry ---"
	docker push $(CONSUMER_IMAGE):$(TAG)

docker-push-message-queue: check_login
	@echo "--- Pushing message queue image to registry ---"
	docker push $(MESSAGE_QUEUE_IMAGE):$(TAG)

docker-push-metrics: check_login
	@echo "--- Pushing metrics image to registry ---"
	docker push $(METRICS_IMAGE):$(TAG)

docker-push-producer: check_login
	@echo "--- Pushing producer image to registry ---"
	docker push $(PRODUCER_IMAGE):$(TAG)

docker-push-all: docker-push-consumer docker-push-message-queue docker-push-metrics docker-push-producer

check_docker_registry:
	@echo "--- Checking Docker registry environment variable ---"
	@if [ -z "$(REGISTRY)" ]; then \
		echo "Error: REGISTRY environment variable is not set. Please set it to your Docker registry URL (e.g., docker.io/yourusername)"; \
		exit 1; \
	fi

check_docker_credentials:
	@echo "--- Checking Docker environment variables---"
	@if [ -z "$(DOCKER_USERNAME)" ]; then \
		echo "Error: DOCKER_USERNAME environment variable is not set. Please set it to your Docker registry username"; \
		exit 1; \
	fi
	@if [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "Error: DOCKER_PASSWORD environment variable is not set. Please set it to your Docker registry password"; \
		exit 1; \
	fi

check_login:
	@echo "--- Ensuring Docker login ---"
	@docker login $(REGISTRY) --username $(DOCKER_USERNAME) --password-stdin <<< "$(DOCKER_PASSWORD)"

# Build OpenAPI spec using apispec
.PHONY: build-spec
build-spec:
	# install apispec (latest) and generate openapi.yaml
	@echo "Installing apispec..."
	@go install github.com/ehabterra/apispec/cmd/apispec@latest
	@echo "Generating openapi.yaml..."
	@apispec -o openapi.yaml -O 3.0.1

# Test targets
test:
	@echo "Running tests for all services..."
	go test ./internal/... -v

coverage:
	@echo "Running tests with coverage analysis..."
	@mkdir -p coverage
	go test ./internal/... -coverprofile=coverage/coverage.out
	@echo ""
	@echo "Coverage Summary:"
	@go tool cover -func=coverage/coverage.out | tail -1
	@echo ""
	@echo "Generating coverage HTML report..."
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated: coverage/coverage.html"

coverage-check: coverage
	@echo ""
	@echo "Checking coverage threshold (70%)..."
	@coverage_percent=$$(go tool cover -func=coverage/coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	if (( $$(echo "$$coverage_percent < 70" | bc -l) )); then \
		echo "❌ Coverage check FAILED: $$coverage_percent% < 70%"; \
		exit 1; \
	else \
		echo "✓ Coverage check PASSED: $$coverage_percent% >= 70%"; \
	fi

# command for installing tools (e.g. apispec) - can be extended for other tools as needed
.PHONY: install-tools
install-tools:
	@echo "Installing python"
	@which python3 > /dev/null || (echo "Python3 is not installed. Please install Python3 to proceed." && exit 1)
	@echo "Installing apispec..."
	@go install github.com/ehabterra/apispec/cmd/apispec@latest
	@echo "Installing yaml-merge-cli..."
	@go install github.com/ericwenn/yaml-merge-cli

.PHONY: generate-openapi
generate-openapi: install-tools
	@echo "Generating OpenAPI specification..."
	@apispec -o tempout.yaml
	@go get github.com/ericwenn/yaml-merge-cli
	yaml-merge-cli tempout.yaml apispec.yaml > openapi.yaml
	rm tempout.yaml
# target to run yaml linting using kubeval
.PHONY: lint-yaml
lint-yaml:
	@echo "Linting YAML files with yamlLint..."
	@docker run -it --rm -v "$(CURDIR):/code" pipelinecomponents/yamllint yamllint .

.PHONY: deploy
deploy:
	@echo "Deploying application using Helm..."
	helm upgrade --install message-streaming-app charts/message-streaming-app -f charts/message-streaming-app/values.yaml --set consumer.image.tag=$(TAG) \
		--set messageQueue.image.tag=$(TAG) \
		--set metrics.image.tag=$(TAG) \
		--set producer.image.tag=$(TAG)

.PHONY: create-cluster
create-cluster:
	@echo "Creating Kubernetes cluster with kind..."
	export HOSTPATH="$(CURDIR)/internal/data" && envsubst < ./kind/config.yaml | kind create cluster --name kind-cluster --config -
