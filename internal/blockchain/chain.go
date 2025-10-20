package blockchain

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/blockchain/layer1/internal/consensus"
	"github.com/blockchain/layer1/internal/crypto"
	"github.com/blockchain/layer1/internal/mempool"
	"github.com/blockchain/layer1/internal/state"
	"github.com/blockchain/layer1/internal/types"
	"github.com/blockchain/layer1/internal/vm"
	"github.com/dgraph-io/badger/v4"
)

const (
	// DefaultGasLimit is the default gas limit per block
	DefaultGasLimit = 10_000_000
	// MinGasPrice is the minimum gas price
	MinGasPrice = 1
)

// Blockchain manages the blockchain state and operations
type Blockchain struct {
	db          *badger.DB
	stateDB     *state.StateDB
	mempool     *mempool.Mempool
	validatorSet *consensus.ValidatorSet
	
	currentHeight uint64
	bestBlock     *types.Block
	mu            sync.RWMutex
	
	genesisHash crypto.Hash
}

// NewBlockchain creates a new blockchain
func NewBlockchain(dbPath string, validatorSet *consensus.ValidatorSet) (*Blockchain, error) {
	// Open block database
	opts := badger.DefaultOptions(dbPath + "/blocks")
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open block database: %w", err)
	}
	
	// Open state database
	stateDB, err := state.NewStateDB(dbPath + "/state")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to open state database: %w", err)
	}
	
	// Create mempool
	mp := mempool.NewMempool(stateDB)
	
	bc := &Blockchain{
		db:           db,
		stateDB:      stateDB,
		mempool:      mp,
		validatorSet: validatorSet,
		currentHeight: 0,
	}
	
	// Load or create genesis block
	if err := bc.initGenesis(); err != nil {
		db.Close()
		stateDB.Close()
		return nil, fmt.Errorf("failed to initialize genesis: %w", err)
	}
	
	return bc, nil
}

// Close closes the blockchain
func (bc *Blockchain) Close() error {
	if err := bc.stateDB.Close(); err != nil {
		return err
	}
	return bc.db.Close()
}

// initGenesis initializes the genesis block
func (bc *Blockchain) initGenesis() error {
	// Check if genesis exists
	var genesisBlock *types.Block
	
	err := bc.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(makeHeightKey(0))
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		
		return item.Value(func(val []byte) error {
			var err error
			genesisBlock, err = types.DeserializeBlock(val)
			return err
		})
	})
	
	if err != nil {
		return err
	}
	
	if genesisBlock != nil {
		bc.bestBlock = genesisBlock
		bc.currentHeight = 0
		bc.genesisHash = genesisBlock.Hash
		return nil
	}
	
	// Create genesis block
	genesisBlock = bc.createGenesisBlock()
	
	if err := bc.storeBlock(genesisBlock); err != nil {
		return err
	}
	
	bc.bestBlock = genesisBlock
	bc.currentHeight = 0
	bc.genesisHash = genesisBlock.Hash
	
	return nil
}

// createGenesisBlock creates the genesis block
func (bc *Blockchain) createGenesisBlock() *types.Block {
	header := types.NewBlockHeader(
		0,
		crypto.ZeroHash(),
		crypto.ZeroHash(),
		crypto.ZeroAddress(),
		DefaultGasLimit,
	)
	
	block := types.NewBlock(header, []*types.Transaction{})
	block.Finalize()
	
	return block
}

// GetCurrentHeight returns the current blockchain height
func (bc *Blockchain) GetCurrentHeight() uint64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.currentHeight
}

// GetBestBlock returns the best block
func (bc *Blockchain) GetBestBlock() *types.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.bestBlock
}

// GetBlockByHeight retrieves a block by height
func (bc *Blockchain) GetBlockByHeight(height uint64) (*types.Block, error) {
	var block *types.Block
	
	err := bc.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(makeHeightKey(height))
		if err != nil {
			return err
		}
		
		return item.Value(func(val []byte) error {
			var err error
			block, err = types.DeserializeBlock(val)
			return err
		})
	})
	
	return block, err
}

// GetBlockByHash retrieves a block by hash
func (bc *Blockchain) GetBlockByHash(hash crypto.Hash) (*types.Block, error) {
	var block *types.Block
	
	err := bc.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(makeHashKey(hash))
		if err != nil {
			return err
		}
		
		return item.Value(func(val []byte) error {
			var err error
			block, err = types.DeserializeBlock(val)
			return err
		})
	})
	
	return block, err
}

// ProposeBlock creates a new block proposal
func (bc *Blockchain) ProposeBlock(proposer crypto.Address, privKey *crypto.PrivateKey) (*types.Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Select transactions from mempool
	txs := bc.mempool.SelectTxs(DefaultGasLimit)
	
	// Get state root
	stateRoot, err := bc.stateDB.ComputeStateRoot()
	if err != nil {
		return nil, err
	}
	
	// Create block header
	header := types.NewBlockHeader(
		bc.currentHeight+1,
		bc.bestBlock.Hash,
		stateRoot,
		proposer,
		DefaultGasLimit,
	)
	
	// Create block
	block := types.NewBlock(header, txs)
	
	// Execute transactions and create receipts
	if err := bc.executeTransactions(block); err != nil {
		return nil, err
	}
	
	// Finalize and sign block
	block.Finalize()
	if err := block.Sign(privKey); err != nil {
		return nil, err
	}
	
	return block, nil
}

// ProcessBlock processes and commits a block
func (bc *Blockchain) ProcessBlock(block *types.Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Validate block
	if err := bc.validateBlock(block); err != nil {
		return fmt.Errorf("invalid block: %w", err)
	}
	
	// Execute transactions
	if err := bc.executeTransactions(block); err != nil {
		return fmt.Errorf("failed to execute transactions: %w", err)
	}
	
	// Store block
	if err := bc.storeBlock(block); err != nil {
		return fmt.Errorf("failed to store block: %w", err)
	}
	
	// Update state
	bc.currentHeight = block.Header.Height
	bc.bestBlock = block
	
	// Remove transactions from mempool
	for _, tx := range block.Transactions {
		bc.mempool.RemoveTx(tx.Hash)
	}
	
	// Commit state changes
	if err := bc.stateDB.Commit(); err != nil {
		return fmt.Errorf("failed to commit state: %w", err)
	}
	
	return nil
}

// validateBlock validates a block
func (bc *Blockchain) validateBlock(block *types.Block) error {
	// Basic validation
	if err := block.Validate(); err != nil {
		return err
	}
	
	// Check height
	if block.Header.Height != bc.currentHeight+1 {
		return fmt.Errorf("invalid height: expected %d, got %d", bc.currentHeight+1, block.Header.Height)
	}
	
	// Check previous hash
	if block.Header.PrevBlockHash != bc.bestBlock.Hash {
		return fmt.Errorf("invalid previous hash")
	}
	
	// Verify proposer
	proposer := bc.validatorSet.GetProposer(block.Header.Height, []byte("seed"))
	if proposer == nil {
		return fmt.Errorf("no proposer for height")
	}
	
	if proposer.Address != block.Header.ProposerAddress {
		return fmt.Errorf("invalid proposer")
	}
	
	return nil
}

// executeTransactions executes all transactions in a block
func (bc *Blockchain) executeTransactions(block *types.Block) error {
	block.Receipts = make([]*types.Receipt, 0, len(block.Transactions))
	totalGasUsed := uint64(0)
	
	// Create VM
	vmInstance, err := vm.NewVM(nil, bc.stateDB)
	if err != nil {
		return err
	}
	defer vmInstance.Close()
	
	for _, tx := range block.Transactions {
		receipt, err := bc.executeTx(tx, vmInstance)
		if err != nil {
			// Transaction failed, but we still include it with failed status
			receipt = &types.Receipt{
				TxHash:      tx.Hash,
				BlockHash:   block.Hash,
				BlockHeight: block.Header.Height,
				GasUsed:     tx.GasLimit, // Charge full gas on failure
				Status:      0,           // Failed
			}
		}
		
		block.Receipts = append(block.Receipts, receipt)
		totalGasUsed += receipt.GasUsed
	}
	
	block.Header.GasUsed = totalGasUsed
	return nil
}

// executeTx executes a single transaction
func (bc *Blockchain) executeTx(tx *types.Transaction, vmInstance *vm.VM) (*types.Receipt, error) {
	receipt := &types.Receipt{
		TxHash: tx.Hash,
		Status: 1, // Success by default
	}
	
	switch tx.Type {
	case types.TxTypeTransfer:
		if err := bc.stateDB.Transfer(tx.From, tx.To, tx.Amount); err != nil {
			return nil, err
		}
		receipt.GasUsed = 21000 // Base gas for transfer
		
	case types.TxTypeContractDeploy:
		result, contractAddr, err := vmInstance.DeployContract(tx.From, tx.Payload, tx.GasLimit)
		if err != nil {
			return nil, err
		}
		receipt.GasUsed = result.GasUsed
		receipt.ContractAddress = contractAddr
		receipt.Logs = convertLogs(result.Logs)
		
	case types.TxTypeContractCall:
		result := vmInstance.Execute(tx.From, tx.To, tx.Payload, tx.GasLimit, tx.Amount)
		if result.Error != nil {
			return nil, result.Error
		}
		receipt.GasUsed = result.GasUsed
		receipt.Logs = convertLogs(result.Logs)
	}
	
	// Increment nonce
	if err := bc.stateDB.IncrementNonce(tx.From); err != nil {
		return nil, err
	}
	
	return receipt, nil
}

// Helper functions

func makeHeightKey(height uint64) []byte {
	key := make([]byte, 9)
	key[0] = 'h'
	binary.BigEndian.PutUint64(key[1:], height)
	return key
}

func makeHashKey(hash crypto.Hash) []byte {
	return append([]byte("b"), hash.Bytes()...)
}

func (bc *Blockchain) storeBlock(block *types.Block) error {
	data, err := block.Serialize()
	if err != nil {
		return err
	}
	
	return bc.db.Update(func(txn *badger.Txn) error {
		// Store by height
		if err := txn.Set(makeHeightKey(block.Header.Height), data); err != nil {
			return err
		}
		
		// Store by hash
		return txn.Set(makeHashKey(block.Hash), data)
	})
}

func convertLogs(vmLogs []vm.Log) []types.Log {
	logs := make([]types.Log, len(vmLogs))
	for i, l := range vmLogs {
		logs[i] = types.Log{
			Address: l.Address,
			Topics:  l.Topics,
			Data:    l.Data,
		}
	}
	return logs
}

// AddTransaction adds a transaction to the mempool
func (bc *Blockchain) AddTransaction(tx *types.Transaction) error {
	return bc.mempool.AddTx(tx)
}

// GetTransaction retrieves a transaction from the mempool
func (bc *Blockchain) GetTransaction(hash crypto.Hash) (*types.Transaction, bool) {
	return bc.mempool.GetTx(hash)
}

// GetAccount retrieves an account from state
func (bc *Blockchain) GetAccount(addr crypto.Address) (*state.Account, error) {
	return bc.stateDB.GetAccount(addr)
}

