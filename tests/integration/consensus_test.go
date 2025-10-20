package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/blockchain/layer1/internal/blockchain"
	"github.com/blockchain/layer1/internal/consensus"
	"github.com/blockchain/layer1/internal/crypto"
	"github.com/blockchain/layer1/internal/mempool"
	"github.com/blockchain/layer1/internal/p2p"
	"github.com/blockchain/layer1/internal/state"
	"github.com/blockchain/layer1/internal/types"
	"github.com/blockchain/layer1/internal/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNode represents a single node in the test network
type TestNode struct {
	ID         int
	PrivKey    *crypto.PrivateKey
	PubKey     *crypto.PublicKey
	Address    crypto.Address
	Blockchain *blockchain.Blockchain
	Consensus  *consensus.Engine
	Network    *p2p.Network
	Mempool    *mempool.Mempool
	StateDB    *state.StateDB
	VM         *vm.VM
}

func setupTestNode(t *testing.T, id int, validators []*consensus.Validator) *TestNode {
	privKey, pubKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	addr := pubKey.Address()

	// Create state DB (in-memory)
	stateDB, err := state.NewStateDB("", true)
	require.NoError(t, err)

	// Create VM
	vmInstance := vm.NewVM(stateDB)

	// Create mempool
	mp := mempool.NewMempool(stateDB)

	// Create validator set
	validatorSet := consensus.NewValidatorSet(validators)

	// Create blockchain
	bc := blockchain.NewBlockchain(stateDB, mp, validatorSet, vmInstance)

	// Create consensus engine
	consensusEngine := consensus.NewEngine(validatorSet, bc, privKey, pubKey)

	// Create P2P network (using in-memory transport for testing)
	network, err := p2p.NewNetwork(context.Background(), fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 30300+id), []string{})
	require.NoError(t, err)

	return &TestNode{
		ID:         id,
		PrivKey:    privKey,
		PubKey:     pubKey,
		Address:    addr,
		Blockchain: bc,
		Consensus:  consensusEngine,
		Network:    network,
		Mempool:    mp,
		StateDB:    stateDB,
		VM:         vmInstance,
	}
}

func TestSingleNodeBlockProduction(t *testing.T) {
	// Create single validator
	privKey, pubKey, _ := crypto.GenerateKey()
	validators := []*consensus.Validator{
		{
			Address:     pubKey.Address(),
			PubKey:      pubKey,
			VotingPower: 100,
			Jailed:      false,
		},
	}

	node := setupTestNode(t, 0, validators)
	defer node.StateDB.Close()

	// Fund an account for transactions
	senderPrivKey, senderPubKey, _ := crypto.GenerateKey()
	senderAddr := senderPubKey.Address()
	node.StateDB.SetAccount(&state.Account{
		Address: senderAddr,
		Balance: 1000000,
		Nonce:   0,
	})
	node.StateDB.Commit()

	// Create and add transaction to mempool
	tx := &types.Transaction{
		Type:     types.TxTypeTransfer,
		From:     senderAddr,
		To:       crypto.ZeroAddress(),
		Nonce:    0,
		Amount:   100,
		GasLimit: 21000,
		GasPrice: 1,
		Payload:  []byte{},
	}
	tx.Sign(senderPrivKey)
	node.Mempool.AddTx(tx)

	// Propose block
	block, err := node.Blockchain.ProposeBlock(pubKey.Address(), []byte("seed"))
	require.NoError(t, err)
	require.NotNil(t, block)

	// Sign block
	block.Sign(privKey)

	// Process block
	err = node.Blockchain.ProcessBlock(block)
	require.NoError(t, err)

	// Verify block was added
	assert.Equal(t, uint64(1), node.Blockchain.GetHeight())

	// Verify transaction was executed
	acc := node.StateDB.GetAccount(senderAddr)
	require.NotNil(t, acc)
	assert.Equal(t, uint64(1), acc.Nonce)
	assert.Less(t, acc.Balance, uint64(1000000)) // Balance reduced by amount + gas
}

func TestMultiNodeConsensus(t *testing.T) {
	t.Skip("Skipping multi-node test - requires full P2P integration")

	// This test would require:
	// 1. Setting up multiple nodes with P2P connections
	// 2. Implementing message passing between nodes
	// 3. Coordinating block proposals and votes
	// 4. Verifying consensus is reached

	// For a complete implementation, this would involve:
	// - Starting 4 validator nodes
	// - Connecting them via P2P
	// - Having proposer create blocks
	// - Other validators vote (prevote/precommit)
	// - Verifying all nodes reach same chain state
}

func TestConsensusVoting(t *testing.T) {
	// Create 4 validators
	validators := make([]*consensus.Validator, 4)
	privKeys := make([]*crypto.PrivateKey, 4)
	pubKeys := make([]*crypto.PublicKey, 4)

	for i := 0; i < 4; i++ {
		privKey, pubKey, _ := crypto.GenerateKey()
		privKeys[i] = privKey
		pubKeys[i] = pubKey
		validators[i] = &consensus.Validator{
			Address:     pubKey.Address(),
			PubKey:      pubKey,
			VotingPower: 100,
			Jailed:      false,
		}
	}

	validatorSet := consensus.NewValidatorSet(validators)

	// Create a test block
	stateDB, _ := state.NewStateDB("", true)
	defer stateDB.Close()

	block := &types.Block{
		Header: &types.BlockHeader{
			Height:         1,
			Timestamp:      uint64(time.Now().Unix()),
			PrevBlockHash:  crypto.ZeroHash(),
			StateRoot:      stateDB.ComputeStateRoot(),
			TxRoot:         crypto.ZeroHash(),
			ReceiptsRoot:   crypto.ZeroHash(),
			ProposerAddress: validators[0].Address,
			GasLimit:       10000000,
			GasUsed:        0,
		},
		Transactions: []*types.Transaction{},
		Receipts:     []*types.Receipt{},
	}
	block.Finalize()
	block.Sign(privKeys[0])

	// Create vote set
	voteSet := consensus.NewVoteSet(validatorSet)

	// Add prevotes from 3 validators (75% > 2/3)
	for i := 0; i < 3; i++ {
		voteSet.AddVote(validators[i].Address, block.Hash, validators[i].VotingPower)
	}

	// Should have 2/3+ majority
	assert.True(t, voteSet.HasTwoThirdsMajority(block.Hash))

	// Create precommit vote set
	precommitSet := consensus.NewVoteSet(validatorSet)

	// Add precommits from 3 validators
	for i := 0; i < 3; i++ {
		precommitSet.AddVote(validators[i].Address, block.Hash, validators[i].VotingPower)
	}

	// Should have 2/3+ majority for finalization
	assert.True(t, precommitSet.HasTwoThirdsMajority(block.Hash))
}

func TestSlashingOnDoubleSign(t *testing.T) {
	privKey, pubKey, _ := crypto.GenerateKey()
	validators := []*consensus.Validator{
		{
			Address:     pubKey.Address(),
			PubKey:      pubKey,
			VotingPower: 1000,
			Jailed:      false,
			SlashCount:  0,
		},
	}

	validatorSet := consensus.NewValidatorSet(validators)
	stateDB, _ := state.NewStateDB("", true)
	defer stateDB.Close()

	mp := mempool.NewMempool(stateDB)
	vmInstance := vm.NewVM(stateDB)
	bc := blockchain.NewBlockchain(stateDB, mp, validatorSet, vmInstance)
	engine := consensus.NewEngine(validatorSet, bc, privKey, pubKey)

	// Create two different blocks at same height (double-sign)
	block1 := &types.Block{
		Header: &types.BlockHeader{
			Height:          1,
			Timestamp:       uint64(time.Now().Unix()),
			PrevBlockHash:   crypto.ZeroHash(),
			StateRoot:       crypto.ZeroHash(),
			TxRoot:          crypto.ZeroHash(),
			ReceiptsRoot:    crypto.ZeroHash(),
			ProposerAddress: pubKey.Address(),
			GasLimit:        10000000,
			GasUsed:         0,
		},
		Transactions: []*types.Transaction{},
		Receipts:     []*types.Receipt{},
	}
	block1.Finalize()
	block1.Sign(privKey)

	block2 := &types.Block{
		Header: &types.BlockHeader{
			Height:          1,
			Timestamp:       uint64(time.Now().Unix()) + 1,
			PrevBlockHash:   crypto.ZeroHash(),
			StateRoot:       crypto.HashData([]byte("different")),
			TxRoot:          crypto.ZeroHash(),
			ReceiptsRoot:    crypto.ZeroHash(),
			ProposerAddress: pubKey.Address(),
			GasLimit:        10000000,
			GasUsed:         0,
		},
		Transactions: []*types.Transaction{},
		Receipts:     []*types.Receipt{},
	}
	block2.Finalize()
	block2.Sign(privKey)

	// Detect double-sign
	isDoubleSign := engine.DetectDoubleSign(block1, block2)
	assert.True(t, isDoubleSign)

	// Slash validator
	validatorSet.Slash(pubKey.Address(), consensus.SlashFractionDoubleSign)

	// Verify slashing
	val := validatorSet.GetValidator(pubKey.Address())
	require.NotNil(t, val)
	expectedPower := uint64(1000 * (1 - consensus.SlashFractionDoubleSign))
	assert.Equal(t, expectedPower, val.VotingPower)
	assert.Equal(t, 1, val.SlashCount)
}

func TestWASMContractDeployment(t *testing.T) {
	stateDB, err := state.NewStateDB("", true)
	require.NoError(t, err)
	defer stateDB.Close()

	vmInstance := vm.NewVM(stateDB)

	// Create deployer account
	deployerPrivKey, deployerPubKey, _ := crypto.GenerateKey()
	deployerAddr := deployerPubKey.Address()
	stateDB.SetAccount(&state.Account{
		Address: deployerAddr,
		Balance: 1000000,
		Nonce:   0,
	})
	stateDB.Commit()

	// Simple WASM contract (just returns)
	wasmCode := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // Version
	}

	// Deploy contract
	contractAddr, err := vmInstance.DeployContract(deployerAddr, wasmCode, []byte{}, 1000000)
	require.NoError(t, err)
	assert.NotEqual(t, crypto.ZeroAddress(), contractAddr)

	// Verify contract code stored
	storedCode := stateDB.GetCode(contractAddr)
	assert.Equal(t, wasmCode, storedCode)

	// Verify contract account created
	contractAcc := stateDB.GetAccount(contractAddr)
	require.NotNil(t, contractAcc)
	assert.True(t, contractAcc.IsContract)
}

func TestTransactionExecution(t *testing.T) {
	stateDB, err := state.NewStateDB("", true)
	require.NoError(t, err)
	defer stateDB.Close()

	// Create sender and receiver
	senderPrivKey, senderPubKey, _ := crypto.GenerateKey()
	_, receiverPubKey, _ := crypto.GenerateKey()
	senderAddr := senderPubKey.Address()
	receiverAddr := receiverPubKey.Address()

	// Fund sender
	stateDB.SetAccount(&state.Account{
		Address: senderAddr,
		Balance: 1000000,
		Nonce:   0,
	})
	stateDB.Commit()

	// Create transfer transaction
	tx := &types.Transaction{
		Type:     types.TxTypeTransfer,
		From:     senderAddr,
		To:       receiverAddr,
		Nonce:    0,
		Amount:   50000,
		GasLimit: 21000,
		GasPrice: 1,
		Payload:  []byte{},
	}
	tx.Sign(senderPrivKey)

	// Execute transaction
	mp := mempool.NewMempool(stateDB)
	vmInstance := vm.NewVM(stateDB)
	validators := []*consensus.Validator{
		{
			Address:     senderAddr,
			PubKey:      senderPubKey,
			VotingPower: 100,
		},
	}
	validatorSet := consensus.NewValidatorSet(validators)
	bc := blockchain.NewBlockchain(stateDB, mp, validatorSet, vmInstance)

	// Create block with transaction
	block := &types.Block{
		Header: &types.BlockHeader{
			Height:          1,
			Timestamp:       uint64(time.Now().Unix()),
			PrevBlockHash:   crypto.ZeroHash(),
			ProposerAddress: senderAddr,
			GasLimit:        10000000,
		},
		Transactions: []*types.Transaction{tx},
	}

	// Process block
	err = bc.ProcessBlock(block)
	require.NoError(t, err)

	// Verify sender balance decreased
	senderAcc := stateDB.GetAccount(senderAddr)
	require.NotNil(t, senderAcc)
	assert.Less(t, senderAcc.Balance, uint64(1000000))
	assert.Equal(t, uint64(1), senderAcc.Nonce)

	// Verify receiver balance increased
	receiverAcc := stateDB.GetAccount(receiverAddr)
	require.NotNil(t, receiverAcc)
	assert.Equal(t, uint64(50000), receiverAcc.Balance)
}

func BenchmarkBlockProduction(b *testing.B) {
	privKey, pubKey, _ := crypto.GenerateKey()
	validators := []*consensus.Validator{
		{
			Address:     pubKey.Address(),
			PubKey:      pubKey,
			VotingPower: 100,
		},
	}

	node := setupTestNode(b, 0, validators)
	defer node.StateDB.Close()

	// Add transactions to mempool
	for i := 0; i < 100; i++ {
		senderPrivKey, senderPubKey, _ := crypto.GenerateKey()
		senderAddr := senderPubKey.Address()
		node.StateDB.SetAccount(&state.Account{
			Address: senderAddr,
			Balance: 1000000,
			Nonce:   0,
		})

		tx := &types.Transaction{
			Type:     types.TxTypeTransfer,
			From:     senderAddr,
			To:       crypto.ZeroAddress(),
			Nonce:    0,
			Amount:   100,
			GasLimit: 21000,
			GasPrice: 1,
		}
		tx.Sign(senderPrivKey)
		node.Mempool.AddTx(tx)
	}
	node.StateDB.Commit()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node.Blockchain.ProposeBlock(pubKey.Address(), []byte("seed"))
	}
}

