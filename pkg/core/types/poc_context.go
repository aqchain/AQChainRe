package types

import (
	"AQChainRe/pkg/common"
	"AQChainRe/pkg/crypto/sha3"
	"AQChainRe/pkg/ethdb"
	"AQChainRe/pkg/log"
	"AQChainRe/pkg/rlp"
	"AQChainRe/pkg/trie"
	"fmt"
	"math/big"

)

type PocContext struct {
	epochTrie        *trie.Trie
	contributionTrie *trie.Trie
	latestTxTrie     *trie.Trie
	candidateTrie    *trie.Trie
	mintCntTrie      *trie.Trie

	db ethdb.Database
}

var (
	epochPrefix        = []byte("epoch-")
	contributionPrefix = []byte("contributionTrie-")
	latestTxPrefix     = []byte("latestTx-")
	candidatePrefix    = []byte("candidate-")
	mintCntPrefix      = []byte("mintCnt-")
)

func NewEpochTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, epochPrefix, db)
}

func NewContributionTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, contributionPrefix, db)
}

func NewLatestTxTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, latestTxPrefix, db)
}

func NewCandidateTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, candidatePrefix, db)
}

func NewMintCntTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, mintCntPrefix, db)
}

func NewPocContext(db ethdb.Database) (*PocContext, error) {
	epochTrie, err := NewEpochTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	contributionTrie, err := NewContributionTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	latestTx, err := NewLatestTxTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	candidateTrie, err := NewCandidateTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	mintCntTrie, err := NewMintCntTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	return &PocContext{
		epochTrie:        epochTrie,
		contributionTrie: contributionTrie,
		latestTxTrie:     latestTx,
		candidateTrie:    candidateTrie,
		mintCntTrie:      mintCntTrie,
		db:               db,
	}, nil
}

func NewPocContextFromProto(db ethdb.Database, ctxProto *PocContextProto) (*PocContext, error) {
	epochTrie, err := NewEpochTrie(ctxProto.EpochHash, db)
	if err != nil {
		return nil, err
	}
	contributionTrie, err := NewContributionTrie(ctxProto.ContributionHash, db)
	if err != nil {
		return nil, err
	}
	latestTxTrie, err := NewLatestTxTrie(ctxProto.LatestTxHash, db)
	if err != nil {
		return nil, err
	}
	candidateTrie, err := NewCandidateTrie(ctxProto.CandidateHash, db)
	if err != nil {
		return nil, err
	}
	mintCntTrie, err := NewMintCntTrie(ctxProto.MintCntHash, db)
	if err != nil {
		return nil, err
	}
	return &PocContext{
		epochTrie:        epochTrie,
		contributionTrie: contributionTrie,
		latestTxTrie:     latestTxTrie,
		candidateTrie:    candidateTrie,
		mintCntTrie:      mintCntTrie,
		db:               db,
	}, nil
}

func (pc *PocContext) Copy() *PocContext {
	epochTrie := *pc.epochTrie
	contributionTrie := *pc.contributionTrie
	latestTxTrie := *pc.latestTxTrie
	candidateTrie := *pc.candidateTrie
	mintCntTrie := *pc.mintCntTrie
	return &PocContext{
		epochTrie:        &epochTrie,
		contributionTrie: &contributionTrie,
		latestTxTrie:     &latestTxTrie,
		candidateTrie:    &candidateTrie,
		mintCntTrie:      &mintCntTrie,
	}
}

func (pc *PocContext) Root() (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, pc.epochTrie.Hash())
	rlp.Encode(hw, pc.contributionTrie.Hash())
	rlp.Encode(hw, pc.candidateTrie.Hash())
	rlp.Encode(hw, pc.latestTxTrie.Hash())
	rlp.Encode(hw, pc.mintCntTrie.Hash())
	hw.Sum(h[:0])
	return h
}

func (pc *PocContext) Snapshot() *PocContext {
	return pc.Copy()
}

func (pc *PocContext) RevertToSnapShot(snapshot *PocContext) {
	pc.epochTrie = snapshot.epochTrie
	pc.contributionTrie = snapshot.contributionTrie
	pc.candidateTrie = snapshot.candidateTrie
	pc.latestTxTrie = snapshot.latestTxTrie
	pc.mintCntTrie = snapshot.mintCntTrie
}

func (pc *PocContext) FromProto(dcp *PocContextProto) error {
	var err error
	pc.epochTrie, err = NewEpochTrie(dcp.EpochHash, pc.db)
	if err != nil {
		return err
	}
	pc.contributionTrie, err = NewContributionTrie(dcp.ContributionHash, pc.db)
	if err != nil {
		return err
	}
	pc.candidateTrie, err = NewCandidateTrie(dcp.CandidateHash, pc.db)
	if err != nil {
		return err
	}
	pc.latestTxTrie, err = NewLatestTxTrie(dcp.LatestTxHash, pc.db)
	if err != nil {
		return err
	}
	pc.mintCntTrie, err = NewMintCntTrie(dcp.MintCntHash, pc.db)
	return err
}

type PocContextProto struct {
	EpochHash        common.Hash `json:"epochRoot"        gencodec:"required"`
	ContributionHash common.Hash `json:"contributionRoot"     gencodec:"required"`
	CandidateHash    common.Hash `json:"candidateRoot"    gencodec:"required"`
	LatestTxHash     common.Hash `json:"latestTxRoot"         gencodec:"required"`
	MintCntHash      common.Hash `json:"mintCntRoot"      gencodec:"required"`
}

func (pc *PocContext) ToProto() *PocContextProto {
	return &PocContextProto{
		EpochHash:        pc.epochTrie.Hash(),
		ContributionHash: pc.contributionTrie.Hash(),
		CandidateHash:    pc.candidateTrie.Hash(),
		LatestTxHash:     pc.latestTxTrie.Hash(),
		MintCntHash:      pc.mintCntTrie.Hash(),
	}
}

func (p *PocContextProto) Root() (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, p.EpochHash)
	rlp.Encode(hw, p.ContributionHash)
	rlp.Encode(hw, p.CandidateHash)
	rlp.Encode(hw, p.LatestTxHash)
	rlp.Encode(hw, p.MintCntHash)
	hw.Sum(h[:0])
	return h
}

func (pc *PocContext) KickoutCandidate(candidateAddr common.Address) error {
	candidate := candidateAddr.Bytes()
	err := pc.candidateTrie.TryDelete(candidate)
	if err != nil {
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return err
		}
	}
	log.Info(" Kick out Candidate "+candidateAddr.String())
	return nil
}

func (pc *PocContext) BecomeCandidate(candidateAddr common.Address) error {
	candidate := candidateAddr.Bytes()
	err := pc.candidateTrie.TryUpdate(candidate, candidate)
	if err != nil {
		return err

	}
	log.Info(" Become Candidate "+candidateAddr.String())
	return nil
}

func (pc *PocContext) CommitTo(dbw trie.DatabaseWriter) (*PocContextProto, error) {
	epochRoot, err := pc.epochTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}
	contributionRoot, err := pc.contributionTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}
	latestTxRoot, err := pc.latestTxTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}
	candidateRoot, err := pc.candidateTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}
	mintCntRoot, err := pc.mintCntTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}
	return &PocContextProto{
		EpochHash:        epochRoot,
		ContributionHash: contributionRoot,
		LatestTxHash:     latestTxRoot,
		CandidateHash:    candidateRoot,
		MintCntHash:      mintCntRoot,
	}, nil
}

func (pc *PocContext) CandidateTrie() *trie.Trie               { return pc.candidateTrie }
func (pc *PocContext) ContributionTrie() *trie.Trie            { return pc.contributionTrie }
func (pc *PocContext) LatestTxTrie() *trie.Trie                { return pc.latestTxTrie }
func (pc *PocContext) EpochTrie() *trie.Trie                   { return pc.epochTrie }
func (pc *PocContext) MintCntTrie() *trie.Trie                 { return pc.mintCntTrie }
func (pc *PocContext) DB() ethdb.Database                      { return pc.db }
func (pc *PocContext) SetEpoch(epoch *trie.Trie)               { pc.epochTrie = epoch }
func (pc *PocContext) SetContribution(contribution *trie.Trie) { pc.contributionTrie = contribution }
func (pc *PocContext) SetLatestTx(latestTx *trie.Trie)         { pc.latestTxTrie = latestTx }
func (pc *PocContext) SetCandidate(candidate *trie.Trie)       { pc.candidateTrie = candidate }
func (pc *PocContext) SetMintCnt(mintCnt *trie.Trie)           { pc.mintCntTrie = mintCnt }

func (pc *PocContext) GetValidators() ([]common.Address, error) {
	var validators []common.Address
	key := []byte("validator")
	validatorsRLP := pc.epochTrie.Get(key)
	if err := rlp.DecodeBytes(validatorsRLP, &validators); err != nil {
		return nil, fmt.Errorf("failed to decode validators: %s", err)
	}
	return validators, nil
}

func (pc *PocContext) SetValidators(validators []common.Address) error {
	key := []byte("validator")
	validatorsRLP, err := rlp.EncodeToBytes(validators)
	if err != nil {
		return fmt.Errorf("failed to encode validators to rlp bytes: %s", err)
	}
	pc.epochTrie.Update(key, validatorsRLP)
	return nil
}

type AccountContribution struct {
	Account      common.Address
	Contribution *big.Int
}

type AccountLastedTx struct {
	Account      common.Address
	TxHash       common.Hash
	RecordTime   *big.Int
}

func (pc *PocContext) GetContributions() ([]AccountContribution, error) {
	var contributions []AccountContribution
	key := []byte("contribution")
	contributionsRLP := pc.contributionTrie.Get(key)
	if err := rlp.DecodeBytes(contributionsRLP, &contributions); err != nil {
		return contributions, fmt.Errorf("failed to decode contributions: %s", err)
	}
	return contributions, nil
}

func (pc *PocContext) SetContributions(contributions []AccountContribution) error {
	key := []byte("contribution")
	contributionsRLP, err := rlp.EncodeToBytes(contributions)
	if err != nil {
		return fmt.Errorf("failed to encode contributions to rlp bytes: %s", err)
	}
	pc.contributionTrie.Update(key, contributionsRLP)
	return nil
}

func (pc *PocContext) GetLastedTx(account common.Address) (AccountLastedTx, error) {
	var lastedTx AccountLastedTx
	txRLP := pc.latestTxTrie.Get(account.Bytes())
	if err := rlp.DecodeBytes(txRLP, &lastedTx); err != nil {
		return lastedTx, fmt.Errorf("failed to decode contributions: %s", err)
	}
	return lastedTx, nil
}

func (pc *PocContext) SetLastedTx(lastedTx AccountLastedTx) error {
	txRLP, err := rlp.EncodeToBytes(lastedTx)
	if err != nil {
		return fmt.Errorf("failed to encode Transaction to rlp bytes: %s", err)
	}
	pc.latestTxTrie.Update(lastedTx.Account.Bytes(), txRLP)
	return nil
}

func (pc *PocContext) GetCandidates() ([]common.Address, error) {
	var candidates []common.Address
	iter := trie.NewIterator(pc.candidateTrie.NodeIterator(nil))
	for iter.Next() {
		candidates = append(candidates, common.BytesToAddress(iter.Value))
	}
	return candidates, nil
}
