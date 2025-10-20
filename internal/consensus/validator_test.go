package consensus

import (
	"testing"

	"github.com/blockchain/layer1/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatorSetGetProposer(t *testing.T) {
	// Create validator set
	validators := make([]*Validator, 3)
	for i := 0; i < 3; i++ {
		_, pubKey, _ := crypto.GenerateKey()
		validators[i] = &Validator{
			Address:     pubKey.Address(),
			PubKey:      pubKey,
			VotingPower: uint64((i + 1) * 100), // 100, 200, 300
			Jailed:      false,
		}
	}
	
	vs := NewValidatorSet(validators)
	
	// Get proposer for different heights
	proposer1 := vs.GetProposer(1, []byte("seed1"))
	proposer2 := vs.GetProposer(2, []byte("seed1"))
	
	// Proposers should be valid validators
	assert.NotNil(t, proposer1)
	assert.NotNil(t, proposer2)
	
	// Same height and seed should give same proposer
	proposer1b := vs.GetProposer(1, []byte("seed1"))
	assert.Equal(t, proposer1.Address, proposer1b.Address)
	
	// Different seed should potentially give different proposer
	proposer1c := vs.GetProposer(1, []byte("seed2"))
	assert.NotNil(t, proposer1c)
}

func TestValidatorSetProposerDistribution(t *testing.T) {
	// Create validator set with different voting powers
	validators := make([]*Validator, 3)
	for i := 0; i < 3; i++ {
		_, pubKey, _ := crypto.GenerateKey()
		validators[i] = &Validator{
			Address:     pubKey.Address(),
			PubKey:      pubKey,
			VotingPower: uint64((i + 1) * 100), // 100, 200, 300
			Jailed:      false,
		}
	}
	
	vs := NewValidatorSet(validators)
	
	// Count proposer selections over many heights
	counts := make(map[string]int)
	for height := uint64(1); height <= 1000; height++ {
		proposer := vs.GetProposer(height, []byte("seed"))
		counts[proposer.Address.String()]++
	}
	
	// Validator with more voting power should be selected more often
	// This is probabilistic, so we just check that all validators are selected
	assert.Equal(t, 3, len(counts))
	for _, count := range counts {
		assert.Greater(t, count, 0)
	}
}

func TestValidatorSetHasTwoThirdsMajority(t *testing.T) {
	validators := make([]*Validator, 4)
	for i := 0; i < 4; i++ {
		_, pubKey, _ := crypto.GenerateKey()
		validators[i] = &Validator{
			Address:     pubKey.Address(),
			PubKey:      pubKey,
			VotingPower: 100, // Equal voting power
			Jailed:      false,
		}
	}
	
	vs := NewValidatorSet(validators)
	
	// Total power = 400
	// 2/3 = 266.67
	
	assert.False(t, vs.HasTwoThirdsMajority(200)) // 50%
	assert.False(t, vs.HasTwoThirdsMajority(266)) // Just under 2/3
	assert.True(t, vs.HasTwoThirdsMajority(267))  // Just over 2/3
	assert.True(t, vs.HasTwoThirdsMajority(300))  // 75%
	assert.True(t, vs.HasTwoThirdsMajority(400))  // 100%
}

func TestValidatorSetGetValidator(t *testing.T) {
	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()
	
	validators := []*Validator{
		{
			Address:     addr,
			PubKey:      pubKey,
			VotingPower: 100,
			Jailed:      false,
		},
	}
	
	vs := NewValidatorSet(validators)
	
	// Get existing validator
	val := vs.GetValidator(addr)
	require.NotNil(t, val)
	assert.Equal(t, addr, val.Address)
	
	// Get non-existent validator
	_, otherPubKey, _ := crypto.GenerateKey()
	val = vs.GetValidator(otherPubKey.Address())
	assert.Nil(t, val)
}

func TestValidatorSetSlash(t *testing.T) {
	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()
	
	validators := []*Validator{
		{
			Address:     addr,
			PubKey:      pubKey,
			VotingPower: 1000,
			Jailed:      false,
			SlashCount:  0,
		},
	}
	
	vs := NewValidatorSet(validators)
	
	// Slash 10%
	vs.Slash(addr, 0.1)
	
	val := vs.GetValidator(addr)
	require.NotNil(t, val)
	assert.Equal(t, uint64(900), val.VotingPower) // 1000 - 10%
	assert.Equal(t, 1, val.SlashCount)
	
	// Slash again
	vs.Slash(addr, 0.2)
	val = vs.GetValidator(addr)
	assert.Equal(t, uint64(720), val.VotingPower) // 900 - 20%
	assert.Equal(t, 2, val.SlashCount)
}

func TestValidatorSetJail(t *testing.T) {
	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()
	
	validators := []*Validator{
		{
			Address:     addr,
			PubKey:      pubKey,
			VotingPower: 1000,
			Jailed:      false,
		},
	}
	
	vs := NewValidatorSet(validators)
	
	// Jail validator
	vs.Jail(addr)
	
	val := vs.GetValidator(addr)
	require.NotNil(t, val)
	assert.True(t, val.Jailed)
	
	// Unjail validator
	vs.Unjail(addr)
	val = vs.GetValidator(addr)
	assert.False(t, val.Jailed)
}

func TestValidatorSetJailedValidatorNotProposer(t *testing.T) {
	validators := make([]*Validator, 2)
	for i := 0; i < 2; i++ {
		_, pubKey, _ := crypto.GenerateKey()
		validators[i] = &Validator{
			Address:     pubKey.Address(),
			PubKey:      pubKey,
			VotingPower: 100,
			Jailed:      i == 0, // First validator is jailed
		}
	}
	
	vs := NewValidatorSet(validators)
	
	// Get proposer multiple times
	for height := uint64(1); height <= 100; height++ {
		proposer := vs.GetProposer(height, []byte("seed"))
		// Should never select jailed validator
		assert.False(t, proposer.Jailed)
		assert.Equal(t, validators[1].Address, proposer.Address)
	}
}

func TestVoteSetAddVote(t *testing.T) {
	_, pubKey, _ := crypto.GenerateKey()
	addr := pubKey.Address()
	
	validators := []*Validator{
		{
			Address:     addr,
			PubKey:      pubKey,
			VotingPower: 100,
			Jailed:      false,
		},
	}
	
	vs := NewValidatorSet(validators)
	voteSet := NewVoteSet(vs)
	
	blockHash := crypto.HashData([]byte("block"))
	
	// Add vote
	added := voteSet.AddVote(addr, blockHash, 100)
	assert.True(t, added)
	
	// Add duplicate vote (should be ignored)
	added = voteSet.AddVote(addr, blockHash, 100)
	assert.False(t, added)
	
	// Check voting power
	power := voteSet.GetVotingPower(blockHash)
	assert.Equal(t, uint64(100), power)
}

func TestVoteSetTwoThirdsMajority(t *testing.T) {
	validators := make([]*Validator, 4)
	addresses := make([]crypto.Address, 4)
	
	for i := 0; i < 4; i++ {
		_, pubKey, _ := crypto.GenerateKey()
		addresses[i] = pubKey.Address()
		validators[i] = &Validator{
			Address:     addresses[i],
			PubKey:      pubKey,
			VotingPower: 100,
			Jailed:      false,
		}
	}
	
	vs := NewValidatorSet(validators)
	voteSet := NewVoteSet(vs)
	
	blockHash := crypto.HashData([]byte("block"))
	
	// Add 2 votes (50%)
	voteSet.AddVote(addresses[0], blockHash, 100)
	voteSet.AddVote(addresses[1], blockHash, 100)
	assert.False(t, voteSet.HasTwoThirdsMajority(blockHash))
	
	// Add 3rd vote (75% > 2/3)
	voteSet.AddVote(addresses[2], blockHash, 100)
	assert.True(t, voteSet.HasTwoThirdsMajority(blockHash))
}

func BenchmarkGetProposer(b *testing.B) {
	validators := make([]*Validator, 100)
	for i := 0; i < 100; i++ {
		_, pubKey, _ := crypto.GenerateKey()
		validators[i] = &Validator{
			Address:     pubKey.Address(),
			PubKey:      pubKey,
			VotingPower: uint64(i + 1),
			Jailed:      false,
		}
	}
	
	vs := NewValidatorSet(validators)
	seed := []byte("benchmark")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.GetProposer(uint64(i), seed)
	}
}

