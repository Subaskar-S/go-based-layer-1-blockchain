package state

import (
	"testing"

	"github.com/blockchain/layer1/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateDBAccountOperations(t *testing.T) {
	db, err := NewStateDB("", true) // in-memory
	require.NoError(t, err)
	defer db.Close()

	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()

	// Get non-existent account
	acc := db.GetAccount(addr)
	assert.Nil(t, acc)

	// Create account
	newAcc := &Account{
		Address:    addr,
		Nonce:      0,
		Balance:    1000,
		CodeHash:   crypto.ZeroHash(),
		IsContract: false,
	}
	db.SetAccount(newAcc)

	// Get account
	acc = db.GetAccount(addr)
	require.NotNil(t, acc)
	assert.Equal(t, uint64(1000), acc.Balance)
	assert.Equal(t, uint64(0), acc.Nonce)

	// Update account
	acc.Balance = 2000
	acc.Nonce = 1
	db.SetAccount(acc)

	// Verify update
	acc = db.GetAccount(addr)
	assert.Equal(t, uint64(2000), acc.Balance)
	assert.Equal(t, uint64(1), acc.Nonce)
}

func TestStateDBTransfer(t *testing.T) {
	db, err := NewStateDB("", true)
	require.NoError(t, err)
	defer db.Close()

	_, pubKey1, _ := crypto.GenerateKey()
	_, pubKey2, _ := crypto.GenerateKey()
	addr1 := pubKey1.Address()
	addr2 := pubKey2.Address()

	// Create accounts
	db.SetAccount(&Account{Address: addr1, Balance: 1000})
	db.SetAccount(&Account{Address: addr2, Balance: 500})

	// Transfer
	err = db.Transfer(addr1, addr2, 300)
	require.NoError(t, err)

	// Verify balances
	acc1 := db.GetAccount(addr1)
	acc2 := db.GetAccount(addr2)
	assert.Equal(t, uint64(700), acc1.Balance)
	assert.Equal(t, uint64(800), acc2.Balance)
}

func TestStateDBTransferInsufficientBalance(t *testing.T) {
	db, err := NewStateDB("", true)
	require.NoError(t, err)
	defer db.Close()

	_, pubKey1, _ := crypto.GenerateKey()
	_, pubKey2, _ := crypto.GenerateKey()
	addr1 := pubKey1.Address()
	addr2 := pubKey2.Address()

	db.SetAccount(&Account{Address: addr1, Balance: 100})
	db.SetAccount(&Account{Address: addr2, Balance: 0})

	// Try to transfer more than balance
	err = db.Transfer(addr1, addr2, 200)
	assert.Error(t, err)

	// Balances should be unchanged
	acc1 := db.GetAccount(addr1)
	assert.Equal(t, uint64(100), acc1.Balance)
}

func TestStateDBStorage(t *testing.T) {
	db, err := NewStateDB("", true)
	require.NoError(t, err)
	defer db.Close()

	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()

	key := []byte("test_key")
	value := []byte("test_value")

	// Set storage
	db.SetStorage(addr, key, value)

	// Get storage
	retrieved := db.GetStorage(addr, key)
	assert.Equal(t, value, retrieved)

	// Update storage
	newValue := []byte("new_value")
	db.SetStorage(addr, key, newValue)
	retrieved = db.GetStorage(addr, key)
	assert.Equal(t, newValue, retrieved)
}

func TestStateDBCode(t *testing.T) {
	db, err := NewStateDB("", true)
	require.NoError(t, err)
	defer db.Close()

	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()

	code := []byte{0x00, 0x61, 0x73, 0x6d} // WASM magic number

	// Set code
	db.SetCode(addr, code)

	// Get code
	retrieved := db.GetCode(addr)
	assert.Equal(t, code, retrieved)

	// Verify account is marked as contract
	acc := db.GetAccount(addr)
	require.NotNil(t, acc)
	assert.True(t, acc.IsContract)
	assert.NotEqual(t, crypto.ZeroHash(), acc.CodeHash)
}

func TestStateDBSnapshot(t *testing.T) {
	db, err := NewStateDB("", true)
	require.NoError(t, err)
	defer db.Close()

	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()

	// Create account
	db.SetAccount(&Account{Address: addr, Balance: 1000})

	// Take snapshot
	snapshot := db.Snapshot()

	// Modify state
	acc := db.GetAccount(addr)
	acc.Balance = 2000
	db.SetAccount(acc)

	// Verify modification
	acc = db.GetAccount(addr)
	assert.Equal(t, uint64(2000), acc.Balance)

	// Revert to snapshot
	db.Revert(snapshot)

	// Verify revert
	acc = db.GetAccount(addr)
	assert.Equal(t, uint64(1000), acc.Balance)
}

func TestStateDBCommit(t *testing.T) {
	db, err := NewStateDB("", true)
	require.NoError(t, err)
	defer db.Close()

	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()

	// Create account
	db.SetAccount(&Account{Address: addr, Balance: 1000})

	// Commit
	err = db.Commit()
	require.NoError(t, err)

	// Clear cache to force read from disk
	db.accountCache = make(map[string]*Account)

	// Verify data persisted
	acc := db.GetAccount(addr)
	require.NotNil(t, acc)
	assert.Equal(t, uint64(1000), acc.Balance)
}

func TestStateDBComputeStateRoot(t *testing.T) {
	db, err := NewStateDB("", true)
	require.NoError(t, err)
	defer db.Close()

	// Empty state
	root1 := db.ComputeStateRoot()
	assert.NotEqual(t, crypto.ZeroHash(), root1)

	// Add account
	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()
	db.SetAccount(&Account{Address: addr, Balance: 1000})

	// State root should change
	root2 := db.ComputeStateRoot()
	assert.NotEqual(t, root1, root2)

	// Same state should produce same root
	root3 := db.ComputeStateRoot()
	assert.Equal(t, root2, root3)

	// Modify account
	acc := db.GetAccount(addr)
	acc.Balance = 2000
	db.SetAccount(acc)

	// State root should change again
	root4 := db.ComputeStateRoot()
	assert.NotEqual(t, root2, root4)
}

func TestStateDBMultipleSnapshots(t *testing.T) {
	db, err := NewStateDB("", true)
	require.NoError(t, err)
	defer db.Close()

	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()

	// Initial state
	db.SetAccount(&Account{Address: addr, Balance: 1000})
	snap1 := db.Snapshot()

	// Modify
	acc := db.GetAccount(addr)
	acc.Balance = 2000
	db.SetAccount(acc)
	snap2 := db.Snapshot()

	// Modify again
	acc = db.GetAccount(addr)
	acc.Balance = 3000
	db.SetAccount(acc)

	// Revert to snap2
	db.Revert(snap2)
	acc = db.GetAccount(addr)
	assert.Equal(t, uint64(2000), acc.Balance)

	// Revert to snap1
	db.Revert(snap1)
	acc = db.GetAccount(addr)
	assert.Equal(t, uint64(1000), acc.Balance)
}

func TestAccountAddBalance(t *testing.T) {
	acc := &Account{Balance: 1000}

	acc.AddBalance(500)
	assert.Equal(t, uint64(1500), acc.Balance)

	// Test overflow protection
	acc.Balance = ^uint64(0) - 100
	acc.AddBalance(200)
	assert.Equal(t, ^uint64(0), acc.Balance) // Should cap at max
}

func TestAccountSubBalance(t *testing.T) {
	acc := &Account{Balance: 1000}

	err := acc.SubBalance(500)
	require.NoError(t, err)
	assert.Equal(t, uint64(500), acc.Balance)

	// Test insufficient balance
	err = acc.SubBalance(1000)
	assert.Error(t, err)
	assert.Equal(t, uint64(500), acc.Balance) // Should be unchanged
}

func TestAccountIncrementNonce(t *testing.T) {
	acc := &Account{Nonce: 5}

	acc.IncrementNonce()
	assert.Equal(t, uint64(6), acc.Nonce)

	acc.IncrementNonce()
	assert.Equal(t, uint64(7), acc.Nonce)
}

func BenchmarkStateDBGetAccount(b *testing.B) {
	db, _ := NewStateDB("", true)
	defer db.Close()

	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()
	db.SetAccount(&Account{Address: addr, Balance: 1000})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.GetAccount(addr)
	}
}

func BenchmarkStateDBSetAccount(b *testing.B) {
	db, _ := NewStateDB("", true)
	defer db.Close()

	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()
	acc := &Account{Address: addr, Balance: 1000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.SetAccount(acc)
	}
}

func BenchmarkStateDBTransfer(b *testing.B) {
	db, _ := NewStateDB("", true)
	defer db.Close()

	_, pubKey1, _ := crypto.GenerateKey()
	_, pubKey2, _ := crypto.GenerateKey()
	addr1 := pubKey1.Address()
	addr2 := pubKey2.Address()

	db.SetAccount(&Account{Address: addr1, Balance: 1000000000})
	db.SetAccount(&Account{Address: addr2, Balance: 0})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Transfer(addr1, addr2, 1)
	}
}

func BenchmarkStateDBComputeStateRoot(b *testing.B) {
	db, _ := NewStateDB("", true)
	defer db.Close()

	// Add 100 accounts
	for i := 0; i < 100; i++ {
		_, pubKey, _ := crypto.GenerateKey()
		addr := pubKey.Address()
		db.SetAccount(&Account{Address: addr, Balance: uint64(i * 1000)})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.ComputeStateRoot()
	}
}

