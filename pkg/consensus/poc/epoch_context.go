package poc

import (
	"AQChainRe/pkg/common"
	"AQChainRe/pkg/core/state"
	"AQChainRe/pkg/crypto"
	"AQChainRe/pkg/log"
	"AQChainRe/pkg/trie"
	"AQChainRe/pkg/core/types"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
)

type EpochContext struct {
	TimeStamp  int64
	PocContext *types.PocContext
	stateDB    *state.StateDB
}

// 获取申请者的贡献值
func (ec *EpochContext) countContributions() (ctb []types.AccountContribution, err error) {
	ctb = []types.AccountContribution{}
	candidateTrie := ec.PocContext.CandidateTrie()
	iterCandidate := trie.NewIterator(candidateTrie.NodeIterator(nil))
	existCandidate := iterCandidate.Next()

	if !existCandidate {
		return ctb, errors.New("no candidates")
	}
	for existCandidate {
		candidate := iterCandidate.Value
		candidateAddr := common.BytesToAddress(candidate)
		c := types.AccountContribution{
			Account:      candidateAddr,
			Contribution: ec.stateDB.GetContribution(candidateAddr),
		}
		ctb = append(ctb, c)
		// 获取贡献值之后应该再检查一下贡献值是否符合要求
		existCandidate = iterCandidate.Next()
	}

	return ctb, nil
}

func (ec *EpochContext) kickoutValidator(epoch int64) error {
	validators, err := ec.PocContext.GetValidators()
	if err != nil {
		return fmt.Errorf("failed to get validator: %s", err)
	}
	if len(validators) == 0 {
		return errors.New("no validator could be kickout")
	}

	epochDuration := epochInterval
	// First epoch duration may lt epoch interval,
	// while the first block time wouldn't always align with epoch interval,
	// so caculate the first epoch duartion with first block time instead of epoch interval,
	// prevent the validators were kickout incorrectly.
	if ec.TimeStamp-timeOfFirstBlock < epochInterval {
		epochDuration = ec.TimeStamp - timeOfFirstBlock
	}

	needKickoutValidators := sortableAddresses{}
	for _, validator := range validators {
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(epoch))
		key = append(key, validator.Bytes()...)
		cnt := int64(0)
		if cntBytes := ec.PocContext.MintCntTrie().Get(key); cntBytes != nil {
			cnt = int64(binary.BigEndian.Uint64(cntBytes))
		}
		if cnt < epochDuration/blockInterval/maxValidatorSize/2 {
			// not active validators need kickout
			needKickoutValidators = append(needKickoutValidators, &sortableAddress{validator, big.NewInt(cnt)})
		}
	}
	// no validators need kickout
	needKickoutValidatorCnt := len(needKickoutValidators)
	if needKickoutValidatorCnt <= 0 {
		return nil
	}
	sort.Sort(sort.Reverse(needKickoutValidators))

	candidateCount := 0
	iter := trie.NewIterator(ec.PocContext.CandidateTrie().NodeIterator(nil))
	for iter.Next() {
		candidateCount++
		if candidateCount >= needKickoutValidatorCnt+safeSize {
			break
		}
	}

	for i, validator := range needKickoutValidators {
		// ensure candidate count greater than or equal to safeSize
		if candidateCount <= safeSize {
			log.Info("No more candidate can be kickout", "prevEpochID", epoch, "candidateCount", candidateCount, "needKickoutCount", len(needKickoutValidators)-i)
			return nil
		}

		if err := ec.PocContext.KickoutCandidate(validator.address); err != nil {
			return err
		}
		// if kickout success, candidateCount minus 1
		candidateCount--
		log.Info("Kickout candidate", "prevEpochID", epoch, "candidate", validator.address.String(), "mintCnt", validator.weight.String())
	}
	return nil
}

func (ec *EpochContext) lookupValidator(now int64) (validator common.Address, err error) {
	validator = common.Address{}
	offset := now % epochInterval
	if offset%blockInterval != 0 {
		return common.Address{}, ErrInvalidMintBlockTime
	}
	offset /= blockInterval

	validators, err := ec.PocContext.GetValidators()
	if err != nil {
		return common.Address{}, err
	}
	validatorSize := len(validators)
	if validatorSize == 0 {
		return common.Address{}, errors.New("failed to lookup validator")
	}
	offset %= int64(validatorSize)
	return validators[offset], nil
}

func (ec *EpochContext) tryElect(genesis, parent *types.Header) error {
	// 根据当前块和上一块的时间计算当前块和上一块是否属于同一个周期，
	// 如果是同一个周期，意味着当前块不是周期的第一块，不需要触发选举
	// 如果不是同一周期，说明当前块是该周期的第一块，则触发选举
	genesisEpoch := genesis.Time.Int64() / epochInterval
	prevEpoch := parent.Time.Int64() / epochInterval
	currentEpoch := ec.TimeStamp / epochInterval

	prevEpochIsGenesis := prevEpoch == genesisEpoch
	if prevEpochIsGenesis && prevEpoch < currentEpoch {
		prevEpoch = currentEpoch - 1
	}

	prevEpochBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(prevEpochBytes, uint64(prevEpoch))
	iter := trie.NewIterator(ec.PocContext.MintCntTrie().PrefixIterator(prevEpochBytes))
	for i := prevEpoch; i < currentEpoch; i++ {
		// if prevEpoch is not genesis, kickout not active candidate
		if !prevEpochIsGenesis && iter.Next() {
			if err := ec.kickoutValidator(prevEpoch); err != nil {
				return err
			}
		}
		ctbs, err := ec.countContributions()
		if err != nil {
			return err
		}
		// 将贡献值作为排序权重 权重可能需要加入其他因素
		candidates := sortableAddresses{}
		for _, c := range ctbs {
			candidates = append(candidates, &sortableAddress{c.Account, c.Contribution})
		}
		if len(candidates) < safeSize {
			return errors.New("too few candidates")
		}
		sort.Sort(candidates)
		if len(candidates) > maxValidatorSize {
			candidates = candidates[:maxValidatorSize]
		}

		// shuffle candidates
		seed := int64(binary.LittleEndian.Uint32(crypto.Keccak512(parent.Hash().Bytes()))) + i
		r := rand.New(rand.NewSource(seed))
		for i := len(candidates) - 1; i > 0; i-- {
			j := int(r.Int31n(int32(i + 1)))
			candidates[i], candidates[j] = candidates[j], candidates[i]
		}
		sortedValidators := make([]common.Address, 0)
		for _, candidate := range candidates {
			sortedValidators = append(sortedValidators, candidate.address)
		}

		epochTrie, _ := types.NewEpochTrie(common.Hash{}, ec.PocContext.DB())
		ec.PocContext.SetEpoch(epochTrie)
		err = ec.PocContext.SetValidators(sortedValidators)
		if err != nil {
			return err
		}
		err = ec.PocContext.SetContributions(ctbs)
		if err != nil {
			return err
		}
		log.Info("Come to new epoch", "prevEpoch", i, "nextEpoch", i+1)
	}
	return nil
}

type sortableAddress struct {
	address common.Address
	weight  *big.Int
}
type sortableAddresses []*sortableAddress

func (p sortableAddresses) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p sortableAddresses) Len() int      { return len(p) }
func (p sortableAddresses) Less(i, j int) bool {
	if p[i].weight.Cmp(p[j].weight) < 0 {
		return false
	} else if p[i].weight.Cmp(p[j].weight) > 0 {
		return true
	} else {
		return p[i].address.String() < p[j].address.String()
	}
}
