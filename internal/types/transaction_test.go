package types

import (
	"testing"

	"github.com/blockchain/layer1/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionSigningHash(t *testing.T) {
	tx := &Transaction{
		Type:     TxTypeTransfer,
		From:     crypto.ZeroAddress(),
		To:       crypto.ZeroAddress(),
		Nonce:    1,
		Amount:   100,
		GasLimit: 21000,
		GasPrice: 1,
		Payload:  []byte{},
	}

	hash1 := tx.SigningHash()
	hash2 := tx.SigningHash()

	// Same transaction should produce same signing hash
	assert.Equal(t, hash1, hash2)

	// Modifying transaction should change hash
	tx.Nonce = 2
	hash3 := tx.SigningHash()
	assert.NotEqual(t, hash1, hash3)
}

func TestTransactionSign(t *testing.T) {
	privKey, pubKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := &Transaction{
		Type:     TxTypeTransfer,
		From:     pubKey.Address(),
		To:       crypto.ZeroAddress(),
		Nonce:    1,
		Amount:   100,
		GasLimit: 21000,
		GasPrice: 1,
		Payload:  []byte{},
	}

	err = tx.Sign(privKey)
	require.NoError(t, err)

	// Signature should be set
	assert.NotNil(t, tx.Signature)
	assert.NotEmpty(t, tx.Hash)

	// Verify signature
	assert.True(t, pubKey.Verify(tx.SigningHash().Bytes(), tx.Signature))
}

func TestTransactionValidate(t *testing.T) {
	privKey, pubKey, _ := crypto.GenerateKey()

	tests := []struct {
		name    string
		tx      *Transaction
		wantErr bool
	}{
		{
			name: "valid transfer",
			tx: &Transaction{
				Type:     TxTypeTransfer,
				From:     pubKey.Address(),
				To:       crypto.ZeroAddress(),
				Nonce:    1,
				Amount:   100,
				GasLimit: 21000,
				GasPrice: 1,
				Payload:  []byte{},
			},
			wantErr: false,
		},
		{
			name: "zero gas limit",
			tx: &Transaction{
				Type:     TxTypeTransfer,
				From:     pubKey.Address(),
				To:       crypto.ZeroAddress(),
				Nonce:    1,
				Amount:   100,
				GasLimit: 0,
				GasPrice: 1,
				Payload:  []byte{},
			},
			wantErr: true,
		},
		{
			name: "zero gas price",
			tx: &Transaction{
				Type:     TxTypeTransfer,
				From:     pubKey.Address(),
				To:       crypto.ZeroAddress(),
				Nonce:    1,
				Amount:   100,
				GasLimit: 21000,
				GasPrice: 0,
				Payload:  []byte{},
			},
			wantErr: true,
		},
		{
			name: "zero from address",
			tx: &Transaction{
				Type:     TxTypeTransfer,
				From:     crypto.ZeroAddress(),
				To:       crypto.ZeroAddress(),
				Nonce:    1,
				Amount:   100,
				GasLimit: 21000,
				GasPrice: 1,
				Payload:  []byte{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				tt.tx.Sign(privKey)
			}
			err := tt.tx.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransactionCost(t *testing.T) {
	tx := &Transaction{
		Amount:   100,
		GasLimit: 21000,
		GasPrice: 2,
	}

	cost := tx.Cost()
	expected := uint64(100 + 21000*2) // amount + gas
	assert.Equal(t, expected, cost)
}

func TestTransactionCostOverflow(t *testing.T) {
	tx := &Transaction{
		Amount:   ^uint64(0), // max uint64
		GasLimit: 1,
		GasPrice: 1,
	}

	// Should handle overflow gracefully
	cost := tx.Cost()
	assert.Equal(t, ^uint64(0), cost)
}

func BenchmarkTransactionSign(b *testing.B) {
	privKey, pubKey, _ := crypto.GenerateKey()
	tx := &Transaction{
		Type:     TxTypeTransfer,
		From:     pubKey.Address(),
		To:       crypto.ZeroAddress(),
		Nonce:    1,
		Amount:   100,
		GasLimit: 21000,
		GasPrice: 1,
		Payload:  []byte{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx.Sign(privKey)
	}
}

func BenchmarkTransactionValidate(b *testing.B) {
	privKey, pubKey, _ := crypto.GenerateKey()
	tx := &Transaction{
		Type:     TxTypeTransfer,
		From:     pubKey.Address(),
		To:       crypto.ZeroAddress(),
		Nonce:    1,
		Amount:   100,
		GasLimit: 21000,
		GasPrice: 1,
		Payload:  []byte{},
	}
	tx.Sign(privKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx.Validate()
	}
}

