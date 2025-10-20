package mempool

import (
	"container/heap"
	"fmt"
	"sync"

	"github.com/blockchain/layer1/internal/crypto"
	"github.com/blockchain/layer1/internal/state"
	"github.com/blockchain/layer1/internal/types"
)

const (
	// MaxMempoolSize is the maximum number of transactions in the mempool
	MaxMempoolSize = 10000
	// MaxAccountTxs is the maximum number of pending txs per account
	MaxAccountTxs = 100
)

// Mempool manages pending transactions
type Mempool struct {
	mu           sync.RWMutex
	txs          map[crypto.Hash]*types.Transaction // All transactions by hash
	accountTxs   map[crypto.Address][]*types.Transaction // Transactions by account
	priorityQueue *TxPriorityQueue // Priority queue for tx selection
	stateDB      *state.StateDB
}

// NewMempool creates a new mempool
func NewMempool(stateDB *state.StateDB) *Mempool {
	pq := &TxPriorityQueue{}
	heap.Init(pq)
	
	return &Mempool{
		txs:          make(map[crypto.Hash]*types.Transaction),
		accountTxs:   make(map[crypto.Address][]*types.Transaction),
		priorityQueue: pq,
		stateDB:      stateDB,
	}
}

// AddTx adds a transaction to the mempool
func (m *Mempool) AddTx(tx *types.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if already in mempool
	if _, exists := m.txs[tx.Hash]; exists {
		return fmt.Errorf("transaction already in mempool")
	}
	
	// Validate transaction
	if err := m.validateTx(tx); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}
	
	// Check mempool size limit
	if len(m.txs) >= MaxMempoolSize {
		// Evict lowest priority transaction
		if err := m.evictLowestPriority(); err != nil {
			return fmt.Errorf("mempool full and cannot evict: %w", err)
		}
	}
	
	// Check per-account limit
	accountTxs := m.accountTxs[tx.From]
	if len(accountTxs) >= MaxAccountTxs {
		return fmt.Errorf("too many pending transactions for account")
	}
	
	// Add to mempool
	m.txs[tx.Hash] = tx
	m.accountTxs[tx.From] = append(m.accountTxs[tx.From], tx)
	
	// Add to priority queue
	heap.Push(m.priorityQueue, &TxPriorityItem{
		tx:       tx,
		priority: tx.GasPrice,
	})
	
	return nil
}

// RemoveTx removes a transaction from the mempool
func (m *Mempool) RemoveTx(hash crypto.Hash) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	tx, exists := m.txs[hash]
	if !exists {
		return
	}
	
	delete(m.txs, hash)
	
	// Remove from account txs
	accountTxs := m.accountTxs[tx.From]
	for i, accTx := range accountTxs {
		if accTx.Hash == hash {
			m.accountTxs[tx.From] = append(accountTxs[:i], accountTxs[i+1:]...)
			break
		}
	}
	
	if len(m.accountTxs[tx.From]) == 0 {
		delete(m.accountTxs, tx.From)
	}
}

// GetTx retrieves a transaction by hash
func (m *Mempool) GetTx(hash crypto.Hash) (*types.Transaction, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	tx, exists := m.txs[hash]
	return tx, exists
}

// SelectTxs selects transactions for block inclusion
// Returns transactions sorted by priority (gas price)
func (m *Mempool) SelectTxs(maxGas uint64) []*types.Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	selected := make([]*types.Transaction, 0)
	totalGas := uint64(0)
	accountNonces := make(map[crypto.Address]uint64)
	
	// Get current nonces from state
	for addr := range m.accountTxs {
		nonce, err := m.stateDB.GetNonce(addr)
		if err != nil {
			continue
		}
		accountNonces[addr] = nonce
	}
	
	// Create a copy of the priority queue for iteration
	pqCopy := make(TxPriorityQueue, len(*m.priorityQueue))
	copy(pqCopy, *m.priorityQueue)
	heap.Init(&pqCopy)
	
	// Select transactions in priority order
	for pqCopy.Len() > 0 {
		item := heap.Pop(&pqCopy).(*TxPriorityItem)
		tx := item.tx
		
		// Check if we have space for this tx
		if totalGas+tx.GasLimit > maxGas {
			continue
		}
		
		// Check nonce ordering
		expectedNonce := accountNonces[tx.From]
		if tx.Nonce != expectedNonce {
			continue
		}
		
		selected = append(selected, tx)
		totalGas += tx.GasLimit
		accountNonces[tx.From]++
		
		if totalGas >= maxGas {
			break
		}
	}
	
	return selected
}

// Size returns the number of transactions in the mempool
func (m *Mempool) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.txs)
}

// Clear clears all transactions from the mempool
func (m *Mempool) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.txs = make(map[crypto.Hash]*types.Transaction)
	m.accountTxs = make(map[crypto.Address][]*types.Transaction)
	m.priorityQueue = &TxPriorityQueue{}
	heap.Init(m.priorityQueue)
}

// validateTx validates a transaction before adding to mempool
func (m *Mempool) validateTx(tx *types.Transaction) error {
	// Basic validation
	if err := tx.Validate(); err != nil {
		return err
	}
	
	// Get account from state
	account, err := m.stateDB.GetAccount(tx.From)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	
	// Check nonce (must be >= current nonce)
	if tx.Nonce < account.Nonce {
		return fmt.Errorf("nonce too low: have %d, want >= %d", tx.Nonce, account.Nonce)
	}
	
	// Check balance
	cost := tx.Cost()
	if account.Balance < cost {
		return fmt.Errorf("insufficient balance: have %d, need %d", account.Balance, cost)
	}
	
	// Verify signature
	pubKey, err := crypto.NewPublicKeyFromBytes(tx.From.Bytes()[:32]) // Simplified
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}
	
	if !tx.VerifySignature(pubKey) {
		return fmt.Errorf("invalid signature")
	}
	
	return nil
}

// evictLowestPriority evicts the transaction with the lowest gas price
func (m *Mempool) evictLowestPriority() error {
	if m.priorityQueue.Len() == 0 {
		return fmt.Errorf("no transactions to evict")
	}
	
	// Pop the lowest priority item (min heap)
	item := heap.Pop(m.priorityQueue).(*TxPriorityItem)
	
	// Remove from maps
	delete(m.txs, item.tx.Hash)
	
	accountTxs := m.accountTxs[item.tx.From]
	for i, tx := range accountTxs {
		if tx.Hash == item.tx.Hash {
			m.accountTxs[item.tx.From] = append(accountTxs[:i], accountTxs[i+1:]...)
			break
		}
	}
	
	if len(m.accountTxs[item.tx.From]) == 0 {
		delete(m.accountTxs, item.tx.From)
	}
	
	return nil
}

// TxPriorityItem represents a transaction in the priority queue
type TxPriorityItem struct {
	tx       *types.Transaction
	priority uint64 // Gas price
	index    int    // Index in the heap
}

// TxPriorityQueue implements heap.Interface for transaction prioritization
type TxPriorityQueue []*TxPriorityItem

func (pq TxPriorityQueue) Len() int { return len(pq) }

func (pq TxPriorityQueue) Less(i, j int) bool {
	// Higher gas price = higher priority (max heap)
	return pq[i].priority > pq[j].priority
}

func (pq TxPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *TxPriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*TxPriorityItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *TxPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

