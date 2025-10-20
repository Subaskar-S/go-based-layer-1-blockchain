# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN make build-node build-client

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/build/layer1-node /usr/local/bin/
COPY --from=builder /build/build/layer1-cli /usr/local/bin/

# Copy scripts
COPY scripts/ /app/scripts/

# Create data directory
RUN mkdir -p /app/data

# Expose ports
# 8545: HTTP RPC
# 9090: gRPC
# 30303: P2P
# 6060: Metrics
EXPOSE 8545 9090 30303 6060

ENTRYPOINT ["layer1-node"]
CMD ["start", "--data-dir=/app/data"]

