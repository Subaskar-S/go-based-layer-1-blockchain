package consensus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/blockchain/layer1/internal/crypto"
	"github.com/blockchain/layer1/internal/types"
)

const (
	// BlockTime is the target time between blocks
	BlockTime = 5 * time.Second
	// VoteTimeout is the timeout for collecting votes
	VoteTimeout = 3 * time.Second
	// SlashFractionDoubleSign is the fraction slashed for double signing
	SlashFractionDoubleSign = 0.05
	// SlashFractionDowntime is the fraction slashed for downtime
	SlashFractionDowntime = 0.01
)

// Engine implements the PoS consensus engine
type Engine struct {
	ctx    context.Context
	cancel context.CancelFunc
	
	validatorSet *ValidatorSet
	privKey      *crypto.PrivateKey
	pubKey       *crypto.PublicKey
	address      crypto.Address
	
	currentHeight uint64
	currentRound  uint32
	
	votes        map[uint64]map[crypto.Hash]*VoteSet // height -> blockHash -> votes
	votesMu      sync.RWMutex
	
	proposalCh   chan *types.Block
	voteCh       chan *types.Vote
	commitCh     chan *types.Block
	
	isValidator  bool
}

// VoteSet tracks votes for a block
type VoteSet struct {
	BlockHash   crypto.Hash
	Prevotes    map[crypto.Address]*types.Vote
	Precommits  map[crypto.Address]*types.Vote
	PrevotePower   uint64
	PrecommitPower uint64
}

// NewEngine creates a new consensus engine
func NewEngine(ctx context.Context, validatorSet *ValidatorSet, privKey *crypto.PrivateKey) *Engine {
	ctx, cancel := context.WithCancel(ctx)
	
	var pubKey *crypto.PublicKey
	var address crypto.Address
	var isValidator bool
	
	if privKey != nil {
		pubKey = privKey.Public()
		address = pubKey.Address()
		isValidator = validatorSet.GetValidator(address) != nil
	}
	
	return &Engine{
		ctx:          ctx,
		cancel:       cancel,
		validatorSet: validatorSet,
		privKey:      privKey,
		pubKey:       pubKey,
		address:      address,
		currentHeight: 0,
		currentRound: 0,
		votes:        make(map[uint64]map[crypto.Hash]*VoteSet),
		proposalCh:   make(chan *types.Block, 10),
		voteCh:       make(chan *types.Vote, 100),
		commitCh:     make(chan *types.Block, 10),
		isValidator:  isValidator,
	}
}

// Start starts the consensus engine
func (e *Engine) Start(startHeight uint64) {
	e.currentHeight = startHeight
	go e.run()
}

// Stop stops the consensus engine
func (e *Engine) Stop() {
	e.cancel()
}

// ProposeBlock proposes a new block
func (e *Engine) ProposeBlock(block *types.Block) error {
	select {
	case e.proposalCh <- block:
		return nil
	case <-e.ctx.Done():
		return fmt.Errorf("consensus engine stopped")
	}
}

// AddVote adds a vote to the consensus
func (e *Engine) AddVote(vote *types.Vote) error {
	select {
	case e.voteCh <- vote:
		return nil
	case <-e.ctx.Done():
		return fmt.Errorf("consensus engine stopped")
	}
}

// CommitChan returns the channel for committed blocks
func (e *Engine) CommitChan() <-chan *types.Block {
	return e.commitCh
}

// run is the main consensus loop
func (e *Engine) run() {
	ticker := time.NewTicker(BlockTime)
	defer ticker.Stop()
	
	for {
		select {
		case <-e.ctx.Done():
			return
			
		case <-ticker.C:
			// Check if we are the proposer
			if e.isValidator {
				proposer := e.validatorSet.GetProposer(e.currentHeight, []byte("seed"))
				if proposer != nil && proposer.Address == e.address {
					// We are the proposer - signal to create a block
					// This would be handled by the node layer
				}
			}
			
		case block := <-e.proposalCh:
			if err := e.handleProposal(block); err != nil {
				fmt.Printf("Error handling proposal: %v\n", err)
			}
			
		case vote := <-e.voteCh:
			if err := e.handleVote(vote); err != nil {
				fmt.Printf("Error handling vote: %v\n", err)
			}
		}
	}
}

// handleProposal handles a block proposal
func (e *Engine) handleProposal(block *types.Block) error {
	// Validate block
	if err := block.Validate(); err != nil {
		return fmt.Errorf("invalid block: %w", err)
	}
	
	// Verify proposer
	proposer := e.validatorSet.GetProposer(block.Header.Height, []byte("seed"))
	if proposer == nil {
		return fmt.Errorf("no proposer for height %d", block.Header.Height)
	}
	
	if proposer.Address != block.Header.ProposerAddress {
		return fmt.Errorf("invalid proposer")
	}
	
	// Verify block signature
	pubKey, err := crypto.NewPublicKeyFromBytes(proposer.PubKey)
	if err != nil {
		return fmt.Errorf("invalid proposer public key: %w", err)
	}
	
	if !block.VerifySignature(pubKey) {
		return fmt.Errorf("invalid block signature")
	}
	
	// If we are a validator, vote on the block
	if e.isValidator {
		e.voteOnBlock(block, types.VoteTypePrevote)
	}
	
	return nil
}

// handleVote handles a vote
func (e *Engine) handleVote(vote *types.Vote) error {
	// Validate vote
	validator := e.validatorSet.GetValidator(vote.ValidatorAddress)
	if validator == nil {
		return fmt.Errorf("unknown validator: %s", vote.ValidatorAddress.String())
	}
	
	if validator.Jailed {
		return fmt.Errorf("validator is jailed")
	}
	
	// Verify vote signature
	pubKey, err := crypto.NewPublicKeyFromBytes(validator.PubKey)
	if err != nil {
		return fmt.Errorf("invalid validator public key: %w", err)
	}
	
	if !vote.VerifySignature(pubKey) {
		return fmt.Errorf("invalid vote signature")
	}
	
	// Add vote to vote set
	e.votesMu.Lock()
	defer e.votesMu.Unlock()
	
	if _, exists := e.votes[vote.Height]; !exists {
		e.votes[vote.Height] = make(map[crypto.Hash]*VoteSet)
	}
	
	if _, exists := e.votes[vote.Height][vote.BlockHash]; !exists {
		e.votes[vote.Height][vote.BlockHash] = &VoteSet{
			BlockHash:  vote.BlockHash,
			Prevotes:   make(map[crypto.Address]*types.Vote),
			Precommits: make(map[crypto.Address]*types.Vote),
		}
	}
	
	voteSet := e.votes[vote.Height][vote.BlockHash]
	
	switch vote.VoteType {
	case types.VoteTypePrevote:
		if _, exists := voteSet.Prevotes[vote.ValidatorAddress]; !exists {
			voteSet.Prevotes[vote.ValidatorAddress] = vote
			voteSet.PrevotePower += validator.VotingPower
		}
		
		// Check if we have 2/3+ prevotes
		if e.validatorSet.HasTwoThirdsMajority(voteSet.PrevotePower) {
			// Move to precommit
			if e.isValidator {
				// Find the block for this hash (would be stored elsewhere)
				// For now, we'll signal precommit
				e.voteOnBlockHash(vote.BlockHash, vote.Height, types.VoteTypePrecommit)
			}
		}
		
	case types.VoteTypePrecommit:
		if _, exists := voteSet.Precommits[vote.ValidatorAddress]; !exists {
			voteSet.Precommits[vote.ValidatorAddress] = vote
			voteSet.PrecommitPower += validator.VotingPower
		}
		
		// Check if we have 2/3+ precommits - commit the block
		if e.validatorSet.HasTwoThirdsMajority(voteSet.PrecommitPower) {
			// Signal commit (would retrieve the actual block)
			// For now, we just advance height
			e.currentHeight = vote.Height + 1
			e.currentRound = 0
		}
	}
	
	return nil
}

// voteOnBlock creates and broadcasts a vote for a block
func (e *Engine) voteOnBlock(block *types.Block, voteType types.VoteType) {
	vote := &types.Vote{
		Height:           block.Header.Height,
		Round:            e.currentRound,
		BlockHash:        block.Hash,
		VoteType:         voteType,
		ValidatorAddress: e.address,
		Timestamp:        time.Now().Unix(),
	}
	
	vote.Sign(e.privKey)
	
	// Broadcast vote (would be done via P2P)
	e.voteCh <- vote
}

// voteOnBlockHash creates and broadcasts a vote for a block hash
func (e *Engine) voteOnBlockHash(blockHash crypto.Hash, height uint64, voteType types.VoteType) {
	vote := &types.Vote{
		Height:           height,
		Round:            e.currentRound,
		BlockHash:        blockHash,
		VoteType:         voteType,
		ValidatorAddress: e.address,
		Timestamp:        time.Now().Unix(),
	}
	
	vote.Sign(e.privKey)
	
	// Broadcast vote
	e.voteCh <- vote
}

// DetectDoubleSign detects if a validator has double-signed
func (e *Engine) DetectDoubleSign(vote1, vote2 *types.Vote) bool {
	return vote1.Height == vote2.Height &&
		vote1.Round == vote2.Round &&
		vote1.VoteType == vote2.VoteType &&
		vote1.ValidatorAddress == vote2.ValidatorAddress &&
		vote1.BlockHash != vote2.BlockHash
}

// SlashDoubleSign slashes a validator for double signing
func (e *Engine) SlashDoubleSign(address crypto.Address) {
	e.validatorSet.Slash(address, SlashFractionDoubleSign)
	e.validatorSet.Jail(address)
}

