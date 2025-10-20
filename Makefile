.PHONY: all build test clean localnet bench proto docker

# Build variables
BINARY_NAME=layer1-node
CLIENT_BINARY=layer1-cli
BUILD_DIR=./build
CMD_NODE=./cmd/node
CMD_CLIENT=./cmd/client

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
GOGET=$(GOCMD) get

all: clean build test

build: build-node build-client
	@echo "Build complete! Binaries in $(BUILD_DIR)/"

build-node:
	@echo "Building node binary..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_NODE)

build-client:
	@echo "Building client binary..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(CLIENT_BINARY) $(CMD_CLIENT)

test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration ./tests/integration/...

bench:
	@echo "Running benchmarks..."
	@./scripts/bench.sh

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf ./data
	rm -f coverage.out

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

proto:
	@echo "Generating protobuf files..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/*.proto

localnet:
	@echo "Starting local testnet..."
	@./scripts/localnet.sh

localnet-stop:
	@echo "Stopping local testnet..."
	@pkill -f $(BINARY_NAME) || true
	@rm -rf ./data/node*

docker:
	@echo "Building Docker image..."
	docker build -t layer1-blockchain:latest .

docker-compose-up:
	@echo "Starting Docker Compose testnet..."
	docker-compose up -d

docker-compose-down:
	@echo "Stopping Docker Compose testnet..."
	docker-compose down -v

fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	$(GOGET) -u google.golang.org/protobuf/cmd/protoc-gen-go
	$(GOGET) -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint

# Contract compilation (requires Rust + wasm32 target)
contracts:
	@echo "Building WASM contracts..."
	@./scripts/build_contracts.sh

# Run single node for development
run-dev:
	@echo "Running development node..."
	@mkdir -p ./data/dev
	$(BUILD_DIR)/$(BINARY_NAME) start --data-dir=./data/dev --http-port=8545 --p2p-port=30303

# Generate test accounts
gen-accounts:
	@echo "Generating test accounts..."
	@$(BUILD_DIR)/$(CLIENT_BINARY) keygen --output=./data/accounts.json --count=10

