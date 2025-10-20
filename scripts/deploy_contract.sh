#!/bin/bash

# Deploy WASM contract to local testnet
# Usage: ./scripts/deploy_contract.sh <contract_path> <rpc_url>

set -e

CONTRACT_PATH=${1:-"examples/contracts/build/token.wasm"}
RPC_URL=${2:-"http://localhost:8545"}
KEY_FILE=${3:-"data/deployer.key"}

echo "========================================="
echo "Layer-1 Blockchain - Contract Deployment"
echo "========================================="
echo ""

# Check if contract exists
if [ ! -f "$CONTRACT_PATH" ]; then
    echo "Error: Contract file not found: $CONTRACT_PATH"
    echo "Build contracts first: make contracts"
    exit 1
fi

# Generate deployer key if it doesn't exist
if [ ! -f "$KEY_FILE" ]; then
    echo "Generating deployer key..."
    ./build/layer1-cli keygen --output="$KEY_FILE"
    echo ""
fi

# Extract deployer address
DEPLOYER_ADDRESS=$(cat "$KEY_FILE" | grep -o '"address":"[^"]*"' | cut -d'"' -f4)
echo "Deployer address: $DEPLOYER_ADDRESS"
echo ""

# Check balance (requires RPC implementation)
echo "Checking deployer balance..."
# BALANCE=$(./build/layer1-cli balance --address="$DEPLOYER_ADDRESS" --rpc="$RPC_URL" 2>/dev/null || echo "0")
# echo "Balance: $BALANCE"
echo "Note: Balance check requires RPC implementation"
echo ""

# Read contract bytecode
echo "Reading contract bytecode..."
CONTRACT_SIZE=$(wc -c < "$CONTRACT_PATH")
echo "Contract size: $CONTRACT_SIZE bytes"
echo ""

# Create contract deployment transaction
echo "Creating deployment transaction..."

# For demo purposes, we'll show what the transaction would look like
# In a real implementation, this would use the CLI to create and sign the transaction

cat << EOF
Transaction Details:
--------------------
Type: ContractDeploy
From: $DEPLOYER_ADDRESS
Nonce: 0
GasLimit: 5000000
GasPrice: 1
Payload: <contract bytecode>

To deploy this contract:
1. Ensure the deployer account has sufficient balance
2. Use the CLI to create a deployment transaction:
   
   ./build/layer1-cli deploy \\
     --contract="$CONTRACT_PATH" \\
     --key="$KEY_FILE" \\
     --gas-limit=5000000 \\
     --gas-price=1 \\
     --rpc="$RPC_URL"

3. The contract address will be returned upon successful deployment

Note: Full RPC implementation required for automated deployment
EOF

echo ""
echo "========================================="
echo "Demo: Manual Contract Deployment Steps"
echo "========================================="
echo ""

# Show example of what would happen
echo "1. Transaction created and signed"
echo "2. Transaction broadcast to network"
echo "3. Transaction included in block"
echo "4. Contract deployed at address: 0x$(openssl rand -hex 20)"
echo ""

# If this were a token contract, show initialization
if [[ "$CONTRACT_PATH" == *"token"* ]]; then
    echo "Token Contract Initialization:"
    echo "------------------------------"
    echo "Function: initialize(initial_supply)"
    echo "Initial Supply: 1000000"
    echo ""
    echo "To initialize the token contract:"
    echo "  ./build/layer1-cli call \\"
    echo "    --contract=<contract_address> \\"
    echo "    --function=0 \\"
    echo "    --args=1000000 \\"
    echo "    --key=\"$KEY_FILE\" \\"
    echo "    --rpc=\"$RPC_URL\""
    echo ""
fi

echo "Deployment script completed!"
echo ""
echo "Next steps:"
echo "1. Implement full RPC server (api/rpc/server.go)"
echo "2. Add CLI commands for contract deployment and calls"
echo "3. Test contract deployment on local testnet"
echo ""

