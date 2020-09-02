package poc

import (
	"AQChainRe/pkg/common"
	"AQChainRe/pkg/core/state"
	"AQChainRe/pkg/ethdb"
	"AQChainRe/pkg/trie"
	"AQChainRe/pkg/core/types"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupValidator(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	pocContext, _ := types.NewPocContext(db)
	mockEpochContext := &EpochContext{
		PocContext: pocContext,
	}
	validators := []common.Address{
		common.StringToAddress("addr1"),
		common.StringToAddress("addr2"),
		common.StringToAddress("addr3"),
	}
	mockEpochContext.PocContext.SetValidators(validators)
	for i, expected := range validators {
		got, _ := mockEpochContext.lookupValidator(int64(i) * blockInterval)
		if got != expected {
			t.Errorf("Failed to test lookup validator, %s was expected but got %s", expected.Str(), got.Str())
		}
	}
	_, err := mockEpochContext.lookupValidator(blockInterval - 1)
	if err != ErrInvalidMintBlockTime {
		t.Errorf("Failed to test lookup validator. err '%v' was expected but got '%v'", ErrInvalidMintBlockTime, err)
	}
}

func TestEpochContextKickoutValidator(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	stateDB, _ := state.New(common.Hash{}, state.NewDatabase(db))
	pocContext, err := types.NewPocContext(db)
	assert.Nil(t, err)
	epochContext := &EpochContext{
		TimeStamp:  epochInterval,
		PocContext: pocContext,
		stateDB:    stateDB,
	}
	atLeastMintCnt := epochInterval / blockInterval / maxValidatorSize / 2
	testEpoch := int64(1)

	// no validator can be kickout, because all validators mint enough block at least
	validators := []common.Address{}
	for i := 0; i < maxValidatorSize; i++ {
		validator := common.StringToAddress("addr" + strconv.Itoa(i))
		validators = append(validators, validator)
		assert.Nil(t, pocContext.BecomeCandidate(validator))
		setTestMintCnt(pocContext, testEpoch, validator, atLeastMintCnt)
	}
	assert.Nil(t, pocContext.SetValidators(validators))
	assert.Nil(t, pocContext.BecomeCandidate(common.StringToAddress("addr")))
	assert.Nil(t, epochContext.kickoutValidator(testEpoch))
	candidateMap := getCandidates(pocContext.CandidateTrie())
	assert.Equal(t, maxValidatorSize+1, len(candidateMap))

	// atLeast a safeSize count candidate will reserve
	pocContext, err = types.NewPocContext(db)
	assert.Nil(t, err)
	epochContext = &EpochContext{
		TimeStamp:  epochInterval,
		PocContext: pocContext,
		stateDB:    stateDB,
	}
	validators = []common.Address{}
	for i := 0; i < maxValidatorSize; i++ {
		validator := common.StringToAddress("addr" + strconv.Itoa(i))
		validators = append(validators, validator)
		assert.Nil(t, pocContext.BecomeCandidate(validator))
		setTestMintCnt(pocContext, testEpoch, validator, atLeastMintCnt-int64(i)-1)
	}
	assert.Nil(t, pocContext.SetValidators(validators))
	assert.Nil(t, epochContext.kickoutValidator(testEpoch))
	candidateMap = getCandidates(pocContext.CandidateTrie())
	assert.Equal(t, safeSize, len(candidateMap))
	for i := maxValidatorSize - 1; i >= safeSize; i-- {
		assert.False(t, candidateMap[common.StringToAddress("addr"+strconv.Itoa(i))])
	}

	// all validator will be kickout, because all validators didn't mint enough block at least
	pocContext, err = types.NewPocContext(db)
	assert.Nil(t, err)
	epochContext = &EpochContext{
		TimeStamp:  epochInterval,
		PocContext: pocContext,
		stateDB:    stateDB,
	}
	validators = []common.Address{}
	for i := 0; i < maxValidatorSize; i++ {
		validator := common.StringToAddress("addr" + strconv.Itoa(i))
		validators = append(validators, validator)
		assert.Nil(t, pocContext.BecomeCandidate(validator))
		setTestMintCnt(pocContext, testEpoch, validator, atLeastMintCnt-1)
	}
	for i := maxValidatorSize; i < maxValidatorSize*2; i++ {
		candidate := common.StringToAddress("addr" + strconv.Itoa(i))
		assert.Nil(t, pocContext.BecomeCandidate(candidate))
	}
	assert.Nil(t, pocContext.SetValidators(validators))
	assert.Nil(t, epochContext.kickoutValidator(testEpoch))
	candidateMap = getCandidates(pocContext.CandidateTrie())
	assert.Equal(t, maxValidatorSize, len(candidateMap))

	// only one validator mint count is not enough
	pocContext, err = types.NewPocContext(db)
	assert.Nil(t, err)
	epochContext = &EpochContext{
		TimeStamp:  epochInterval,
		PocContext: pocContext,
		stateDB:    stateDB,
	}
	validators = []common.Address{}
	for i := 0; i < maxValidatorSize; i++ {
		validator := common.StringToAddress("addr" + strconv.Itoa(i))
		validators = append(validators, validator)
		assert.Nil(t, pocContext.BecomeCandidate(validator))
		if i == 0 {
			setTestMintCnt(pocContext, testEpoch, validator, atLeastMintCnt-1)
		} else {
			setTestMintCnt(pocContext, testEpoch, validator, atLeastMintCnt)
		}
	}
	assert.Nil(t, pocContext.BecomeCandidate(common.StringToAddress("addr")))
	assert.Nil(t, pocContext.SetValidators(validators))
	assert.Nil(t, epochContext.kickoutValidator(testEpoch))
	candidateMap = getCandidates(pocContext.CandidateTrie())
	assert.Equal(t, maxValidatorSize, len(candidateMap))
	assert.False(t, candidateMap[common.StringToAddress("addr"+strconv.Itoa(0))])

	// epochTime is not complete, all validators mint enough block at least
	pocContext, err = types.NewPocContext(db)
	assert.Nil(t, err)
	epochContext = &EpochContext{
		TimeStamp:  epochInterval / 2,
		PocContext: pocContext,
		stateDB:    stateDB,
	}
	validators = []common.Address{}
	for i := 0; i < maxValidatorSize; i++ {
		validator := common.StringToAddress("addr" + strconv.Itoa(i))
		validators = append(validators, validator)
		assert.Nil(t, pocContext.BecomeCandidate(validator))
		setTestMintCnt(pocContext, testEpoch, validator, atLeastMintCnt/2)
	}
	for i := maxValidatorSize; i < maxValidatorSize*2; i++ {
		candidate := common.StringToAddress("addr" + strconv.Itoa(i))
		assert.Nil(t, pocContext.BecomeCandidate(candidate))
	}
	assert.Nil(t, pocContext.SetValidators(validators))
	assert.Nil(t, epochContext.kickoutValidator(testEpoch))
	candidateMap = getCandidates(pocContext.CandidateTrie())
	assert.Equal(t, maxValidatorSize*2, len(candidateMap))

	// epochTime is not complete, all validators didn't mint enough block at least
	pocContext, err = types.NewPocContext(db)
	assert.Nil(t, err)
	epochContext = &EpochContext{
		TimeStamp:  epochInterval / 2,
		PocContext: pocContext,
		stateDB:    stateDB,
	}
	validators = []common.Address{}
	for i := 0; i < maxValidatorSize; i++ {
		validator := common.StringToAddress("addr" + strconv.Itoa(i))
		validators = append(validators, validator)
		assert.Nil(t, pocContext.BecomeCandidate(validator))
		setTestMintCnt(pocContext, testEpoch, validator, atLeastMintCnt/2-1)
	}
	for i := maxValidatorSize; i < maxValidatorSize*2; i++ {
		candidate := common.StringToAddress("addr" + strconv.Itoa(i))
		assert.Nil(t, pocContext.BecomeCandidate(candidate))
	}
	assert.Nil(t, pocContext.SetValidators(validators))
	assert.Nil(t, epochContext.kickoutValidator(testEpoch))
	candidateMap = getCandidates(pocContext.CandidateTrie())
	assert.Equal(t, maxValidatorSize, len(candidateMap))

	pocContext, err = types.NewPocContext(db)
	assert.Nil(t, err)
	epochContext = &EpochContext{
		TimeStamp:  epochInterval / 2,
		PocContext: pocContext,
		stateDB:    stateDB,
	}
	assert.NotNil(t, epochContext.kickoutValidator(testEpoch))
	pocContext.SetValidators([]common.Address{})
	assert.NotNil(t, epochContext.kickoutValidator(testEpoch))
}

func setTestMintCnt(pocContext *types.PocContext, epoch int64, validator common.Address, count int64) {
	for i := int64(0); i < count; i++ {
		updateMintCnt(epoch*epochInterval, epoch*epochInterval+blockInterval, validator, pocContext)
	}
}

func getCandidates(candidateTrie *trie.Trie) map[common.Address]bool {
	candidateMap := map[common.Address]bool{}
	iter := trie.NewIterator(candidateTrie.NodeIterator(nil))
	for iter.Next() {
		candidateMap[common.BytesToAddress(iter.Value)] = true
	}
	return candidateMap
}

func TestEpochContextTryElect(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	stateDB, _ := state.New(common.Hash{}, state.NewDatabase(db))
	pocContext, err := types.NewPocContext(db)
	assert.Nil(t, err)
	epochContext := &EpochContext{
		TimeStamp:  epochInterval,
		PocContext: pocContext,
		stateDB:    stateDB,
	}
	atLeastMintCnt := epochInterval / blockInterval / maxValidatorSize / 2
	testEpoch := int64(1)
	validators := []common.Address{}
	for i := 0; i < maxValidatorSize; i++ {
		validator := common.StringToAddress("addr" + strconv.Itoa(i))
		validators = append(validators, validator)
		assert.Nil(t, pocContext.BecomeCandidate(validator))
		stateDB.SetBalance(validator, big.NewInt(1))
		stateDB.SetContribution(validator, big.NewInt(int64(1+i)))
		setTestMintCnt(pocContext, testEpoch, validator, atLeastMintCnt-1)
	}
	pocContext.BecomeCandidate(common.StringToAddress("more"))
	assert.Nil(t, pocContext.SetValidators(validators))

	// genesisEpoch == parentEpoch do not kickout
	genesis := &types.Header{
		Time: big.NewInt(0),
	}
	parent := &types.Header{
		Time: big.NewInt(epochInterval - blockInterval),
	}
	oldHash := pocContext.EpochTrie().Hash()
	assert.Nil(t, epochContext.tryElect(genesis, parent))
	result, err := pocContext.GetValidators()
	assert.Nil(t, err)
	assert.Equal(t, maxValidatorSize, len(result))
	for _, validator := range result {
		assert.True(t, strings.Contains(validator.Str(), "addr"))
	}
	assert.NotEqual(t, oldHash, pocContext.EpochTrie().Hash())
	conts, err := pocContext.GetContributions()
	for _,c := range conts{
		fmt.Println(c.Account.String())
		fmt.Println(c.Contribution)
	}
	assert.Nil(t, err)

	// genesisEpoch != parentEpoch and have none mintCnt do not kickout
	genesis = &types.Header{
		Time: big.NewInt(-epochInterval),
	}
	parent = &types.Header{
		Difficulty: big.NewInt(1),
		Time:       big.NewInt(epochInterval - blockInterval),
	}
	epochContext.TimeStamp = epochInterval
	oldHash = pocContext.EpochTrie().Hash()
	assert.Nil(t, epochContext.tryElect(genesis, parent))
	result, err = pocContext.GetValidators()
	assert.Nil(t, err)
	assert.Equal(t, maxValidatorSize, len(result))
	for _, validator := range result {
		assert.True(t, strings.Contains(validator.Str(), "addr"))
	}
	assert.NotEqual(t, oldHash, pocContext.EpochTrie().Hash())

	// genesisEpoch != parentEpoch kickout
	genesis = &types.Header{
		Time: big.NewInt(0),
	}
	parent = &types.Header{
		Time: big.NewInt(epochInterval*2 - blockInterval),
	}
	epochContext.TimeStamp = epochInterval * 2
	oldHash = pocContext.EpochTrie().Hash()
	assert.Nil(t, epochContext.tryElect(genesis, parent))
	result, err = pocContext.GetValidators()
	assert.Nil(t, err)
	assert.Equal(t, safeSize, len(result))
	moreCnt := 0
	for _, validator := range result {
		if strings.Contains(validator.Str(), "more") {
			moreCnt++
		}
	}
	assert.Equal(t, 1, moreCnt)
	assert.NotEqual(t, oldHash, pocContext.EpochTrie().Hash())

	// parentEpoch == currentEpoch do not elect
	genesis = &types.Header{
		Time: big.NewInt(0),
	}
	parent = &types.Header{
		Time: big.NewInt(epochInterval),
	}
	epochContext.TimeStamp = epochInterval + blockInterval
	oldHash = pocContext.EpochTrie().Hash()
	assert.Nil(t, epochContext.tryElect(genesis, parent))
	result, err = pocContext.GetValidators()
	assert.Nil(t, err)
	assert.Equal(t, safeSize, len(result))
	assert.Equal(t, oldHash, pocContext.EpochTrie().Hash())
}