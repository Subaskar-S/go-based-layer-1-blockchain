package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/blockchain/layer1/internal/crypto"
)

// TxType represents the type of transaction
type TxType uint8

const (
	TxTypeTransfer TxType = iota
	TxTypeContractDeploy
	TxTypeContractCall
	TxTypeStake
	TxTypeUnstake
	TxTypeWithdrawRewards
)

// Transaction represents a blockchain transaction
type Transaction struct {
	Type      TxType          `json:"type"`
	From      crypto.Address  `json:"from"`
	To        crypto.Address  `json:"to"`
	Nonce     uint64          `json:"nonce"`
	Amount    uint64          `json:"amount"`
	GasLimit  uint64          `json:"gas_limit"`
	GasPrice  uint64          `json:"gas_price"`
	Payload   []byte          `json:"payload,omitempty"`
	Signature crypto.Signature `json:"signature"`
	Hash      crypto.Hash     `json:"hash"`
}

// NewTransaction creates a new unsigned transaction
func NewTransaction(txType TxType, from, to crypto.Address, nonce, amount, gasLimit, gasPrice uint64, payload []byte) *Transaction {
	tx := &Transaction{
		Type:     txType,
		From:     from,
		To:       to,
		Nonce:    nonce,
		Amount:   amount,
		GasLimit: gasLimit,
		GasPrice: gasPrice,
		Payload:  payload,
	}
	return tx
}

// SigningHash returns the hash to be signed (excludes signature and hash fields)
func (tx *Transaction) SigningHash() crypto.Hash {
	buf := new(bytes.Buffer)
	
	buf.WriteByte(byte(tx.Type))
	buf.Write(tx.From.Bytes())
	buf.Write(tx.To.Bytes())
	
	binary.Write(buf, binary.BigEndian, tx.Nonce)
	binary.Write(buf, binary.BigEndian, tx.Amount)
	binary.Write(buf, binary.BigEndian, tx.GasLimit)
	binary.Write(buf, binary.BigEndian, tx.GasPrice)
	
	binary.Write(buf, binary.BigEndian, uint32(len(tx.Payload)))
	buf.Write(tx.Payload)
	
	return crypto.HashData(buf.Bytes())
}

// Sign signs the transaction with the given private key
func (tx *Transaction) Sign(privKey *crypto.PrivateKey) error {
	sigHash := tx.SigningHash()
	tx.Signature = privKey.Sign(sigHash.Bytes())
	tx.Hash = tx.ComputeHash()
	return nil
}

// ComputeHash computes the full transaction hash (including signature)
func (tx *Transaction) ComputeHash() crypto.Hash {
	buf := new(bytes.Buffer)
	
	buf.Write(tx.SigningHash().Bytes())
	buf.Write(tx.Signature)
	
	return crypto.HashData(buf.Bytes())
}

// VerifySignature verifies the transaction signature
func (tx *Transaction) VerifySignature(pubKey *crypto.PublicKey) bool {
	sigHash := tx.SigningHash()
	return pubKey.Verify(sigHash.Bytes(), tx.Signature)
}

// Cost returns the total cost of the transaction (amount + gas fees)
func (tx *Transaction) Cost() uint64 {
	gasCost := tx.GasLimit * tx.GasPrice
	// Check for overflow
	if gasCost < tx.GasLimit || gasCost < tx.GasPrice {
		return ^uint64(0) // Return max uint64 on overflow
	}
	
	total := tx.Amount + gasCost
	if total < tx.Amount {
		return ^uint64(0) // Return max uint64 on overflow
	}
	
	return total
}

// Serialize serializes the transaction to bytes
func (tx *Transaction) Serialize() ([]byte, error) {
	return json.Marshal(tx)
}

// DeserializeTransaction deserializes a transaction from bytes
func DeserializeTransaction(data []byte) (*Transaction, error) {
	var tx Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %w", err)
	}
	return &tx, nil
}

// Validate performs basic validation on the transaction
func (tx *Transaction) Validate() error {
	// Check for zero address in from
	if tx.From.IsZero() {
		return fmt.Errorf("from address cannot be zero")
	}
	
	// Check gas limit
	if tx.GasLimit == 0 {
		return fmt.Errorf("gas limit must be greater than zero")
	}
	
	// Check gas price
	if tx.GasPrice == 0 {
		return fmt.Errorf("gas price must be greater than zero")
	}
	
	// Type-specific validation
	switch tx.Type {
	case TxTypeTransfer:
		if tx.To.IsZero() {
			return fmt.Errorf("transfer to address cannot be zero")
		}
		if tx.Amount == 0 {
			return fmt.Errorf("transfer amount must be greater than zero")
		}
	case TxTypeContractDeploy:
		if len(tx.Payload) == 0 {
			return fmt.Errorf("contract deploy must have payload")
		}
		// To address should be zero for deploy
		if !tx.To.IsZero() {
			return fmt.Errorf("contract deploy to address must be zero")
		}
	case TxTypeContractCall:
		if tx.To.IsZero() {
			return fmt.Errorf("contract call to address cannot be zero")
		}
	case TxTypeStake:
		if tx.Amount == 0 {
			return fmt.Errorf("stake amount must be greater than zero")
		}
	}
	
	// Check signature
	if len(tx.Signature) == 0 {
		return fmt.Errorf("transaction must be signed")
	}
	
	return nil
}

// String returns a string representation of the transaction
func (tx *Transaction) String() string {
	return fmt.Sprintf("Tx{Type:%d From:%s To:%s Nonce:%d Amount:%d Gas:%d/%d Hash:%s}",
		tx.Type, tx.From.String()[:8], tx.To.String()[:8], tx.Nonce, tx.Amount, tx.GasLimit, tx.GasPrice, tx.Hash.String()[:8])
}

// Receipt represents a transaction receipt
type Receipt struct {
	TxHash          crypto.Hash    `json:"tx_hash"`
	BlockHash       crypto.Hash    `json:"block_hash"`
	BlockHeight     uint64         `json:"block_height"`
	GasUsed         uint64         `json:"gas_used"`
	Status          uint8          `json:"status"` // 0 = failed, 1 = success
	ContractAddress crypto.Address `json:"contract_address,omitempty"`
	Logs            []Log          `json:"logs,omitempty"`
}

// Log represents an event log from contract execution
type Log struct {
	Address crypto.Address `json:"address"`
	Topics  []crypto.Hash  `json:"topics"`
	Data    []byte         `json:"data"`
}

