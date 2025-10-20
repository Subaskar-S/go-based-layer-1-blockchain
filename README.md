# Layer-1 Blockchain

A production-ready Layer-1 blockchain implementation in Go featuring Proof-of-Stake (PoS) consensus, P2P networking via libp2p, and WASM-based smart contracts.

## Features

- **Consensus**: Tendermint-style PoS with validator set management, staking, and slashing
- **P2P Networking**: libp2p-based gossip protocol for block and transaction propagation
- **Smart Contracts**: WASM runtime (wazero) with deterministic execution and gas metering
- **State Management**: Account-based model with BadgerDB persistence
- **Transaction Pool**: Priority-based mempool with gas price ordering
- **Security**: Ed25519 signatures, replay protection, DoS limits
- **Tooling**: CLI for key management, transaction creation, and node operations

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Node Layer                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │   RPC    │  │   CLI    │  │ Metrics  │  │ Explorer │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
└───────┼─────────────┼─────────────┼─────────────┼──────────┘
        │             │             │             │
┌───────┴─────────────┴─────────────┴─────────────┴──────────┐
│                    Blockchain Core                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │  Consensus   │  │   Mempool    │  │     P2P      │     │
│  │   (PoS)      │  │              │  │   Network    │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         │                  │                  │             │
│  ┌──────┴──────────────────┴──────────────────┴───────┐   │
│  │              Block Processor                        │   │
│  └──────┬──────────────────────────────────────┬───────┘   │
│         │                                       │           │
│  ┌──────┴───────┐                      ┌───────┴────────┐  │
│  │  State DB    │                      │   WASM VM      │  │
│  │  (BadgerDB)  │                      │   (wazero)     │  │
│  └──────────────┘                      └────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Make
- (Optional) Rust for building WASM contracts
- (Optional) Docker for containerized deployment

### Build

```bash
# Clone the repository
git clone <repository-url>
cd layer1-blockchain

# Download dependencies
make deps

# Build binaries
make build

# Binaries will be in ./build/
# - layer1-node: Blockchain node
# - layer1-cli: Command-line client
```

### Run a Single Node

```bash
# Start a development node
make run-dev

# Or manually:
./build/layer1-node start \
  --data-dir=./data/dev \
  --http-port=8545 \
  --grpc-port=9090 \
  --p2p-port=30303 \
  --validator \
  --validator-key=./data/validator.key
```

### Run a Local 4-Node Testnet

```bash
# Start local testnet with 4 validators
make localnet

# This will:
# - Generate 4 validator keys
# - Start 4 nodes on ports 8545-8548 (HTTP), 9090-9093 (gRPC), 30303-30306 (P2P)
# - Connect nodes via P2P
# - Begin block production

# View logs
tail -f ./data/node-0/node.log

# Stop testnet
make localnet-stop
```

### Using the CLI

```bash
# Generate a new keypair
./build/layer1-cli keygen --output=my-key.json

# Create a transfer transaction
./build/layer1-cli transfer \
  --from=<sender-address> \
  --to=<recipient-address> \
  --amount=1000 \
  --nonce=0 \
  --key=my-key.json

# Query account balance (requires running node)
./build/layer1-cli balance \
  --address=<account-address> \
  --rpc=http://localhost:8545

# Query block information
./build/layer1-cli block \
  --height=10 \
  --rpc=http://localhost:8545
```

## Smart Contracts

### Building WASM Contracts

```bash
# Install Rust and wasm32 target
rustup target add wasm32-unknown-unknown

# Build example contracts
make contracts

# Contracts will be in ./examples/contracts/build/
```

### Example: Token Contract

The included token contract demonstrates:
- ERC20-like functionality
- Storage operations
- Event emission
- Balance tracking

```bash
# Deploy token contract
# (Requires running node and RPC implementation)

# Contract functions:
# 0: initialize(initial_supply: u64)
# 1: transfer(to: address, amount: u64) -> bool
# 2: balance_of(address: address) -> u64
# 3: total_supply() -> u64
```

## Testing

```bash
# Run all tests
make test

# Run integration tests
make test-integration

# Run benchmarks
make bench

# View coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Docker Deployment

```bash
# Build Docker image
make docker

# Start multi-node testnet with Docker Compose
make docker-compose-up

# Stop Docker testnet
make docker-compose-down
```

## Configuration

### Node Configuration

- `--data-dir`: Data directory for blockchain and state (default: `./data`)
- `--http-port`: HTTP RPC port (default: `8545`)
- `--grpc-port`: gRPC port (default: `9090`)
- `--p2p-port`: P2P networking port (default: `30303`)
- `--metrics-port`: Prometheus metrics port (default: `6060`)
- `--validator`: Run as validator (default: `false`)
- `--validator-key`: Path to validator private key file
- `--bootstrap-nodes`: Comma-separated list of bootstrap node multiaddrs

### Consensus Parameters

Defined in `internal/consensus/consensus.go`:
- `BlockTime`: 5 seconds
- `VoteTimeout`: 3 seconds
- `SlashFractionDoubleSign`: 5%
- `SlashFractionDowntime`: 1%

### Gas Parameters

Defined in `internal/blockchain/chain.go`:
- `DefaultGasLimit`: 10,000,000 per block
- `MinGasPrice`: 1

## Project Structure

```
.
├── cmd/
│   ├── node/           # Node binary
│   └── client/         # CLI client binary
├── internal/
│   ├── blockchain/     # Blockchain core logic
│   ├── consensus/      # PoS consensus engine
│   ├── crypto/         # Cryptographic primitives
│   ├── mempool/        # Transaction pool
│   ├── p2p/            # P2P networking
│   ├── state/          # State management
│   ├── types/          # Core types (Block, Transaction, etc.)
│   └── vm/             # WASM virtual machine
├── api/                # RPC/API definitions
├── examples/
│   └── contracts/      # Example WASM contracts
├── scripts/            # Automation scripts
├── docs/               # Documentation
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## Performance

Expected performance on modern hardware:
- **Transaction Throughput**: ~1000 TPS
- **Block Time**: 5 seconds
- **Finality**: ~15 seconds (3 blocks)
- **WASM Execution**: ~10,000 ops/ms

Run benchmarks:
```bash
make bench
```

## Security Considerations

### Current Implementation

- ✅ Ed25519 signature verification
- ✅ Nonce-based replay protection
- ✅ Gas metering for DoS prevention
- ✅ Mempool size limits
- ✅ Block size limits
- ✅ Validator slashing for equivocation

### Production Hardening (TODO)

- [ ] TLS for RPC endpoints
- [ ] Rate limiting on RPC
- [ ] HSM/KMS integration for validator keys
- [ ] Formal verification of consensus logic
- [ ] External security audit
- [ ] DDoS protection at network layer
- [ ] State sync for fast bootstrapping
- [ ] Light client support

## Roadmap

### Phase 1: Core Functionality (Current)
- ✅ PoS consensus
- ✅ P2P networking
- ✅ WASM VM
- ✅ Basic CLI

### Phase 2: Production Features
- [ ] Complete RPC implementation (JSON-RPC 2.0)
- [ ] Web-based block explorer
- [ ] State sync protocol
- [ ] Light client
- [ ] Improved slashing conditions

### Phase 3: Advanced Features
- [ ] Cross-shard transactions (sharding)
- [ ] On-chain governance
- [ ] Optimistic rollups support
- [ ] MEV protection
- [ ] Advanced cryptography (BLS signatures, VRF)

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Tendermint for consensus design inspiration
- Ethereum for smart contract model
- libp2p for P2P networking
- wazero for pure-Go WASM runtime

## Support

For questions and support:
- GitHub Issues: <repository-url>/issues
- Documentation: `./docs/`

## Interview Talking Points

See `docs/interview_talking_points.md` for detailed discussion of:
- Design decisions and tradeoffs
- Consensus algorithm choice
- Security considerations
- Scaling strategies
- Production deployment considerations

