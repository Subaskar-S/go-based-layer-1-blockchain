package state

import (
	"encoding/json"
	"fmt"

	"github.com/blockchain/layer1/internal/crypto"
)

// Account represents a blockchain account
type Account struct {
	Address  crypto.Address `json:"address"`
	Nonce    uint64         `json:"nonce"`
	Balance  uint64         `json:"balance"`
	CodeHash crypto.Hash    `json:"code_hash,omitempty"` // For contract accounts
	IsContract bool         `json:"is_contract"`
}

// NewAccount creates a new account
func NewAccount(address crypto.Address) *Account {
	return &Account{
		Address:    address,
		Nonce:      0,
		Balance:    0,
		CodeHash:   crypto.ZeroHash(),
		IsContract: false,
	}
}

// Serialize serializes the account to bytes
func (a *Account) Serialize() ([]byte, error) {
	return json.Marshal(a)
}

// DeserializeAccount deserializes an account from bytes
func DeserializeAccount(data []byte) (*Account, error) {
	var acc Account
	if err := json.Unmarshal(data, &acc); err != nil {
		return nil, fmt.Errorf("failed to deserialize account: %w", err)
	}
	return &acc, nil
}

// Copy creates a deep copy of the account
func (a *Account) Copy() *Account {
	return &Account{
		Address:    a.Address,
		Nonce:      a.Nonce,
		Balance:    a.Balance,
		CodeHash:   a.CodeHash,
		IsContract: a.IsContract,
	}
}

// AddBalance adds to the account balance
func (a *Account) AddBalance(amount uint64) error {
	newBalance := a.Balance + amount
	if newBalance < a.Balance {
		return fmt.Errorf("balance overflow")
	}
	a.Balance = newBalance
	return nil
}

// SubBalance subtracts from the account balance
func (a *Account) SubBalance(amount uint64) error {
	if a.Balance < amount {
		return fmt.Errorf("insufficient balance: have %d, need %d", a.Balance, amount)
	}
	a.Balance -= amount
	return nil
}

// IncrementNonce increments the account nonce
func (a *Account) IncrementNonce() {
	a.Nonce++
}

// String returns a string representation of the account
func (a *Account) String() string {
	return fmt.Sprintf("Account{Addr:%s Nonce:%d Balance:%d Contract:%v}",
		a.Address.String()[:8], a.Nonce, a.Balance, a.IsContract)
}

