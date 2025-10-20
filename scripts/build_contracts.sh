#!/bin/bash

# Build WASM smart contracts

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
CONTRACTS_DIR="$ROOT_DIR/examples/contracts"

echo "Building WASM Smart Contracts"
echo "=============================="

# Check if Rust is installed
if ! command -v rustc &> /dev/null; then
    echo "Error: Rust is not installed"
    echo "Install Rust from: https://rustup.rs/"
    exit 1
fi

# Check if wasm32 target is installed
if ! rustup target list | grep -q "wasm32-unknown-unknown (installed)"; then
    echo "Installing wasm32-unknown-unknown target..."
    rustup target add wasm32-unknown-unknown
fi

# Build each contract
for contract_dir in "$CONTRACTS_DIR"/*; do
    if [ -d "$contract_dir" ] && [ -f "$contract_dir/Cargo.toml" ]; then
        contract_name=$(basename "$contract_dir")
        echo ""
        echo "Building contract: $contract_name"
        
        cd "$contract_dir"
        cargo build --target wasm32-unknown-unknown --release
        
        # Copy WASM file to output directory
        WASM_FILE="target/wasm32-unknown-unknown/release/${contract_name}.wasm"
        if [ -f "$WASM_FILE" ]; then
            mkdir -p "$CONTRACTS_DIR/build"
            cp "$WASM_FILE" "$CONTRACTS_DIR/build/"
            echo "  Output: $CONTRACTS_DIR/build/${contract_name}.wasm"
            
            # Show file size
            SIZE=$(wc -c < "$CONTRACTS_DIR/build/${contract_name}.wasm")
            echo "  Size: $SIZE bytes"
        fi
    fi
done

echo ""
echo "=============================="
echo "Contract build complete!"
echo "Output directory: $CONTRACTS_DIR/build"

