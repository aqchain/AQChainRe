// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"AQChainRe/pkg/core/types"
	"fmt"
	"sort"
	"sync"

	"AQChainRe/pkg/common"
	"AQChainRe/pkg/log"
	"AQChainRe/pkg/rlp"
	"AQChainRe/pkg/trie"
)

// empty returns whether the account is considered empty.
func (s *stateObjectRecord) empty() bool {
	return false
	// return s.data.Origin == 0 && s.data.Balance.Sign() == 0 && s.data.Contribution.Sign() == 0 && bytes.Equal(s.data.CodeHash, emptyCodeHash)
}

// 同理statedb 用于数据的存储
type StateDBRecord struct {
	db   Database
	trie Trie

	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateObjects      map[common.Hash]*stateObjectRecord
	stateObjectsDirty map[common.Hash]struct{}

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	thash, bhash common.Hash
	txIndex      int
	logs         map[common.Hash][]*types.Log
	logSize      uint

	preimages map[common.Hash][]byte

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        journalRecord
	validRevisions []revision
	nextRevisionId int

	lock sync.Mutex
}

// Create a new state from a given trie
func NewRecord(root common.Hash, db Database) (*StateDBRecord, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}
	return &StateDBRecord{
		db:                db,
		trie:              tr,
		stateObjects:      make(map[common.Hash]*stateObjectRecord),
		stateObjectsDirty: make(map[common.Hash]struct{}),
		logs:              make(map[common.Hash][]*types.Log),
		preimages:         make(map[common.Hash][]byte),
	}, nil
}

// setError remembers the first non-nil error it is called with.
func (self *StateDBRecord) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *StateDBRecord) Error() error {
	return self.dbErr
}

// Reset clears out all emphemeral state objects from the state db, but keeps
// the underlying state trie to avoid reloading data for the next operations.
func (self *StateDBRecord) Reset(root common.Hash) error {
	tr, err := self.db.OpenTrie(root)
	if err != nil {
		return err
	}
	self.trie = tr
	self.stateObjects = make(map[common.Hash]*stateObjectRecord)
	self.stateObjectsDirty = make(map[common.Hash]struct{})
	self.thash = common.Hash{}
	self.bhash = common.Hash{}
	self.txIndex = 0
	self.logs = make(map[common.Hash][]*types.Log)
	self.logSize = 0
	self.preimages = make(map[common.Hash][]byte)
	self.clearJournalAndRefund()
	return nil
}

func (self *StateDBRecord) AddLog(log *types.Log) {
	self.journal = append(self.journal, addLogChangeRecord{txhash: self.thash})

	log.TxHash = self.thash
	log.BlockHash = self.bhash
	log.TxIndex = uint(self.txIndex)
	log.Index = self.logSize
	self.logs[self.thash] = append(self.logs[self.thash], log)
	self.logSize++
}

func (self *StateDBRecord) GetLogs(hash common.Hash) []*types.Log {
	return self.logs[hash]
}

func (self *StateDBRecord) Logs() []*types.Log {
	var logs []*types.Log
	for _, lgs := range self.logs {
		logs = append(logs, lgs...)
	}
	return logs
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (self *StateDBRecord) AddPreimage(hash common.Hash, preimage []byte) {
	if _, ok := self.preimages[hash]; !ok {
		self.journal = append(self.journal, addPreimageChangeRecord{hash: hash})
		pi := make([]byte, len(preimage))
		copy(pi, preimage)
		self.preimages[hash] = pi
	}
}

// Preimages returns a list of SHA3 preimages that have been submitted.
func (self *StateDBRecord) Preimages() map[common.Hash][]byte {
	return self.preimages
}

// Exist reports whether the given prev hash exists in the state.
// Notably this also returns true for suicided accounts.
func (self *StateDBRecord) Exist(addr common.Hash) bool {
	return self.getStateObject(addr) != nil
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (self *StateDBRecord) Empty(addr common.Hash) bool {
	so := self.getStateObject(addr)
	return so == nil || so.empty()
}

func (self *StateDBRecord) GetState(a common.Hash, b common.Hash) common.Hash {
	stateObject := self.getStateObject(a)
	if stateObject != nil {
		return stateObject.GetState(self.db, b)
	}
	return common.Hash{}
}

func (self *StateDBRecord) GetOrigin(addr common.Hash) common.Address {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Origin()
	}
	return common.Address{}
}

func (self *StateDBRecord) GetOwner(addr common.Hash) common.Address {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Owner()
	}
	return common.Address{}
}

func (self *StateDBRecord) GetStatus(addr common.Hash) uint8 {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Status()
	}
	return 100
}

// StorageTrie returns the storage trie of an prev.
// The return value is a copy and is nil for non-existent accounts.
func (self *StateDBRecord) StorageTrie(a common.Hash) Trie {
	stateObject := self.getStateObject(a)
	if stateObject == nil {
		return nil
	}
	cpy := stateObject.deepCopy(self, nil)
	return cpy.updateTrie(self.db)
}

func (self *StateDBRecord) HasSuicided(addr common.Hash) bool {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.suicided
	}
	return false
}

/*
 * SETTERS
 */

func (self *StateDBRecord) SetOrigin(addr common.Hash, account common.Address) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetOrigin(account)
	}
}

func (self *StateDBRecord) SetOwner(addr common.Hash, account common.Address) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetOrigin(account)
	}
}

func (self *StateDBRecord) SetState(addr common.Hash, key common.Hash, value common.Hash) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(self.db, key, value)
	}
}

// Suicide marks the given prev as suicided.
// This clears the prev balance.
//
// The prev's state object is still available until the state is committed,
// getStateObject will return a non-nil prev after Suicide.
/*func (self *StateDBRecord) Suicide(addr common.Hash) bool {
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		return false
	}
	self.journal = append(self.journal, suicideChangeRecord{
		record:     &addr,
		prev:        stateObject.suicided,
		prevbalance: new(big.Int).Set(stateObject.Balance()),
	})
	stateObject.markSuicided()
	stateObject.data.Balance = new(big.Int)

	return true
}*/

//
// Setting, updating & deleting state object methods
//

// updateStateObject writes the given object to the trie.
func (self *StateDBRecord) updateStateObject(stateObject *stateObjectRecord) {
	addr := stateObject.Hash()
	data, err := rlp.EncodeToBytes(stateObject)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	self.setError(self.trie.TryUpdate(addr[:], data))
}

// deleteStateObject removes the given object from the state trie.
func (self *StateDBRecord) deleteStateObject(stateObject *stateObjectRecord) {
	stateObject.deleted = true
	addr := stateObject.Hash()
	self.setError(self.trie.TryDelete(addr[:]))
}

// Retrieve a state object given my the hash. Returns nil if not found.
func (self *StateDBRecord) getStateObject(addr common.Hash) (stateObject *stateObjectRecord) {
	// Prefer 'live' objects.
	if obj := self.stateObjects[addr]; obj != nil {
		if obj.deleted {
			return nil
		}
		return obj
	}

	// Load the object from the database.
	enc, err := self.trie.TryGet(addr[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}
	var data Record
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newObjectRecord(self, addr, data, self.MarkStateObjectDirty)
	self.setStateObject(obj)
	return obj
}

func (self *StateDBRecord) setStateObject(object *stateObjectRecord) {
	self.stateObjects[object.Hash()] = object
}

// Retrieve a state object or create a new state object if nil
func (self *StateDBRecord) GetOrNewStateObject(addr common.Hash) *stateObjectRecord {
	stateObject := self.getStateObject(addr)
	if stateObject == nil || stateObject.deleted {
		stateObject, _ = self.createObject(addr)
	}
	return stateObject
}

// MarkStateObjectDirty adds the specified object to the dirty map to avoid costly
// state object cache iteration to find a handful of modified ones.
func (self *StateDBRecord) MarkStateObjectDirty(addr common.Hash) {
	self.stateObjectsDirty[addr] = struct{}{}
}

// createObject creates a new state object. If there is an existing account with
// the given hash, it is overwritten and returned as the second return value.
func (self *StateDBRecord) createObject(hash common.Hash) (newobj, prev *stateObjectRecord) {
	prev = self.getStateObject(hash)
	newobj = newObjectRecord(self, hash, Record{}, self.MarkStateObjectDirty)
	newobj.setStatus(0) // sets the object to dirty
	if prev == nil {
		self.journal = append(self.journal, createObjectChangeRecord{record: &hash})
	} else {
		self.journal = append(self.journal, resetObjectChangeRecord{prev: prev})
	}
	self.setStateObject(newobj)
	return newobj, prev
}

// CreateRecord explicitly creates a state object. If a state object with the hash
// already exists the balance is carried over to the new prev.
//
// CreateRecord is called during the EVM CREATE operation. The situation might arise that
// a contract does the following:
//
//   1. sends funds to sha(prev ++ (nonce + 1))
//   2. tx_create(sha(prev ++ nonce)) (note that this gets the hash of 1)
//
// Carrying over the balance ensures that Ether doesn't disappear.
func (self *StateDBRecord) CreateRecord(addr common.Hash) {
	new, prev := self.createObject(addr)
	if prev != nil {
		new.setOrigin(prev.data.Origin)
	}
}

func (db *StateDBRecord) ForEachStorage(addr common.Hash, cb func(key, value common.Hash) bool) {
	so := db.getStateObject(addr)
	if so == nil {
		return
	}

	// When iterating over the storage check the cache first
	for h, value := range so.cachedStorage {
		cb(h, value)
	}

	it := trie.NewIterator(so.getTrie(db.db).NodeIterator(nil))
	for it.Next() {
		// ignore cached values
		key := common.BytesToHash(db.trie.GetKey(it.Key))
		if _, ok := so.cachedStorage[key]; !ok {
			cb(key, common.BytesToHash(it.Value))
		}
	}
}

// Copy creates a deep, independent copy of the state.
// Snapshots of the copied state cannot be applied to the copy.
func (self *StateDBRecord) Copy() *StateDBRecord {
	self.lock.Lock()
	defer self.lock.Unlock()

	// Copy all the basic fields, initialize the memory ones
	state := &StateDBRecord{
		db:                self.db,
		trie:              self.trie,
		stateObjects:      make(map[common.Hash]*stateObjectRecord, len(self.stateObjectsDirty)),
		stateObjectsDirty: make(map[common.Hash]struct{}, len(self.stateObjectsDirty)),
		logs:              make(map[common.Hash][]*types.Log, len(self.logs)),
		logSize:           self.logSize,
		preimages:         make(map[common.Hash][]byte),
	}
	// Copy the dirty states, logs, and preimages
	for addr := range self.stateObjectsDirty {
		state.stateObjects[addr] = self.stateObjects[addr].deepCopy(state, state.MarkStateObjectDirty)
		state.stateObjectsDirty[addr] = struct{}{}
	}
	for hash, logs := range self.logs {
		state.logs[hash] = make([]*types.Log, len(logs))
		copy(state.logs[hash], logs)
	}
	for hash, preimage := range self.preimages {
		state.preimages[hash] = preimage
	}
	return state
}

// Snapshot returns an identifier for the current revision of the state.
func (self *StateDBRecord) Snapshot() int {
	id := self.nextRevisionId
	self.nextRevisionId++
	self.validRevisions = append(self.validRevisions, revision{id, len(self.journal)})
	return id
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (self *StateDBRecord) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(self.validRevisions), func(i int) bool {
		return self.validRevisions[i].id >= revid
	})
	if idx == len(self.validRevisions) || self.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := self.validRevisions[idx].journalIndex

	// Replay the journal to undo changes.
	for i := len(self.journal) - 1; i >= snapshot; i-- {
		self.journal[i].undo(self)
	}
	self.journal = self.journal[:snapshot]

	// Remove invalidated snapshots from the stack.
	self.validRevisions = self.validRevisions[:idx]
}

// Finalise finalises the state by removing the self destructed objects
// and clears the journal as well as the refunds.
func (s *StateDBRecord) Finalise(deleteEmptyObjects bool) {
	for addr := range s.stateObjectsDirty {
		stateObject := s.stateObjects[addr]
		if stateObject.suicided || (deleteEmptyObjects && stateObject.empty()) {
			s.deleteStateObject(stateObject)
		} else {
			stateObject.updateRoot(s.db)
			s.updateStateObject(stateObject)
		}
	}
	// Invalidate journal because reverting across transactions is not allowed.
	s.clearJournalAndRefund()
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (s *StateDBRecord) IntermediateRoot(deleteEmptyObjects bool) (h common.Hash) {
	s.Finalise(deleteEmptyObjects)
	return s.trie.Hash()
}

// Prepare sets the current transaction hash and index and block hash which is
// used when the EVM emits new state logs.
func (self *StateDBRecord) Prepare(thash, bhash common.Hash, ti int) {
	self.thash = thash
	self.bhash = bhash
	self.txIndex = ti
}

// DeleteSuicides flags the suicided objects for deletion so that it
// won't be referenced again when called / queried up on.
//
// DeleteSuicides should not be used for consensus related updates
// under any circumstances.
func (s *StateDBRecord) DeleteSuicides() {
	// Reset refund so that any used-gas calculations can use this method.
	s.clearJournalAndRefund()

	for addr := range s.stateObjectsDirty {
		stateObject := s.stateObjects[addr]

		// If the object has been removed by a suicide
		// flag the object as deleted.
		if stateObject.suicided {
			stateObject.deleted = true
		}
		delete(s.stateObjectsDirty, addr)
	}
}

func (s *StateDBRecord) clearJournalAndRefund() {
	s.journal = nil
	s.validRevisions = s.validRevisions[:0]
}

// CommitTo writes the state to the given database.
func (s *StateDBRecord) CommitTo(dbw trie.DatabaseWriter, deleteEmptyObjects bool) (root common.Hash, err error) {
	defer s.clearJournalAndRefund()

	// Commit objects to the trie.
	for addr, stateObject := range s.stateObjects {
		_, isDirty := s.stateObjectsDirty[addr]
		switch {
		case stateObject.suicided || (isDirty && deleteEmptyObjects && stateObject.empty()):
			// If the object has been removed, don't bother syncing it
			// and just mark it for deletion in the trie.
			s.deleteStateObject(stateObject)
		case isDirty:
			/*// Write any contract code associated with the state object
			if stateObject.code != nil && stateObject.dirtyCode {
				if err := dbw.Put(stateObject.CodeHash(), stateObject.code); err != nil {
					return common.Hash{}, err
				}
				stateObject.dirtyCode = false
			}*/
			// Write any storage changes in the state object to its storage trie.
			if err := stateObject.CommitTrie(s.db, dbw); err != nil {
				return common.Hash{}, err
			}
			// Update the object in the main prev trie.
			s.updateStateObject(stateObject)
		}
		delete(s.stateObjectsDirty, addr)
	}
	// Write trie changes.
	root, err = s.trie.CommitTo(dbw)
	log.Debug("Trie cache stats after commit", "misses", trie.CacheMisses(), "unloads", trie.CacheUnloads())
	return root, err
}