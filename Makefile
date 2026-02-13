# Message Streaming App - Build targets
BINDIR ?= bin

.PHONY: all build clean message-queue producer consumer metrics

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
