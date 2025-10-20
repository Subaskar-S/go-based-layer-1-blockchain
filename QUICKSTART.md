# Quick Start Guide

Get up and running with the Layer-1 blockchain in 5 minutes.

## Prerequisites

- **Go 1.21+**: [Download](https://golang.org/dl/)
- **Make**: Usually pre-installed on Linux/Mac, [Windows](http://gnuwin32.sourceforge.net/packages/make.htm)
- **Git**: [Download](https://git-scm.com/downloads)

Optional:
- **Rust**: For building WASM contracts ([rustup](https://rustup.rs/))
- **Docker**: For containerized deployment ([Docker](https://www.docker.com/))

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/YOUR_USERNAME/layer1-blockchain.git
cd layer1-blockchain
```

### 2. Install Dependencies

```bash
make deps
```

### 3. Build Binaries

```bash
make build
```

This creates:
- `build/layer1-node` - Blockchain node
- `build/layer1-cli` - Command-line client

## Running Your First Node

### Option 1: Single Development Node

```bash
make run-dev
```

This starts a single node on:
- HTTP RPC: `http://localhost:8545`
- gRPC: `localhost:9090`
- P2P: `localhost:30303`

### Option 2: 4-Node Local Testnet

```bash
make localnet
```

This starts 4 validator nodes:

| Node | HTTP | gRPC | P2P |
|------|------|------|-----|
| node-0 | 8545 | 9090 | 30303 |
| node-1 | 8546 | 9091 | 30304 |
| node-2 | 8547 | 9092 | 30305 |
| node-3 | 8548 | 9093 | 30306 |

View logs:
```bash
tail -f data/node-0/node.log
```

Stop testnet:
```bash
make localnet-stop
```

## Using the CLI

### Generate a Keypair

```bash
./build/layer1-cli keygen --output=alice.key
```

Output:
```
Generated new keypair
Address: 0xA1B2C3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0
Public Key: 0x1a2b3c...
Saved to: alice.key
```

### Create a Transaction

```bash
./build/layer1-cli transfer \
  --from=0xA1B2C3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0 \
  --to=0xB2C3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1 \
  --amount=1000000 \
  --nonce=0 \
  --gas-limit=21000 \
  --gas-price=1 \
  --key=alice.key
```

Output:
```
Transaction created and signed:
Hash: 0x7f8a9b0c...
Raw: 0x1a2b3c4d...
```

### Query Balance (requires RPC)

```bash
./build/layer1-cli balance \
  --address=0xA1B2C3D4E5F6a7b8c9d0e1f2a3b4c5d6e7f8a9b0 \
  --rpc=http://localhost:8545
```

### Query Block

```bash
./build/layer1-cli block \
  --height=10 \
  --rpc=http://localhost:8545
```

## Building Smart Contracts

### Install Rust

```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
rustup target add wasm32-unknown-unknown
```

### Build Example Contracts

```bash
make contracts
```

This builds contracts in `examples/contracts/build/`:
- `token.wasm` - ERC20-like token

### Deploy a Contract

```bash
./scripts/deploy_contract.sh examples/contracts/build/token.wasm
```

## Running Tests

### Unit Tests

```bash
make test
```

### Integration Tests

```bash
make test-integration
```

### Benchmarks

```bash
make bench
```

## Docker Deployment

### Build Image

```bash
make docker
```

### Run with Docker Compose

```bash
make docker-compose-up
```

This starts a 4-node testnet in Docker containers.

Stop:
```bash
make docker-compose-down
```

## Monitoring

### View Node Status

```bash
curl http://localhost:8545/status | jq
```

### View Validators

```bash
curl http://localhost:8545/validators | jq
```

### View Latest Block

```bash
curl http://localhost:8545/block/latest | jq
```

## Common Tasks

### Clean Build Artifacts

```bash
make clean
```

### Format Code

```bash
make fmt
```

### Run Linter

```bash
make lint
```

### Generate Test Accounts

```bash
make gen-accounts
```

## Troubleshooting

### Build Fails

```bash
# Clean and rebuild
make clean
make deps
make build
```

### Port Already in Use

```bash
# Stop existing processes
make localnet-stop

# Or manually
pkill -f layer1-node
```

### Database Corruption

```bash
# Remove data directory
rm -rf ./data

# Restart
make localnet
```

### Tests Fail

```bash
# Ensure dependencies are up to date
go mod tidy
go mod download

# Run tests with verbose output
go test -v ./...
```

## Next Steps

1. **Read the Documentation**
   - [README.md](README.md) - Full documentation
   - [docs/architecture.md](docs/architecture.md) - System architecture
   - [docs/design_decisions.md](docs/design_decisions.md) - Design rationale

2. **Explore the Code**
   - Start with `cmd/node/main.go` - Node entry point
   - Check `internal/blockchain/chain.go` - Core blockchain logic
   - Review `internal/consensus/consensus.go` - PoS consensus

3. **Build Something**
   - Write a WASM smart contract
   - Create a custom transaction type
   - Implement a new RPC endpoint

4. **Contribute**
   - See [CONTRIBUTING.md](CONTRIBUTING.md)
   - Open issues for bugs or features
   - Submit pull requests

## Getting Help

- **Documentation**: Check `docs/` directory
- **Issues**: Open a GitHub issue
- **Examples**: See `examples/` directory
- **Tests**: Review test files for usage examples

## Resources

- **Go Documentation**: https://golang.org/doc/
- **libp2p**: https://libp2p.io/
- **wazero**: https://wazero.io/
- **BadgerDB**: https://dgraph.io/docs/badger/
- **Tendermint**: https://docs.tendermint.com/

## Quick Reference

### Make Commands

```bash
make build              # Build binaries
make test               # Run tests
make bench              # Run benchmarks
make localnet           # Start 4-node testnet
make localnet-stop      # Stop testnet
make docker             # Build Docker image
make docker-compose-up  # Start Docker testnet
make clean              # Clean build artifacts
make fmt                # Format code
make lint               # Run linter
make contracts          # Build WASM contracts
```

### CLI Commands

```bash
layer1-cli keygen       # Generate keypair
layer1-cli transfer     # Create transfer transaction
layer1-cli balance      # Query balance
layer1-cli block        # Query block
```

### Node Flags

```bash
--data-dir              # Data directory
--http-port             # HTTP RPC port
--grpc-port             # gRPC port
--p2p-port              # P2P port
--validator             # Run as validator
--validator-key         # Validator key file
--bootstrap-nodes       # Bootstrap node addresses
```

---

**You're all set!** 🚀

Start building on the Layer-1 blockchain.

