package state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/blockchain/layer1/internal/crypto"
	"github.com/dgraph-io/badger/v4"
)

// Prefixes for different key types
var (
	accountPrefix  = []byte("acc:")
	storagePrefix  = []byte("sto:")
	codePrefix     = []byte("cod:")
	blockPrefix    = []byte("blk:")
	txPrefix       = []byte("tx:")
	receiptPrefix  = []byte("rcp:")
	heightPrefix   = []byte("hgt:")
	stateRootPrefix = []byte("srt:")
)

// StateDB manages the blockchain state
type StateDB struct {
	db    *badger.DB
	mu    sync.RWMutex
	cache map[crypto.Address]*Account // In-memory cache
}

// NewStateDB creates a new state database
func NewStateDB(path string) (*StateDB, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable badger logging
	
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	return &StateDB{
		db:    db,
		cache: make(map[crypto.Address]*Account),
	}, nil
}

// Close closes the database
func (s *StateDB) Close() error {
	return s.db.Close()
}

// GetAccount retrieves an account from the state
func (s *StateDB) GetAccount(addr crypto.Address) (*Account, error) {
	s.mu.RLock()
	if acc, ok := s.cache[addr]; ok {
		s.mu.RUnlock()
		return acc.Copy(), nil
	}
	s.mu.RUnlock()
	
	var account *Account
	err := s.db.View(func(txn *badger.Txn) error {
		key := makeAccountKey(addr)
		item, err := txn.Get(key)
		if err == badger.ErrKeyNotFound {
			account = NewAccount(addr)
			return nil
		}
		if err != nil {
			return err
		}
		
		return item.Value(func(val []byte) error {
			var err error
			account, err = DeserializeAccount(val)
			return err
		})
	})
	
	if err != nil {
		return nil, err
	}
	
	// Cache the account
	s.mu.Lock()
	s.cache[addr] = account.Copy()
	s.mu.Unlock()
	
	return account, nil
}

// SetAccount stores an account in the state
func (s *StateDB) SetAccount(acc *Account) error {
	s.mu.Lock()
	s.cache[acc.Address] = acc.Copy()
	s.mu.Unlock()
	
	return s.db.Update(func(txn *badger.Txn) error {
		key := makeAccountKey(acc.Address)
		data, err := acc.Serialize()
		if err != nil {
			return err
		}
		return txn.Set(key, data)
	})
}

// GetBalance retrieves the balance of an account
func (s *StateDB) GetBalance(addr crypto.Address) (uint64, error) {
	acc, err := s.GetAccount(addr)
	if err != nil {
		return 0, err
	}
	return acc.Balance, nil
}

// GetNonce retrieves the nonce of an account
func (s *StateDB) GetNonce(addr crypto.Address) (uint64, error) {
	acc, err := s.GetAccount(addr)
	if err != nil {
		return 0, err
	}
	return acc.Nonce, nil
}

// Transfer transfers balance from one account to another
func (s *StateDB) Transfer(from, to crypto.Address, amount uint64) error {
	fromAcc, err := s.GetAccount(from)
	if err != nil {
		return err
	}
	
	toAcc, err := s.GetAccount(to)
	if err != nil {
		return err
	}
	
	if err := fromAcc.SubBalance(amount); err != nil {
		return err
	}
	
	if err := toAcc.AddBalance(amount); err != nil {
		return err
	}
	
	if err := s.SetAccount(fromAcc); err != nil {
		return err
	}
	
	return s.SetAccount(toAcc)
}

// IncrementNonce increments the nonce of an account
func (s *StateDB) IncrementNonce(addr crypto.Address) error {
	acc, err := s.GetAccount(addr)
	if err != nil {
		return err
	}
	
	acc.IncrementNonce()
	return s.SetAccount(acc)
}

// GetStorage retrieves a storage value for a contract
func (s *StateDB) GetStorage(addr crypto.Address, key []byte) ([]byte, error) {
	var value []byte
	err := s.db.View(func(txn *badger.Txn) error {
		storageKey := makeStorageKey(addr, key)
		item, err := txn.Get(storageKey)
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		
		return item.Value(func(val []byte) error {
			value = append([]byte{}, val...)
			return nil
		})
	})
	
	return value, err
}

// SetStorage sets a storage value for a contract
func (s *StateDB) SetStorage(addr crypto.Address, key, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		storageKey := makeStorageKey(addr, key)
		return txn.Set(storageKey, value)
	})
}

// GetCode retrieves the code for a contract
func (s *StateDB) GetCode(codeHash crypto.Hash) ([]byte, error) {
	var code []byte
	err := s.db.View(func(txn *badger.Txn) error {
		key := makeCodeKey(codeHash)
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		
		return item.Value(func(val []byte) error {
			code = append([]byte{}, val...)
			return nil
		})
	})
	
	return code, err
}

// SetCode stores contract code
func (s *StateDB) SetCode(codeHash crypto.Hash, code []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		key := makeCodeKey(codeHash)
		return txn.Set(key, code)
	})
}

// ComputeStateRoot computes the state root hash
// Simple implementation: hash all account data
// Production should use Merkle Patricia Trie
func (s *StateDB) ComputeStateRoot() (crypto.Hash, error) {
	buf := new(bytes.Buffer)
	
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = accountPrefix
		it := txn.NewIterator(opts)
		defer it.Close()
		
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				buf.Write(val)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	
	if err != nil {
		return crypto.ZeroHash(), err
	}
	
	return crypto.HashData(buf.Bytes()), nil
}

// Snapshot creates a snapshot of the current state
func (s *StateDB) Snapshot() *StateSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	snapshot := &StateSnapshot{
		accounts: make(map[crypto.Address]*Account),
	}
	
	for addr, acc := range s.cache {
		snapshot.accounts[addr] = acc.Copy()
	}
	
	return snapshot
}

// Revert reverts the state to a snapshot
func (s *StateDB) Revert(snapshot *StateSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.cache = make(map[crypto.Address]*Account)
	for addr, acc := range snapshot.accounts {
		s.cache[addr] = acc.Copy()
	}
}

// Commit commits all cached changes to the database
func (s *StateDB) Commit() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	return s.db.Update(func(txn *badger.Txn) error {
		for _, acc := range s.cache {
			key := makeAccountKey(acc.Address)
			data, err := acc.Serialize()
			if err != nil {
				return err
			}
			if err := txn.Set(key, data); err != nil {
				return err
			}
		}
		return nil
	})
}

// StateSnapshot represents a snapshot of the state
type StateSnapshot struct {
	accounts map[crypto.Address]*Account
}

// Helper functions to create database keys
func makeAccountKey(addr crypto.Address) []byte {
	return append(accountPrefix, addr.Bytes()...)
}

func makeStorageKey(addr crypto.Address, key []byte) []byte {
	return append(append(storagePrefix, addr.Bytes()...), key...)
}

func makeCodeKey(hash crypto.Hash) []byte {
	return append(codePrefix, hash.Bytes()...)
}

func makeBlockKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return append(blockPrefix, buf...)
}

func makeTxKey(hash crypto.Hash) []byte {
	return append(txPrefix, hash.Bytes()...)
}

func makeReceiptKey(hash crypto.Hash) []byte {
	return append(receiptPrefix, hash.Bytes()...)
}

