package state

import (
	"AQChainRe/pkg/common"
	"math/big"
)

type journalEntryRecord interface {
	undo(record *StateDBRecord)
}

type journalRecord []journalEntryRecord

type (
	// Changes to the Record trie.
	createObjectChangeRecord struct {
		record *common.Hash
	}
	resetObjectChangeRecord struct {
		prev *stateObjectRecord
	}
	suicideChangeRecord struct {
		record     *common.Hash
		prev        bool // whether account had already suicided
		prevbalance *big.Int
	}

	// Changes to individual accounts.
	storageChangeRecord struct {
		account       *common.Hash
		key, prevalue common.Hash
	}

	// Changes to other state values.
	addLogChangeRecord struct {
		txhash common.Hash
	}
	addPreimageChangeRecord struct {
		hash common.Hash
	}
	touchChangeRecord struct {
		account   *common.Hash
		prev      bool
		prevDirty bool
	}

	originChange struct{
		hash *common.Hash
		prev common.Address
	}

	ownerChange struct{
		hash *common.Hash
		prev common.Address
	}

	statusChange struct {
		hash *common.Hash
		prev    uint8
	}
)

func (ch statusChange) undo(s *StateDBRecord) {
	s.getStateObject(*ch.hash).setStatus(ch.prev)
}

func (ch originChange) undo(s *StateDBRecord) {
	s.getStateObject(*ch.hash).setOrigin(ch.prev)
}

func (ch ownerChange) undo(s *StateDBRecord) {
	s.getStateObject(*ch.hash).setOwner(ch.prev)
}

func (a addPreimageChangeRecord) undo(record *StateDBRecord) {
	panic("implement me")
}

func (a addLogChangeRecord) undo(record *StateDBRecord) {
	panic("implement me")
}

func (ch storageChangeRecord) undo(s *StateDBRecord) {
	panic("implement me")
}

func (t touchChangeRecord) undo(s *StateDBRecord) {
	panic("implement me")
}

func (ch createObjectChangeRecord) undo(s *StateDBRecord) {
	delete(s.stateObjects, *ch.record)
	delete(s.stateObjectsDirty, *ch.record)
}

func (ch resetObjectChangeRecord) undo(s *StateDBRecord) {
	s.setStateObject(ch.prev)
}

func (ch suicideChangeRecord) undo(s *StateDBRecord) {
	obj := s.getStateObject(*ch.record)
	if obj != nil {
		//obj.suicided = ch.prev
		//obj.setBalance(ch.prevbalance)
	}
}

