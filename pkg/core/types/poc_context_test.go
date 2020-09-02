package types

import (
	"AQChainRe/pkg/common"
	"AQChainRe/pkg/ethdb"
	"AQChainRe/pkg/trie"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPocContextSnapshot(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	pocContext, err := NewPocContext(db)
	assert.Nil(t, err)

	snapshot := pocContext.Snapshot()
	assert.Equal(t, pocContext.Root(), snapshot.Root())
	assert.NotEqual(t, pocContext, snapshot)

	// change pocContext
	assert.Nil(t, pocContext.BecomeCandidate(common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")))
	assert.NotEqual(t, pocContext.Root(), snapshot.Root())

	// revert snapshot
	pocContext.RevertToSnapShot(snapshot)
	assert.Equal(t, pocContext.Root(), snapshot.Root())
	assert.NotEqual(t, pocContext, snapshot)
}

func TestPocContextBecomeCandidate(t *testing.T) {
	candidates := []common.Address{
		common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6e"),
		common.HexToAddress("0xa60a3886b552ff9992cfcd208ec1152079e046c2"),
		common.HexToAddress("0x4e080e49f62694554871e669aeb4ebe17c4a9670"),
	}
	db, _ := ethdb.NewMemDatabase()
	pocContext, err := NewPocContext(db)
	assert.Nil(t, err)
	for _, candidate := range candidates {
		assert.Nil(t, pocContext.BecomeCandidate(candidate))
	}

	candidateMap := map[common.Address]bool{}
	candidateIter := trie.NewIterator(pocContext.candidateTrie.NodeIterator(nil))
	for candidateIter.Next() {
		candidateMap[common.BytesToAddress(candidateIter.Value)] = true
	}
	assert.Equal(t, len(candidates), len(candidateMap))
	for _, candidate := range candidates {
		assert.True(t, candidateMap[candidate])
	}
}

func TestPocContextKickoutCandidate(t *testing.T) {
	candidates := []common.Address{
		common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6e"),
		common.HexToAddress("0xa60a3886b552ff9992cfcd208ec1152079e046c2"),
		common.HexToAddress("0x4e080e49f62694554871e669aeb4ebe17c4a9670"),
	}
	db, _ := ethdb.NewMemDatabase()
	pocContext, err := NewPocContext(db)
	assert.Nil(t, err)
	for _, candidate := range candidates {
		assert.Nil(t, pocContext.BecomeCandidate(candidate))
	}

	kickIdx := 1
	assert.Nil(t, pocContext.KickoutCandidate(candidates[kickIdx]))
	candidateMap := map[common.Address]bool{}
	candidateIter := trie.NewIterator(pocContext.candidateTrie.NodeIterator(nil))
	for candidateIter.Next() {
		candidateMap[common.BytesToAddress(candidateIter.Value)] = true
	}
}

func TestPocContextValidators(t *testing.T) {
	validators := []common.Address{
		common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6e"),
		common.HexToAddress("0xa60a3886b552ff9992cfcd208ec1152079e046c2"),
		common.HexToAddress("0x4e080e49f62694554871e669aeb4ebe17c4a9670"),
	}

	db, _ := ethdb.NewMemDatabase()
	pocContext, err := NewPocContext(db)
	assert.Nil(t, err)

	assert.Nil(t, pocContext.SetValidators(validators))

	result, err := pocContext.GetValidators()
	assert.Nil(t, err)
	assert.Equal(t, len(validators), len(result))
	validatorMap := map[common.Address]bool{}
	for _, validator := range validators {
		validatorMap[validator] = true
	}
	for _, validator := range result {
		assert.True(t, validatorMap[validator])
	}
}
