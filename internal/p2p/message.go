package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/blockchain/layer1/internal/types"
)

// MessageType represents the type of P2P message
type MessageType uint8

const (
	MsgTypeTx MessageType = iota
	MsgTypeBlock
	MsgTypeVote
	MsgTypeBlockRequest
	MsgTypeBlockResponse
	MsgTypeStatusRequest
	MsgTypeStatusResponse
)

// Message represents a P2P message
type Message struct {
	Type    MessageType `json:"type"`
	Payload []byte      `json:"payload"`
}

// NewMessage creates a new message
func NewMessage(msgType MessageType, payload []byte) *Message {
	return &Message{
		Type:    msgType,
		Payload: payload,
	}
}

// Serialize serializes the message to bytes
func (m *Message) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

// DeserializeMessage deserializes a message from bytes
func DeserializeMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to deserialize message: %w", err)
	}
	return &msg, nil
}

// TxMessage wraps a transaction for P2P transmission
type TxMessage struct {
	Tx *types.Transaction `json:"tx"`
}

// BlockMessage wraps a block for P2P transmission
type BlockMessage struct {
	Block *types.Block `json:"block"`
}

// VoteMessage wraps a vote for P2P transmission
type VoteMessage struct {
	Vote *types.Vote `json:"vote"`
}

// BlockRequestMessage requests a block by height
type BlockRequestMessage struct {
	Height uint64 `json:"height"`
}

// BlockResponseMessage responds with a block
type BlockResponseMessage struct {
	Block *types.Block `json:"block"`
}

// StatusMessage contains node status information
type StatusMessage struct {
	Height    uint64 `json:"height"`
	BestHash  string `json:"best_hash"`
	Validator bool   `json:"validator"`
}

// EncodeTxMessage encodes a transaction message
func EncodeTxMessage(tx *types.Transaction) ([]byte, error) {
	msg := &TxMessage{Tx: tx}
	return json.Marshal(msg)
}

// DecodeTxMessage decodes a transaction message
func DecodeTxMessage(data []byte) (*types.Transaction, error) {
	var msg TxMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return msg.Tx, nil
}

// EncodeBlockMessage encodes a block message
func EncodeBlockMessage(block *types.Block) ([]byte, error) {
	msg := &BlockMessage{Block: block}
	return json.Marshal(msg)
}

// DecodeBlockMessage decodes a block message
func DecodeBlockMessage(data []byte) (*types.Block, error) {
	var msg BlockMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return msg.Block, nil
}

// EncodeVoteMessage encodes a vote message
func EncodeVoteMessage(vote *types.Vote) ([]byte, error) {
	msg := &VoteMessage{Vote: vote}
	return json.Marshal(msg)
}

// DecodeVoteMessage decodes a vote message
func DecodeVoteMessage(data []byte) (*types.Vote, error) {
	var msg VoteMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return msg.Vote, nil
}

