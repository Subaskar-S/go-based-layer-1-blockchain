package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blockchain/layer1/internal/crypto"
)

// BlockHeader represents the header of a block
type BlockHeader struct {
	Height          uint64         `json:"height"`
	Timestamp       int64          `json:"timestamp"`
	PrevBlockHash   crypto.Hash    `json:"prev_block_hash"`
	StateRoot       crypto.Hash    `json:"state_root"`
	TxRoot          crypto.Hash    `json:"tx_root"`
	ReceiptsRoot    crypto.Hash    `json:"receipts_root"`
	ProposerAddress crypto.Address `json:"proposer_address"`
	GasLimit        uint64         `json:"gas_limit"`
	GasUsed         uint64         `json:"gas_used"`
}

// Block represents a complete block
type Block struct {
	Header       *BlockHeader   `json:"header"`
	Transactions []*Transaction `json:"transactions"`
	Receipts     []*Receipt     `json:"receipts,omitempty"`
	Signature    crypto.Signature `json:"signature"`
	Hash         crypto.Hash    `json:"hash"`
}

// Vote represents a validator vote on a block
type Vote struct {
	Height          uint64           `json:"height"`
	Round           uint32           `json:"round"`
	BlockHash       crypto.Hash      `json:"block_hash"`
	VoteType        VoteType         `json:"vote_type"`
	ValidatorAddress crypto.Address  `json:"validator_address"`
	Timestamp       int64            `json:"timestamp"`
	Signature       crypto.Signature `json:"signature"`
}

// VoteType represents the type of vote
type VoteType uint8

const (
	VoteTypePrevote VoteType = iota
	VoteTypePrecommit
)

// NewBlockHeader creates a new block header
func NewBlockHeader(height uint64, prevHash, stateRoot crypto.Hash, proposer crypto.Address, gasLimit uint64) *BlockHeader {
	return &BlockHeader{
		Height:          height,
		Timestamp:       time.Now().Unix(),
		PrevBlockHash:   prevHash,
		StateRoot:       stateRoot,
		ProposerAddress: proposer,
		GasLimit:        gasLimit,
		GasUsed:         0,
	}
}

// NewBlock creates a new block
func NewBlock(header *BlockHeader, txs []*Transaction) *Block {
	return &Block{
		Header:       header,
		Transactions: txs,
		Receipts:     make([]*Receipt, 0),
	}
}

// Hash computes the hash of the block header
func (h *BlockHeader) Hash() crypto.Hash {
	buf := new(bytes.Buffer)
	
	binary.Write(buf, binary.BigEndian, h.Height)
	binary.Write(buf, binary.BigEndian, h.Timestamp)
	buf.Write(h.PrevBlockHash.Bytes())
	buf.Write(h.StateRoot.Bytes())
	buf.Write(h.TxRoot.Bytes())
	buf.Write(h.ReceiptsRoot.Bytes())
	buf.Write(h.ProposerAddress.Bytes())
	binary.Write(buf, binary.BigEndian, h.GasLimit)
	binary.Write(buf, binary.BigEndian, h.GasUsed)
	
	return crypto.HashData(buf.Bytes())
}

// ComputeTxRoot computes the merkle root of transactions
func (b *Block) ComputeTxRoot() crypto.Hash {
	if len(b.Transactions) == 0 {
		return crypto.ZeroHash()
	}
	
	// Simple implementation: hash all tx hashes together
	// Production should use proper Merkle tree
	buf := new(bytes.Buffer)
	for _, tx := range b.Transactions {
		buf.Write(tx.Hash.Bytes())
	}
	
	return crypto.HashData(buf.Bytes())
}

// ComputeReceiptsRoot computes the merkle root of receipts
func (b *Block) ComputeReceiptsRoot() crypto.Hash {
	if len(b.Receipts) == 0 {
		return crypto.ZeroHash()
	}
	
	buf := new(bytes.Buffer)
	for _, receipt := range b.Receipts {
		buf.Write(receipt.TxHash.Bytes())
		binary.Write(buf, binary.BigEndian, receipt.GasUsed)
		buf.WriteByte(receipt.Status)
	}
	
	return crypto.HashData(buf.Bytes())
}

// Finalize finalizes the block by computing roots and hash
func (b *Block) Finalize() {
	b.Header.TxRoot = b.ComputeTxRoot()
	b.Header.ReceiptsRoot = b.ComputeReceiptsRoot()
	b.Hash = b.Header.Hash()
}

// Sign signs the block with the proposer's private key
func (b *Block) Sign(privKey *crypto.PrivateKey) error {
	b.Finalize()
	b.Signature = privKey.Sign(b.Hash.Bytes())
	return nil
}

// VerifySignature verifies the block signature
func (b *Block) VerifySignature(pubKey *crypto.PublicKey) bool {
	return pubKey.Verify(b.Hash.Bytes(), b.Signature)
}

// Validate performs basic validation on the block
func (b *Block) Validate() error {
	if b.Header == nil {
		return fmt.Errorf("block header is nil")
	}
	
	if b.Header.Height == 0 && !b.Header.PrevBlockHash.IsZero() {
		return fmt.Errorf("genesis block must have zero prev hash")
	}
	
	if b.Header.Height > 0 && b.Header.PrevBlockHash.IsZero() {
		return fmt.Errorf("non-genesis block must have prev hash")
	}
	
	if b.Header.ProposerAddress.IsZero() {
		return fmt.Errorf("proposer address cannot be zero")
	}
	
	if b.Header.GasUsed > b.Header.GasLimit {
		return fmt.Errorf("gas used exceeds gas limit")
	}
	
	// Verify tx root
	computedTxRoot := b.ComputeTxRoot()
	if computedTxRoot != b.Header.TxRoot {
		return fmt.Errorf("tx root mismatch")
	}
	
	// Verify receipts root if receipts exist
	if len(b.Receipts) > 0 {
		computedReceiptsRoot := b.ComputeReceiptsRoot()
		if computedReceiptsRoot != b.Header.ReceiptsRoot {
			return fmt.Errorf("receipts root mismatch")
		}
	}
	
	// Verify hash
	computedHash := b.Header.Hash()
	if computedHash != b.Hash {
		return fmt.Errorf("block hash mismatch")
	}
	
	// Validate all transactions
	for i, tx := range b.Transactions {
		if err := tx.Validate(); err != nil {
			return fmt.Errorf("transaction %d invalid: %w", i, err)
		}
	}
	
	return nil
}

// Serialize serializes the block to bytes
func (b *Block) Serialize() ([]byte, error) {
	return json.Marshal(b)
}

// DeserializeBlock deserializes a block from bytes
func DeserializeBlock(data []byte) (*Block, error) {
	var block Block
	if err := json.Unmarshal(data, &block); err != nil {
		return nil, fmt.Errorf("failed to deserialize block: %w", err)
	}
	return &block, nil
}

// SigningHash returns the hash to be signed for a vote
func (v *Vote) SigningHash() crypto.Hash {
	buf := new(bytes.Buffer)
	
	binary.Write(buf, binary.BigEndian, v.Height)
	binary.Write(buf, binary.BigEndian, v.Round)
	buf.Write(v.BlockHash.Bytes())
	buf.WriteByte(byte(v.VoteType))
	buf.Write(v.ValidatorAddress.Bytes())
	binary.Write(buf, binary.BigEndian, v.Timestamp)
	
	return crypto.HashData(buf.Bytes())
}

// Sign signs the vote
func (v *Vote) Sign(privKey *crypto.PrivateKey) {
	sigHash := v.SigningHash()
	v.Signature = privKey.Sign(sigHash.Bytes())
}

// VerifySignature verifies the vote signature
func (v *Vote) VerifySignature(pubKey *crypto.PublicKey) bool {
	sigHash := v.SigningHash()
	return pubKey.Verify(sigHash.Bytes(), v.Signature)
}

// String returns a string representation of the block
func (b *Block) String() string {
	return fmt.Sprintf("Block{Height:%d Txs:%d Hash:%s Proposer:%s}",
		b.Header.Height, len(b.Transactions), b.Hash.String()[:8], b.Header.ProposerAddress.String()[:8])
}

