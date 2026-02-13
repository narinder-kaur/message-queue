# Message Streaming App - Build targets
BINDIR ?= bin

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
