# Layer-1 Blockchain Architecture

## Overview

This document describes the architecture of the Layer-1 blockchain implementation, including component interactions, data flow, and design rationale.

## System Architecture

### High-Level Components

```
┌─────────────────────────────────────────────────────────────┐
│                      External Interfaces                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ HTTP RPC │  │   gRPC   │  │   CLI    │  │ Explorer │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
└───────┼─────────────┼─────────────┼─────────────┼──────────┘
        │             │             │             │
        └─────────────┴─────────────┴─────────────┘
                      │
┌─────────────────────┴─────────────────────────────────────┐
│                      Node Layer                            │
│  - Request routing                                         │
│  - Authentication & authorization                          │
│  - Metrics collection                                      │
└─────────────────────┬─────────────────────────────────────┘
                      │
┌─────────────────────┴─────────────────────────────────────┐
│                   Blockchain Core                          │
│                                                            │
│  ┌──────────────────────────────────────────────────┐    │
│  │              Consensus Engine (PoS)              │    │
│  │  - Validator set management                      │    │
│  │  - Proposer selection (weighted round-robin)     │    │
│  │  - Voting (prevote/precommit)                    │    │
│  │  - Finalization (2/3+ majority)                  │    │
│  │  - Slashing (double-sign, downtime)              │    │
│  └────────┬─────────────────────────────────────────┘    │
│           │                                               │
│  ┌────────┴──────────┐         ┌──────────────────┐     │
│  │   Block Processor │◄────────┤   P2P Network    │     │
│  │  - Validation     │         │  - Peer discovery│     │
│  │  - Execution      │         │  - Gossip        │     │
│  │  - Commitment     │         │  - Anti-entropy  │     │
│  └────────┬──────────┘         └──────────────────┘     │
│           │                                               │
│  ┌────────┴──────────┐         ┌──────────────────┐     │
│  │     Mempool       │         │   WASM VM        │     │
│  │  - Tx validation  │         │  - Gas metering  │     │
│  │  - Priority queue │         │  - Host functions│     │
│  │  - Eviction       │         │  - Determinism   │     │
│  └────────┬──────────┘         └────────┬─────────┘     │
│           │                              │               │
│  ┌────────┴──────────────────────────────┴─────────┐    │
│  │              State Manager                       │    │
│  │  - Account model                                 │    │
│  │  - Storage (key-value)                           │    │
│  │  - State root computation                        │    │
│  │  - Snapshot/revert                               │    │
│  └────────┬─────────────────────────────────────────┘    │
│           │                                               │
│  ┌────────┴─────────────────────────────────────────┐    │
│  │           Persistence Layer (BadgerDB)           │    │
│  │  - Blocks                                        │    │
│  │  - State (accounts, storage, code)               │    │
│  │  - Receipts                                      │    │
│  │  - Indexes                                       │    │
│  └──────────────────────────────────────────────────┘    │
└────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. Consensus Engine

**Purpose**: Implements Proof-of-Stake consensus with Byzantine Fault Tolerance.

**Key Responsibilities**:
- Validator set management (add, remove, update voting power)
- Proposer selection using deterministic weighted round-robin
- Vote collection and validation (prevote, precommit)
- Finalization when 2/3+ voting power commits
- Slashing for equivocation and downtime

**Algorithm**: Tendermint-inspired BFT consensus
- **Prevote phase**: Validators vote on proposed block
- **Precommit phase**: Validators commit to block after 2/3+ prevotes
- **Commit**: Block finalized after 2/3+ precommits

**Slashing Conditions**:
- Double-sign: 5% slash + jail
- Downtime: 1% slash (configurable threshold)

### 2. P2P Network

**Purpose**: Peer discovery and message propagation.

**Technology**: libp2p
- **Transport**: TCP
- **Discovery**: Kademlia DHT + bootstrap nodes
- **Protocol**: Custom `/layer1/1.0.0`

**Message Types**:
- `MsgTypeTx`: Transaction broadcast
- `MsgTypeBlock`: Block proposal
- `MsgTypeVote`: Consensus vote
- `MsgTypeBlockRequest`: Request block by height
- `MsgTypeStatusRequest`: Peer status query

**Anti-Entropy**: Periodic state sync to handle network partitions.

### 3. Transaction Mempool

**Purpose**: Manage pending transactions before block inclusion.

**Features**:
- Priority queue ordered by gas price (max heap)
- Per-account nonce ordering
- Size limits (10,000 txs global, 100 per account)
- Eviction policy: lowest gas price first

**Validation**:
- Signature verification
- Nonce check (must be >= current nonce)
- Balance check (must cover amount + gas)
- Gas limit check

### 4. Block Processor

**Purpose**: Execute and commit blocks to state.

**Block Lifecycle**:
1. **Proposal**: Proposer creates block from mempool txs
2. **Validation**: Verify header, signatures, tx validity
3. **Execution**: Apply transactions to state
4. **Finalization**: Compute state root, receipts root
5. **Commitment**: Persist block and state changes

**Transaction Execution**:
- Transfer: Simple balance update
- Contract Deploy: Store code, create contract account
- Contract Call: Execute WASM with gas metering

### 5. WASM Virtual Machine

**Purpose**: Execute smart contracts deterministically.

**Runtime**: wazero (pure Go, no CGo dependencies)

**Host Functions**:
- `get_balance(address)`: Query account balance
- `transfer(to, amount)`: Transfer native tokens
- `storage_get(key)`: Read contract storage
- `storage_set(key, value)`: Write contract storage
- `emit_log(data)`: Emit event log
- `get_caller()`: Get transaction sender
- `get_call_value()`: Get transferred value

**Gas Metering**:
- Base cost per WASM operation
- Storage read: 200 gas
- Storage write: 500 gas
- Transfer: 1000 gas
- Out-of-gas panics and reverts transaction

**Determinism**:
- No floating-point operations
- No non-deterministic syscalls
- Fixed memory limits
- Reproducible execution across nodes

### 6. State Management

**Purpose**: Maintain blockchain state (accounts, storage, code).

**Model**: Account-based (similar to Ethereum)

**Account Structure**:
```go
type Account struct {
    Address    Address
    Nonce      uint64
    Balance    uint64
    CodeHash   Hash      // For contracts
    IsContract bool
}
```

**Storage**:
- Key-value store per contract
- Prefix: `storage:<contract_address>:<key>`

**State Root**:
- Simple hash of all account data (production should use Merkle Patricia Trie)
- Included in block header for verification

**Snapshot/Revert**:
- In-memory cache for pending changes
- Atomic commit on block finalization
- Revert on validation failure

### 7. Persistence Layer

**Technology**: BadgerDB (LSM-tree based key-value store)

**Key Spaces**:
- `acc:<address>`: Account data
- `sto:<address>:<key>`: Contract storage
- `cod:<hash>`: Contract code
- `blk:<height>`: Block by height
- `b:<hash>`: Block by hash
- `tx:<hash>`: Transaction
- `rcp:<hash>`: Receipt

**Write-Ahead Log**: BadgerDB provides durability and crash recovery.

## Data Flow

### Transaction Submission

```
User → CLI → RPC → Mempool → Validation → Priority Queue
                                    ↓
                              Broadcast to Peers
```

### Block Production

```
Proposer Selection → Mempool.SelectTxs() → Create Block
                                    ↓
                            Execute Transactions
                                    ↓
                            Compute State Root
                                    ↓
                            Sign Block → Broadcast
```

### Block Commitment

```
Receive Block → Validate → Execute → Prevote
                                    ↓
                            Collect 2/3+ Prevotes
                                    ↓
                                Precommit
                                    ↓
                            Collect 2/3+ Precommits
                                    ↓
                            Commit to State → Persist
```

## Security Architecture

### Cryptography

- **Signatures**: Ed25519 (fast, secure)
- **Hashing**: SHA-256 (block headers, state root)
- **Address**: First 20 bytes of SHA-256(public_key)

### DoS Protection

- Mempool size limits
- Block size limits (gas limit)
- Max message size (10 MB)
- Max peers (configurable)
- Gas metering in VM

### Replay Protection

- Nonce per account (incremented on each tx)
- Chain ID (future: prevent cross-chain replay)

### Validator Security

- Slashing for misbehavior
- Jailing mechanism
- Stake-weighted voting (Sybil resistance)

## Performance Considerations

### Bottlenecks

1. **State I/O**: BadgerDB read/write latency
2. **WASM Execution**: Contract complexity
3. **Signature Verification**: Ed25519 ops per block
4. **Network Latency**: Block propagation time

### Optimizations

- **Caching**: In-memory account cache
- **Batch Writes**: Atomic commits to DB
- **Parallel Validation**: Signature verification (future)
- **State Sync**: Fast bootstrap for new nodes (future)

### Scalability

- **Current**: ~1000 TPS on single chain
- **Future**:
  - Sharding: Multiple parallel chains
  - Rollups: Off-chain execution, on-chain data
  - State channels: Off-chain transactions

## Failure Modes

### Network Partition

- Consensus halts if <2/3 validators reachable
- Resumes when partition heals
- No double-spend risk (BFT safety)

### Validator Crash

- Other validators continue if >2/3 online
- Crashed validator can rejoin and sync

### State Corruption

- BadgerDB checksums detect corruption
- Restore from snapshot or re-sync from peers

### Byzantine Validator

- Slashing for double-sign
- Requires >1/3 Byzantine validators to break safety
- Liveness requires >2/3 honest validators

## Future Enhancements

1. **Light Clients**: Verify block headers without full state
2. **State Sync**: Fast sync using state snapshots
3. **Sharding**: Horizontal scaling via multiple chains
4. **Optimistic Rollups**: Layer-2 scaling
5. **BLS Signatures**: Aggregate signatures for efficiency
6. **VRF**: Verifiable random function for proposer selection
7. **On-Chain Governance**: Protocol upgrades via voting

