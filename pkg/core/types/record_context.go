package types

import (
	"AQChainRe/pkg/common"
	"AQChainRe/pkg/crypto/sha3"
	"AQChainRe/pkg/ethdb"
	"AQChainRe/pkg/log"
	"AQChainRe/pkg/rlp"
	"AQChainRe/pkg/trie"
	"io"
	"math/big"
)

type Record struct {
	Time        *big.Int
	from        common.Address
	to          common.Address
	value       *big.Int
}

func (r *Record) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, r)
}

// 记录
type RecordContext struct {
	// 确权记录
	confirmationTrie *trie.Trie
	// 授权记录
	authorizationTrie *trie.Trie
	// 转移记录
	transferTrie     *trie.Trie

	db ethdb.Database
}

var (
	confirmationPrefix  = []byte("confirmation-")
	authorizationPrefix = []byte("authorization-")
	transferPrefix      = []byte("transfer-")
)

func NewConfirmationTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, confirmationPrefix, db)
}

func NewAuthorizationTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, authorizationPrefix, db)
}

func NewTransferTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, transferPrefix, db)
}

func NewRecordContext(db ethdb.Database) (*RecordContext, error) {
	confirmationTrie, err := NewConfirmationTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	authorizationTrie, err := NewAuthorizationTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	transferTrie, err := NewTransferTrie(common.Hash{}, db)
	return &RecordContext{
		confirmationTrie:  confirmationTrie,
		authorizationTrie: authorizationTrie,
		transferTrie:      transferTrie,
		db:                db,
	}, nil
}

func NewRecordContextFromProto(db ethdb.Database, ctxProto *RecordContextProto) (*RecordContext, error) {
	confirmationTrie, err := NewConfirmationTrie(ctxProto.ConfirmationHash, db)
	if err != nil {
		return nil, err
	}
	authorizationTrie, err := NewAuthorizationTrie(ctxProto.AuthorizationHash, db)
	if err != nil {
		return nil, err
	}
	transferTrie, err := NewTransferTrie(ctxProto.TransferHash, db)
	if err != nil {
		return nil, err
	}
	return &RecordContext{
		confirmationTrie:        confirmationTrie,
		authorizationTrie: authorizationTrie,
		transferTrie:     transferTrie,
		db:               db,
	}, nil
}

func (rc *RecordContext) Copy() *RecordContext {
	confirmationTrie := *rc.confirmationTrie
	authorizationTrie := *rc.authorizationTrie
	transferTrie := *rc.transferTrie
	return &RecordContext{
		confirmationTrie:  &confirmationTrie,
		authorizationTrie: &authorizationTrie,
		transferTrie:      &transferTrie,
	}
}

func (rc *RecordContext) Root() (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, rc.confirmationTrie.Hash())
	rlp.Encode(hw, rc.authorizationTrie.Hash())
	rlp.Encode(hw, rc.transferTrie.Hash())
	hw.Sum(h[:0])
	return h
}

func (rc *RecordContext) Snapshot() *RecordContext {
	return rc.Copy()
}

func (rc *RecordContext) RevertToSnapShot(snapshot *RecordContext) {
	rc.confirmationTrie = snapshot.confirmationTrie
	rc.authorizationTrie = snapshot.authorizationTrie
	rc.transferTrie = snapshot.transferTrie
}

func (rc *RecordContext) FromProto(ctx *RecordContextProto) error {
	var err error
	rc.confirmationTrie, err = NewConfirmationTrie(ctx.ConfirmationHash, rc.db)
	if err != nil {
		return err
	}
	rc.authorizationTrie, err = NewAuthorizationTrie(ctx.AuthorizationHash, rc.db)
	if err != nil {
		return err
	}
	rc.transferTrie, err = NewTransferTrie(ctx.TransferHash, rc.db)
	return err
}

type RecordContextProto struct {
	ConfirmationHash    common.Hash `json:"confirmationRoot"     gencodec:"required"`
	AuthorizationHash   common.Hash `json:"authorizationRoot"    gencodec:"required"`
	TransferHash        common.Hash `json:"TransferRoot"         gencodec:"required"`
}

func (rc *RecordContext) ToProto() *RecordContextProto {
	return &RecordContextProto{
		ConfirmationHash:  rc.confirmationTrie.Hash(),
		AuthorizationHash: rc.authorizationTrie.Hash(),
		TransferHash:      rc.transferTrie.Hash(),
	}
}

func (rc *RecordContext) CommitTo(dbw trie.DatabaseWriter) (*RecordContextProto, error) {
	confirmationRoot, err := rc.confirmationTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}
	authorizationRoot, err := rc.authorizationTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}
	transferRoot, err := rc.transferTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}

	return &RecordContextProto{
		ConfirmationHash:        confirmationRoot,
		AuthorizationHash: authorizationRoot,
		TransferHash:     transferRoot,
	}, nil
}

func (rc *RecordContext) ConfirmationTrie() *trie.Trie            { return rc.confirmationTrie }
func (rc *RecordContext) AuthorizationTrie() *trie.Trie           { return rc.authorizationTrie }
func (rc *RecordContext) TransferTrie() *trie.Trie                { return rc.transferTrie }
func (rc *RecordContext) DB() ethdb.Database                      { return rc.db }
func (rc *RecordContext) SetConfirmation(confirmation *trie.Trie) { rc.confirmationTrie = confirmation }
func (rc *RecordContext) SetAuthorization(authorization *trie.Trie) { rc.authorizationTrie = authorization }
func (rc *RecordContext) SetTransfer(transfer *trie.Trie)         { rc.transferTrie = transfer }

// 保存数据
func (rc *RecordContext) ConfirmRecord(addr common.Address,data []byte,record Record) error {
	/*err := rc.confirmationTrie.TryUpdate(data, record)
	if err != nil {
		return err
	}
	log.Info(" Confirm Record account: "+addr.String())*/
	return nil
}

func (rc *RecordContext)AuthorizeRecord(addr common.Address,data []byte) error{
	err := rc.authorizationTrie.TryUpdate(data, data)
	if err != nil {
		return err
	}
	log.Info(" Authorize Record account: "+addr.String())
	return nil
}

func (rc *RecordContext)TransferRecord(addr common.Address,data []byte) error{
	err := rc.transferTrie.TryUpdate(data, data)
	if err != nil {
		return err
	}
	log.Info(" Authorize Record account: "+addr.String())
	return nil
}