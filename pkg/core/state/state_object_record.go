package state

import (
	"AQChainRe/pkg/common"
	"AQChainRe/pkg/rlp"
	"AQChainRe/pkg/trie"
	"bytes"
	"fmt"
	"io"
)

type stateObjectRecord struct {
	hash common.Hash
	data Record
	db   *StateDBRecord

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// Write caches.
	trie        Trie // storage trie, which becomes non-nil on first access
	confirmTrie Trie

	cachedStorage Storage // Storage entry cache to avoid duplicate reads
	dirtyStorage  Storage // Storage entries that need to be flushed to disk

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	suicided bool
	touched  bool
	deleted  bool
	onDirty  func(addr common.Hash) // Callback method to mark a state object newly dirty
}

/*// empty returns whether the prev is considered empty.
func (s *stateObject) empty() bool {
	return s.data.Nonce == 0 && s.data.Balance.Sign() == 0 && s.data.Contribution.Sign() == 0 && bytes.Equal(s.data.CodeHash, emptyCodeHash)
}*/

// Record 记录保存的数据
type Record struct {
	Origin common.Address
	Owner  common.Address
	Txs    []common.Hash
	Status uint8

	Root common.Hash // merkle root of the storage trie
}

// newObject creates a state object.
func newObjectRecord(db *StateDBRecord, address common.Hash, data Record, onDirty func(addr common.Hash)) *stateObjectRecord {

	return &stateObjectRecord{
		db:            db,
		hash:          address,
		data:          data,
		cachedStorage: make(Storage),
		dirtyStorage:  make(Storage),
		onDirty:       onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (c *stateObjectRecord) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, c.data)
}

// setError remembers the first non-nil error it is called with.
func (self *stateObjectRecord) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *stateObjectRecord) markSuicided() {
	self.suicided = true
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

func (c *stateObjectRecord) touch() {
	c.db.journal = append(c.db.journal, touchChangeRecord{
		account:   &c.hash,
		prev:      c.touched,
		prevDirty: c.onDirty == nil,
	})
	if c.onDirty != nil {
		c.onDirty(c.Hash())
		c.onDirty = nil
	}
	c.touched = true
}

func (c *stateObjectRecord) getTrie(db Database) Trie {
	if c.trie == nil {
		var err error
		c.trie, err = db.OpenStorageTrie(c.hash, c.data.Root)
		if err != nil {
			c.trie, _ = db.OpenStorageTrie(c.hash, common.Hash{})
			c.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return c.trie
}

// GetState returns a value in prev storage.
func (self *stateObjectRecord) GetState(db Database, key common.Hash) common.Hash {
	value, exists := self.cachedStorage[key]
	if exists {
		return value
	}
	// Load from DB in case it is missing.
	enc, err := self.getTrie(db).TryGet(key[:])
	if err != nil {
		self.setError(err)
		return common.Hash{}
	}
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			self.setError(err)
		}
		value.SetBytes(content)
	}
	if (value != common.Hash{}) {
		self.cachedStorage[key] = value
	}
	return value
}

func (self *stateObjectRecord) Origin() common.Address {
	return self.data.Origin
}

func (self *stateObjectRecord) Owner() common.Address {
	return self.data.Owner
}

func (self *stateObjectRecord) Status() uint8 {
	return self.data.Status
}

func (self *stateObjectRecord) Txs() []common.Hash {
	return self.data.Txs
}

func (self *stateObjectRecord) SetOrigin(address common.Address) {
	self.db.journal = append(self.db.journal, originChange{
		hash: &self.hash,
		prev: self.data.Origin,
	})
	self.setOrigin(address)
}

func (self *stateObjectRecord) setOrigin(address common.Address) {
	self.data.Origin = address
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

func (self *stateObjectRecord) SetOwner(address common.Address) {
	self.db.journal = append(self.db.journal, ownerChange{
		hash: &self.hash,
		prev: self.data.Owner,
	})
	self.setOwner(address)
}

func (self *stateObjectRecord) setOwner(address common.Address) {
	self.data.Owner = address
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

func (self *stateObjectRecord) SetTxs(txs []common.Hash) {
	self.db.journal = append(self.db.journal, txsChange{
		hash: &self.hash,
		prev: self.data.Txs,
	})
	self.setTxs(txs)
}

func (self *stateObjectRecord) setTxs(txs []common.Hash) {
	self.data.Txs = txs
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

func (self *stateObjectRecord) SetStatus(status uint8) {
	self.db.journal = append(self.db.journal, statusChange{
		hash: &self.hash,
		prev: self.data.Status,
	})
	self.setStatus(status)
}

func (self *stateObjectRecord) setStatus(status uint8) {
	self.data.Status = status
	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

// SetState updates a value in prev storage.
func (self *stateObjectRecord) SetState(db Database, key, value common.Hash) {
	self.db.journal = append(self.db.journal, storageChangeRecord{
		account:  &self.hash,
		key:      key,
		prevalue: self.GetState(db, key),
	})
	self.setState(key, value)
}

func (self *stateObjectRecord) setState(key, value common.Hash) {
	self.cachedStorage[key] = value
	self.dirtyStorage[key] = value

	if self.onDirty != nil {
		self.onDirty(self.Hash())
		self.onDirty = nil
	}
}

// updateTrie writes cached storage modifications into the object's storage trie.
func (self *stateObjectRecord) updateTrie(db Database) Trie {
	tr := self.getTrie(db)
	for key, value := range self.dirtyStorage {
		delete(self.dirtyStorage, key)
		if (value == common.Hash{}) {
			self.setError(tr.TryDelete(key[:]))
			continue
		}
		// Encoding []byte cannot fail, ok to ignore the error.
		v, _ := rlp.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
		self.setError(tr.TryUpdate(key[:], v))
	}
	return tr
}

// UpdateRoot sets the trie root to the current root hash of
func (self *stateObjectRecord) updateRoot(db Database) {
	self.updateTrie(db)
	self.data.Root = self.trie.Hash()
}

// CommitTrie the storage trie of the object to dwb.
// This updates the trie root.
func (self *stateObjectRecord) CommitTrie(db Database, dbw trie.DatabaseWriter) error {
	self.updateTrie(db)
	if self.dbErr != nil {
		return self.dbErr
	}
	root, err := self.trie.CommitTo(dbw)
	if err == nil {
		self.data.Root = root
	}
	return err
}

func (self *stateObjectRecord) deepCopy(db *StateDBRecord, onDirty func(addr common.Hash)) *stateObjectRecord {
	stateObject := newObjectRecord(db, self.hash, self.data, onDirty)
	if self.trie != nil {
		stateObject.trie = db.db.CopyTrie(self.trie)
	}
	stateObject.dirtyStorage = self.dirtyStorage.Copy()
	stateObject.cachedStorage = self.dirtyStorage.Copy()
	stateObject.suicided = self.suicided
	stateObject.deleted = self.deleted
	return stateObject
}

//
// Attribute accessors
//

// Returns the hash of the contract/prev
func (c *stateObjectRecord) Hash() common.Hash {
	return c.hash
}
