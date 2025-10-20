package mempool

import (
	"testing"

	"github.com/blockchain/layer1/internal/crypto"
	"github.com/blockchain/layer1/internal/state"
	"github.com/blockchain/layer1/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestMempool(t *testing.T) (*Mempool, *state.StateDB) {
	db, err := state.NewStateDB("", true) // in-memory
	require.NoError(t, err)
	
	mp := NewMempool(db)
	return mp, db
}

func createTestTx(t *testing.T, from crypto.Address, nonce, amount, gasPrice uint64) *types.Transaction {
	privKey, pubKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := &types.Transaction{
		Type:     types.TxTypeTransfer,
		From:     pubKey.Address(),
		To:       crypto.ZeroAddress(),
		Nonce:    nonce,
		Amount:   amount,
		GasLimit: 21000,
		GasPrice: gasPrice,
		Payload:  []byte{},
	}
	
	err = tx.Sign(privKey)
	require.NoError(t, err)
	
	return tx
}

func TestMempoolAddTx(t *testing.T) {
	mp, db := setupTestMempool(t)
	
	privKey, pubKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	
	// Fund account
	acc := &state.Account{
		Address: pubKey.Address(),
		Balance: 1000000,
		Nonce:   0,
	}
	db.SetAccount(acc)
	db.Commit()
	
	tx := &types.Transaction{
		Type:     types.TxTypeTransfer,
		From:     pubKey.Address(),
		To:       crypto.ZeroAddress(),
		Nonce:    0,
		Amount:   100,
		GasLimit: 21000,
		GasPrice: 1,
		Payload:  []byte{},
	}
	tx.Sign(privKey)
	
	err = mp.AddTx(tx)
	assert.NoError(t, err)
	assert.Equal(t, 1, mp.Size())
}

func TestMempoolAddTxInsufficientBalance(t *testing.T) {
	mp, db := setupTestMempool(t)
	
	privKey, pubKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	
	// Fund account with insufficient balance
	acc := &state.Account{
		Address: pubKey.Address(),
		Balance: 100, // Not enough for amount + gas
		Nonce:   0,
	}
	db.SetAccount(acc)
	db.Commit()
	
	tx := &types.Transaction{
		Type:     types.TxTypeTransfer,
		From:     pubKey.Address(),
		To:       crypto.ZeroAddress(),
		Nonce:    0,
		Amount:   100,
		GasLimit: 21000,
		GasPrice: 1,
		Payload:  []byte{},
	}
	tx.Sign(privKey)
	
	err = mp.AddTx(tx)
	assert.Error(t, err)
	assert.Equal(t, 0, mp.Size())
}

func TestMempoolAddTxInvalidNonce(t *testing.T) {
	mp, db := setupTestMempool(t)
	
	privKey, pubKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	
	// Fund account
	acc := &state.Account{
		Address: pubKey.Address(),
		Balance: 1000000,
		Nonce:   5, // Current nonce is 5
	}
	db.SetAccount(acc)
	db.Commit()
	
	// Try to add tx with old nonce
	tx := &types.Transaction{
		Type:     types.TxTypeTransfer,
		From:     pubKey.Address(),
		To:       crypto.ZeroAddress(),
		Nonce:    3, // Old nonce
		Amount:   100,
		GasLimit: 21000,
		GasPrice: 1,
		Payload:  []byte{},
	}
	tx.Sign(privKey)
	
	err = mp.AddTx(tx)
	assert.Error(t, err)
}

func TestMempoolSelectTxs(t *testing.T) {
	mp, db := setupTestMempool(t)
	
	// Create multiple accounts and transactions
	for i := 0; i < 5; i++ {
		privKey, pubKey, _ := crypto.GenerateKey()
		
		acc := &state.Account{
			Address: pubKey.Address(),
			Balance: 1000000,
			Nonce:   0,
		}
		db.SetAccount(acc)
		
		tx := &types.Transaction{
			Type:     types.TxTypeTransfer,
			From:     pubKey.Address(),
			To:       crypto.ZeroAddress(),
			Nonce:    0,
			Amount:   100,
			GasLimit: 21000,
			GasPrice: uint64(i + 1), // Different gas prices
			Payload:  []byte{},
		}
		tx.Sign(privKey)
		mp.AddTx(tx)
	}
	db.Commit()
	
	// Select transactions
	txs := mp.SelectTxs(100000)
	
	// Should select transactions ordered by gas price (highest first)
	assert.True(t, len(txs) > 0)
	for i := 1; i < len(txs); i++ {
		assert.GreaterOrEqual(t, txs[i-1].GasPrice, txs[i].GasPrice)
	}
}

func TestMempoolRemoveTx(t *testing.T) {
	mp, db := setupTestMempool(t)
	
	privKey, pubKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	
	acc := &state.Account{
		Address: pubKey.Address(),
		Balance: 1000000,
		Nonce:   0,
	}
	db.SetAccount(acc)
	db.Commit()
	
	tx := &types.Transaction{
		Type:     types.TxTypeTransfer,
		From:     pubKey.Address(),
		To:       crypto.ZeroAddress(),
		Nonce:    0,
		Amount:   100,
		GasLimit: 21000,
		GasPrice: 1,
		Payload:  []byte{},
	}
	tx.Sign(privKey)
	
	mp.AddTx(tx)
	assert.Equal(t, 1, mp.Size())
	
	mp.RemoveTx(tx.Hash)
	assert.Equal(t, 0, mp.Size())
}

func TestMempoolEviction(t *testing.T) {
	mp, db := setupTestMempool(t)
	
	// Fill mempool beyond capacity
	for i := 0; i < MaxMempoolSize+10; i++ {
		privKey, pubKey, _ := crypto.GenerateKey()
		
		acc := &state.Account{
			Address: pubKey.Address(),
			Balance: 1000000,
			Nonce:   0,
		}
		db.SetAccount(acc)
		
		tx := &types.Transaction{
			Type:     types.TxTypeTransfer,
			From:     pubKey.Address(),
			To:       crypto.ZeroAddress(),
			Nonce:    0,
			Amount:   100,
			GasLimit: 21000,
			GasPrice: uint64(i + 1),
			Payload:  []byte{},
		}
		tx.Sign(privKey)
		mp.AddTx(tx)
	}
	db.Commit()
	
	// Mempool should not exceed max size
	assert.LessOrEqual(t, mp.Size(), MaxMempoolSize)
}

func BenchmarkMempoolAddTx(b *testing.B) {
	db, _ := state.NewStateDB("", true)
	mp := NewMempool(db)
	
	// Pre-create transactions
	txs := make([]*types.Transaction, b.N)
	for i := 0; i < b.N; i++ {
		privKey, pubKey, _ := crypto.GenerateKey()
		acc := &state.Account{
			Address: pubKey.Address(),
			Balance: 1000000,
			Nonce:   0,
		}
		db.SetAccount(acc)
		
		tx := &types.Transaction{
			Type:     types.TxTypeTransfer,
			From:     pubKey.Address(),
			To:       crypto.ZeroAddress(),
			Nonce:    0,
			Amount:   100,
			GasLimit: 21000,
			GasPrice: 1,
			Payload:  []byte{},
		}
		tx.Sign(privKey)
		txs[i] = tx
	}
	db.Commit()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mp.AddTx(txs[i])
	}
}

func BenchmarkMempoolSelectTxs(b *testing.B) {
	db, _ := state.NewStateDB("", true)
	mp := NewMempool(db)
	
	// Add 1000 transactions
	for i := 0; i < 1000; i++ {
		privKey, pubKey, _ := crypto.GenerateKey()
		acc := &state.Account{
			Address: pubKey.Address(),
			Balance: 1000000,
			Nonce:   0,
		}
		db.SetAccount(acc)
		
		tx := &types.Transaction{
			Type:     types.TxTypeTransfer,
			From:     pubKey.Address(),
			To:       crypto.ZeroAddress(),
			Nonce:    0,
			Amount:   100,
			GasLimit: 21000,
			GasPrice: uint64(i + 1),
			Payload:  []byte{},
		}
		tx.Sign(privKey)
		mp.AddTx(tx)
	}
	db.Commit()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mp.SelectTxs(1000000)
	}
}

