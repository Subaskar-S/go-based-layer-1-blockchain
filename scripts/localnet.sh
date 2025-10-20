#!/bin/bash

# Local testnet script for Layer-1 blockchain
# Starts 4 validator nodes locally

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$ROOT_DIR/build"
DATA_DIR="$ROOT_DIR/data"

NUM_NODES=4
BASE_HTTP_PORT=8545
BASE_GRPC_PORT=9090
BASE_P2P_PORT=30303
BASE_METRICS_PORT=6060

echo "Starting Layer-1 Local Testnet"
echo "=============================="

# Build binaries
echo "Building binaries..."
cd "$ROOT_DIR"
make build

# Clean up old data
echo "Cleaning up old data..."
rm -rf "$DATA_DIR/node"*

# Generate validator keys
echo "Generating validator keys..."
mkdir -p "$DATA_DIR/keys"

for i in $(seq 0 $((NUM_NODES-1))); do
    KEY_FILE="$DATA_DIR/keys/validator-$i.json"
    if [ ! -f "$KEY_FILE" ]; then
        "$BUILD_DIR/layer1-cli" keygen --output="$KEY_FILE"
    fi
done

# Start nodes
echo "Starting $NUM_NODES validator nodes..."

BOOTSTRAP_NODE=""

for i in $(seq 0 $((NUM_NODES-1))); do
    NODE_DIR="$DATA_DIR/node-$i"
    KEY_FILE="$DATA_DIR/keys/validator-$i.json"
    HTTP_PORT=$((BASE_HTTP_PORT + i))
    GRPC_PORT=$((BASE_GRPC_PORT + i))
    P2P_PORT=$((BASE_P2P_PORT + i))
    METRICS_PORT=$((BASE_METRICS_PORT + i))
    
    mkdir -p "$NODE_DIR"
    
    echo "Starting node $i..."
    echo "  Data dir: $NODE_DIR"
    echo "  HTTP port: $HTTP_PORT"
    echo "  gRPC port: $GRPC_PORT"
    echo "  P2P port: $P2P_PORT"
    echo "  Metrics port: $METRICS_PORT"
    
    BOOTSTRAP_ARG=""
    if [ -n "$BOOTSTRAP_NODE" ]; then
        BOOTSTRAP_ARG="--bootstrap-nodes=$BOOTSTRAP_NODE"
    fi
    
    "$BUILD_DIR/layer1-node" start \
        --data-dir="$NODE_DIR" \
        --http-port=$HTTP_PORT \
        --grpc-port=$GRPC_PORT \
        --p2p-port=$P2P_PORT \
        --metrics-port=$METRICS_PORT \
        --validator \
        --validator-key="$KEY_FILE" \
        $BOOTSTRAP_ARG \
        > "$NODE_DIR/node.log" 2>&1 &
    
    NODE_PID=$!
    echo $NODE_PID > "$NODE_DIR/node.pid"
    echo "  PID: $NODE_PID"
    
    # Set bootstrap node to first node
    if [ $i -eq 0 ]; then
        # Wait a bit for first node to start
        sleep 2
        # Get the multiaddr from the first node (simplified - would parse from logs)
        BOOTSTRAP_NODE="/ip4/127.0.0.1/tcp/$P2P_PORT"
    fi
    
    sleep 1
done

echo ""
echo "=============================="
echo "Local testnet started!"
echo "=============================="
echo ""
echo "Node endpoints:"
for i in $(seq 0 $((NUM_NODES-1))); do
    HTTP_PORT=$((BASE_HTTP_PORT + i))
    GRPC_PORT=$((BASE_GRPC_PORT + i))
    echo "  Node $i:"
    echo "    HTTP RPC: http://localhost:$HTTP_PORT"
    echo "    gRPC: localhost:$GRPC_PORT"
    echo "    Logs: $DATA_DIR/node-$i/node.log"
done

echo ""
echo "To stop the testnet, run: make localnet-stop"
echo "To view logs: tail -f $DATA_DIR/node-0/node.log"

