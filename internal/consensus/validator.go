package consensus

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/blockchain/layer1/internal/crypto"
)

// Validator represents a validator in the network
type Validator struct {
	Address     crypto.Address `json:"address"`
	PubKey      []byte         `json:"pub_key"`
	VotingPower uint64         `json:"voting_power"` // Stake amount
	Jailed      bool           `json:"jailed"`
	SlashCount  uint32         `json:"slash_count"`
}

// ValidatorSet represents a set of validators
type ValidatorSet struct {
	Validators []*Validator `json:"validators"`
	TotalPower uint64       `json:"total_power"`
}

// NewValidator creates a new validator
func NewValidator(address crypto.Address, pubKey []byte, votingPower uint64) *Validator {
	return &Validator{
		Address:     address,
		PubKey:      pubKey,
		VotingPower: votingPower,
		Jailed:      false,
		SlashCount:  0,
	}
}

// NewValidatorSet creates a new validator set
func NewValidatorSet(validators []*Validator) *ValidatorSet {
	vs := &ValidatorSet{
		Validators: validators,
	}
	vs.computeTotalPower()
	vs.sort()
	return vs
}

// AddValidator adds a validator to the set
func (vs *ValidatorSet) AddValidator(val *Validator) {
	vs.Validators = append(vs.Validators, val)
	vs.computeTotalPower()
	vs.sort()
}

// RemoveValidator removes a validator from the set
func (vs *ValidatorSet) RemoveValidator(address crypto.Address) bool {
	for i, val := range vs.Validators {
		if val.Address == address {
			vs.Validators = append(vs.Validators[:i], vs.Validators[i+1:]...)
			vs.computeTotalPower()
			return true
		}
	}
	return false
}

// GetValidator retrieves a validator by address
func (vs *ValidatorSet) GetValidator(address crypto.Address) *Validator {
	for _, val := range vs.Validators {
		if val.Address == address {
			return val
		}
	}
	return nil
}

// UpdateVotingPower updates a validator's voting power
func (vs *ValidatorSet) UpdateVotingPower(address crypto.Address, power uint64) bool {
	val := vs.GetValidator(address)
	if val == nil {
		return false
	}
	val.VotingPower = power
	vs.computeTotalPower()
	vs.sort()
	return true
}

// GetProposer selects the proposer for a given height
// Uses deterministic round-robin weighted by stake
func (vs *ValidatorSet) GetProposer(height uint64, seed []byte) *Validator {
	if len(vs.Validators) == 0 {
		return nil
	}
	
	// Filter out jailed validators
	activeVals := make([]*Validator, 0)
	for _, val := range vs.Validators {
		if !val.Jailed {
			activeVals = append(activeVals, val)
		}
	}
	
	if len(activeVals) == 0 {
		return nil
	}
	
	// Deterministic selection based on height and seed
	// Weighted by voting power
	totalPower := uint64(0)
	for _, val := range activeVals {
		totalPower += val.VotingPower
	}
	
	// Compute selection value
	hash := crypto.HashData(append(seed, byte(height)))
	selectionValue := uint64(0)
	for i := 0; i < 8; i++ {
		selectionValue = (selectionValue << 8) | uint64(hash[i])
	}
	selectionValue = selectionValue % totalPower
	
	// Select validator
	cumulative := uint64(0)
	for _, val := range activeVals {
		cumulative += val.VotingPower
		if selectionValue < cumulative {
			return val
		}
	}
	
	return activeVals[0]
}

// HasTwoThirdsMajority checks if votes represent 2/3+ of total voting power
func (vs *ValidatorSet) HasTwoThirdsMajority(votingPower uint64) bool {
	return votingPower*3 > vs.TotalPower*2
}

// Jail jails a validator
func (vs *ValidatorSet) Jail(address crypto.Address) bool {
	val := vs.GetValidator(address)
	if val == nil {
		return false
	}
	val.Jailed = true
	return true
}

// Unjail unjails a validator
func (vs *ValidatorSet) Unjail(address crypto.Address) bool {
	val := vs.GetValidator(address)
	if val == nil {
		return false
	}
	val.Jailed = false
	return true
}

// Slash slashes a validator's voting power
func (vs *ValidatorSet) Slash(address crypto.Address, slashFraction float64) bool {
	val := vs.GetValidator(address)
	if val == nil {
		return false
	}
	
	slashAmount := uint64(float64(val.VotingPower) * slashFraction)
	if slashAmount > val.VotingPower {
		slashAmount = val.VotingPower
	}
	
	val.VotingPower -= slashAmount
	val.SlashCount++
	vs.computeTotalPower()
	
	return true
}

// Size returns the number of validators
func (vs *ValidatorSet) Size() int {
	return len(vs.Validators)
}

// ActiveSize returns the number of active (non-jailed) validators
func (vs *ValidatorSet) ActiveSize() int {
	count := 0
	for _, val := range vs.Validators {
		if !val.Jailed {
			count++
		}
	}
	return count
}

// Copy creates a deep copy of the validator set
func (vs *ValidatorSet) Copy() *ValidatorSet {
	validators := make([]*Validator, len(vs.Validators))
	for i, val := range vs.Validators {
		validators[i] = &Validator{
			Address:     val.Address,
			PubKey:      append([]byte{}, val.PubKey...),
			VotingPower: val.VotingPower,
			Jailed:      val.Jailed,
			SlashCount:  val.SlashCount,
		}
	}
	return &ValidatorSet{
		Validators: validators,
		TotalPower: vs.TotalPower,
	}
}

// Serialize serializes the validator set
func (vs *ValidatorSet) Serialize() ([]byte, error) {
	return json.Marshal(vs)
}

// DeserializeValidatorSet deserializes a validator set
func DeserializeValidatorSet(data []byte) (*ValidatorSet, error) {
	var vs ValidatorSet
	if err := json.Unmarshal(data, &vs); err != nil {
		return nil, fmt.Errorf("failed to deserialize validator set: %w", err)
	}
	return &vs, nil
}

// computeTotalPower computes the total voting power
func (vs *ValidatorSet) computeTotalPower() {
	total := uint64(0)
	for _, val := range vs.Validators {
		if !val.Jailed {
			total += val.VotingPower
		}
	}
	vs.TotalPower = total
}

// sort sorts validators by address for deterministic ordering
func (vs *ValidatorSet) sort() {
	sort.Slice(vs.Validators, func(i, j int) bool {
		return string(vs.Validators[i].Address.Bytes()) < string(vs.Validators[j].Address.Bytes())
	})
}

// Validate validates the validator set
func (vs *ValidatorSet) Validate() error {
	if len(vs.Validators) == 0 {
		return fmt.Errorf("validator set cannot be empty")
	}
	
	// Check for duplicates
	seen := make(map[crypto.Address]bool)
	for _, val := range vs.Validators {
		if seen[val.Address] {
			return fmt.Errorf("duplicate validator: %s", val.Address.String())
		}
		seen[val.Address] = true
		
		if val.VotingPower == 0 {
			return fmt.Errorf("validator %s has zero voting power", val.Address.String())
		}
	}
	
	return nil
}

