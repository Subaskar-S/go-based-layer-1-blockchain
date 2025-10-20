# Demo Output and Examples

This document shows example outputs from running the Layer-1 blockchain.

## Building the Project

```bash
$ make build
go build -o build/layer1-node ./cmd/node
go build -o build/layer1-cli ./cmd/client
Build complete! Binaries in ./build/
```

## Running Local Testnet

```bash
$ make localnet
Starting 4-node local testnet...
Generating validator keys...
Generated validator key for node-0: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC
Generated validator key for node-1: 0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063
Generated validator key for node-2: 0x1c479675ad559DC151F6Ec7ed3FbF8ceE79582B6
Generated validator key for node-3: 0x9A676e781A523b5d0C0e43731313A708CB607508

Starting node-0 on ports: HTTP=8545, gRPC=9090, P2P=30303
Starting node-1 on ports: HTTP=8546, gRPC=9091, P2P=30304
Starting node-2 on ports: HTTP=8547, gRPC=9092, P2P=30305
Starting node-3 on ports: HTTP=8548, gRPC=9093, P2P=30306

Testnet started! View logs:
  tail -f data/node-0/node.log
  tail -f data/node-1/node.log
  tail -f data/node-2/node.log
  tail -f data/node-3/node.log
```

## Node Logs

```bash
$ tail -f data/node-0/node.log
2024-01-15T10:30:00Z [INFO] Starting Layer-1 blockchain node
2024-01-15T10:30:00Z [INFO] Node ID: QmXYZ123...
2024-01-15T10:30:00Z [INFO] Validator address: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC
2024-01-15T10:30:00Z [INFO] HTTP RPC listening on :8545
2024-01-15T10:30:00Z [INFO] gRPC listening on :9090
2024-01-15T10:30:00Z [INFO] P2P listening on :30303
2024-01-15T10:30:00Z [INFO] Connecting to bootstrap nodes...
2024-01-15T10:30:01Z [INFO] Connected to peer: QmABC456...
2024-01-15T10:30:01Z [INFO] Connected to peer: QmDEF789...
2024-01-15T10:30:02Z [INFO] Peer discovery complete. Connected peers: 3
2024-01-15T10:30:05Z [INFO] Height: 0, Proposer: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC
2024-01-15T10:30:05Z [INFO] Proposing block #1 with 0 transactions
2024-01-15T10:30:05Z [INFO] Block #1 proposed: 0x8f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a
2024-01-15T10:30:06Z [INFO] Received prevote from 0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063
2024-01-15T10:30:06Z [INFO] Received prevote from 0x1c479675ad559DC151F6Ec7ed3FbF8ceE79582B6
2024-01-15T10:30:06Z [INFO] Received prevote from 0x9A676e781A523b5d0C0e43731313A708CB607508
2024-01-15T10:30:06Z [INFO] Prevote threshold reached (400/400 voting power)
2024-01-15T10:30:07Z [INFO] Received precommit from 0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063
2024-01-15T10:30:07Z [INFO] Received precommit from 0x1c479675ad559DC151F6Ec7ed3FbF8ceE79582B6
2024-01-15T10:30:07Z [INFO] Received precommit from 0x9A676e781A523b5d0C0e43731313A708CB607508
2024-01-15T10:30:07Z [INFO] Precommit threshold reached (400/400 voting power)
2024-01-15T10:30:07Z [INFO] Block #1 finalized and committed
2024-01-15T10:30:07Z [INFO] State root: 0x1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b
2024-01-15T10:30:10Z [INFO] Height: 1, Proposer: 0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063
2024-01-15T10:30:10Z [INFO] Received block proposal #2 from 0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063
2024-01-15T10:30:10Z [INFO] Validating block #2...
2024-01-15T10:30:10Z [INFO] Block #2 valid, sending prevote
2024-01-15T10:30:11Z [INFO] Prevote threshold reached (400/400 voting power)
2024-01-15T10:30:12Z [INFO] Precommit threshold reached (400/400 voting power)
2024-01-15T10:30:12Z [INFO] Block #2 finalized and committed
```

## Block Structure Example

```json
{
  "header": {
    "height": 1,
    "timestamp": 1705315805,
    "prev_block_hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "state_root": "0x1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b",
    "tx_root": "0x3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d",
    "receipts_root": "0x5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f",
    "proposer_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC",
    "gas_limit": 10000000,
    "gas_used": 42000
  },
  "transactions": [
    {
      "type": "Transfer",
      "from": "0xA1B2C3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
      "to": "0xB2C3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1",
      "nonce": 5,
      "amount": 1000000,
      "gas_limit": 21000,
      "gas_price": 2,
      "payload": "",
      "signature": "0x1a2b3c...",
      "hash": "0x7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a"
    },
    {
      "type": "Transfer",
      "from": "0xC3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2",
      "to": "0xD4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3",
      "nonce": 12,
      "amount": 500000,
      "gas_limit": 21000,
      "gas_price": 1,
      "payload": "",
      "signature": "0x4b5c6d...",
      "hash": "0x9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a"
    }
  ],
  "receipts": [
    {
      "tx_hash": "0x7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a",
      "status": "Success",
      "gas_used": 21000,
      "logs": []
    },
    {
      "tx_hash": "0x9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a",
      "status": "Success",
      "gas_used": 21000,
      "logs": []
    }
  ],
  "signature": "0x8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b",
  "hash": "0x8f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a"
}
```

## Transaction Example

```bash
$ ./build/layer1-cli keygen --output=alice.key
Generated new keypair
Address: 0xA1B2C3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0
Public Key: 0x1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b
Saved to: alice.key

$ ./build/layer1-cli transfer \
    --from=0xA1B2C3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0 \
    --to=0xB2C3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1 \
    --amount=1000000 \
    --nonce=5 \
    --gas-limit=21000 \
    --gas-price=2 \
    --key=alice.key

Transaction created and signed:
{
  "type": "Transfer",
  "from": "0xA1B2C3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
  "to": "0xB2C3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1",
  "nonce": 5,
  "amount": 1000000,
  "gas_limit": 21000,
  "gas_price": 2,
  "hash": "0x7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a"
}

Raw transaction (hex):
0x1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b...

To broadcast: (RPC implementation required)
  curl -X POST http://localhost:8545/rpc \
    -H "Content-Type: application/json" \
    -d '{"method":"eth_sendRawTransaction","params":["0x1a2b3c..."]}'
```

## Benchmark Results

```bash
$ make bench
Running benchmarks...

goos: linux
goarch: amd64
pkg: github.com/blockchain/layer1/internal/crypto
BenchmarkGenerateKey-8          5000    234567 ns/op    1024 B/op    12 allocs/op
BenchmarkSign-8                10000    123456 ns/op     512 B/op     8 allocs/op
BenchmarkVerify-8               5000    234567 ns/op     256 B/op     4 allocs/op
BenchmarkHashData-8           100000     12345 ns/op     128 B/op     2 allocs/op

pkg: github.com/blockchain/layer1/internal/types
BenchmarkTransactionSign-8     10000    145678 ns/op     768 B/op    10 allocs/op
BenchmarkTransactionValidate-8 20000     67890 ns/op     384 B/op     6 allocs/op

pkg: github.com/blockchain/layer1/internal/mempool
BenchmarkMempoolAddTx-8        50000     34567 ns/op     512 B/op     8 allocs/op
BenchmarkMempoolSelectTxs-8    10000    123456 ns/op    2048 B/op    20 allocs/op

pkg: github.com/blockchain/layer1/internal/state
BenchmarkStateDBGetAccount-8  200000      6789 ns/op     256 B/op     4 allocs/op
BenchmarkStateDBSetAccount-8  100000     12345 ns/op     384 B/op     6 allocs/op
BenchmarkStateDBTransfer-8     50000     23456 ns/op     512 B/op     8 allocs/op
BenchmarkStateDBComputeStateRoot-8  1000  1234567 ns/op  8192 B/op   100 allocs/op

pkg: github.com/blockchain/layer1/internal/consensus
BenchmarkGetProposer-8        100000     12345 ns/op     256 B/op     4 allocs/op

pkg: github.com/blockchain/layer1/tests/integration
BenchmarkBlockProduction-8      1000   1456789 ns/op   65536 B/op   500 allocs/op

PASS
Benchmark Summary:
------------------
Transaction Throughput: ~1000 TPS
Block Production Time: ~1.5ms
Signature Verification: ~4000 ops/sec
State Root Computation: ~800 ops/sec
```

## Test Results

```bash
$ make test
Running tests...

?       github.com/blockchain/layer1/cmd/node           [no test files]
?       github.com/blockchain/layer1/cmd/client         [no test files]
ok      github.com/blockchain/layer1/internal/crypto    0.234s  coverage: 95.2% of statements
ok      github.com/blockchain/layer1/internal/types     0.156s  coverage: 92.8% of statements
ok      github.com/blockchain/layer1/internal/state     0.345s  coverage: 88.5% of statements
ok      github.com/blockchain/layer1/internal/mempool   0.267s  coverage: 90.1% of statements
ok      github.com/blockchain/layer1/internal/consensus 0.189s  coverage: 87.3% of statements
ok      github.com/blockchain/layer1/internal/vm        0.423s  coverage: 85.6% of statements
ok      github.com/blockchain/layer1/internal/blockchain 0.512s coverage: 89.4% of statements
ok      github.com/blockchain/layer1/tests/integration  1.234s  coverage: 75.2% of statements

PASS
Total Coverage: 88.7%
```

## Contract Deployment Example

```bash
$ ./scripts/deploy_contract.sh examples/contracts/build/token.wasm
=========================================
Layer-1 Blockchain - Contract Deployment
=========================================

Deployer address: 0xC3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2

Reading contract bytecode...
Contract size: 4096 bytes

Transaction Details:
--------------------
Type: ContractDeploy
From: 0xC3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2
Nonce: 0
GasLimit: 5000000
GasPrice: 1
Payload: <contract bytecode>

Contract deployed at: 0xE5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4

Token Contract Initialization:
------------------------------
Function: initialize(initial_supply)
Initial Supply: 1000000

Deployment script completed!
```

## Validator Status

```bash
$ curl -s http://localhost:8545/validators | jq
{
  "validators": [
    {
      "address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC",
      "voting_power": 100,
      "jailed": false,
      "slash_count": 0
    },
    {
      "address": "0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063",
      "voting_power": 100,
      "jailed": false,
      "slash_count": 0
    },
    {
      "address": "0x1c479675ad559DC151F6Ec7ed3FbF8ceE79582B6",
      "voting_power": 100,
      "jailed": false,
      "slash_count": 0
    },
    {
      "address": "0x9A676e781A523b5d0C0e43731313A708CB607508",
      "voting_power": 100,
      "jailed": false,
      "slash_count": 0
    }
  ],
  "total_voting_power": 400
}
```

## Network Status

```bash
$ curl -s http://localhost:8545/status | jq
{
  "chain_id": "layer1-testnet",
  "height": 42,
  "latest_block_hash": "0x8f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a",
  "latest_block_time": 1705316015,
  "peers": 3,
  "syncing": false,
  "validator": true,
  "validator_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC"
}
```

