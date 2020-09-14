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

package core

import (
	"AQChainRe/pkg/common"
	"AQChainRe/pkg/core/state"
	"AQChainRe/pkg/core/types"
	"AQChainRe/pkg/log"
	"AQChainRe/pkg/rlp"
	"errors"
	"fmt"
	"math/big"
)

var (
	Big0                         = big.NewInt(0)
	errInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")
	ErrInsufficientBalance       = errors.New("insufficient balance for transfer")
)

/*
The State Transitioning Model

A state transition is a change made when a transaction is applied to the current world state
The state transitioning model does all all the necessary work to work out a valid new state root.

1) Nonce handling
2) Pre pay gas
3) Create a new state object if the recipient is \0*32
4) Value transfer
== If contract creation ==
  4a) Attempt to run transaction data
  4b) If valid, use result as code for the new state object
== end ==
5) Run Script section
6) Derive new state root
*/
type StateTransition struct {
	msg        Message
	gas        uint64
	gasPrice   *big.Int
	initialGas *big.Int
	value      *big.Int
	data       []byte
	statedb    *state.StateDB
}

// Message represents a message sent to a contract.
type Message interface {
	From() common.Address
	//FromFrontier() (common.Address, error)
	To() *common.Address

	GasPrice() *big.Int
	Gas() *big.Int
	Value() *big.Int

	Nonce() uint64
	CheckNonce() bool
	Data() []byte
	Type() types.TxType
}

// NewStateTransition initialises and returns a new state transition object.
func NewStateTransition(msg Message, statedb *state.StateDB) *StateTransition {
	return &StateTransition{
		msg:        msg,
		gasPrice:   msg.GasPrice(),
		initialGas: new(big.Int),
		value:      msg.Value(),
		data:       msg.Data(),
		statedb:    statedb,
	}
}

// ApplyMessage computes the new state by applying the given message
// against the old state within the environment.
//
// ApplyMessage returns the bytes returned by any EVM execution (if it took place),
// the gas used (which includes gas refunds) and an error if it failed. An error always
// indicates a core error meaning that the message would always fail for that particular
// state and would never be accepted within a block.
func ApplyMessage(msg Message, statedb *state.StateDB) ([]byte, bool, error) {
	st := NewStateTransition(msg, statedb)
	return st.TransitionDb()
}

func (st *StateTransition) from() common.Address {
	f := st.msg.From()
	if !st.statedb.Exist(f) {
		st.statedb.CreateAccount(f)
	}
	return f
}

func (st *StateTransition) to() common.Address {
	if st.msg == nil {
		return common.Address{}
	}
	to := st.msg.To()
	if to == nil {
		return common.Address{}
	}

	if !st.statedb.Exist(*to) {
		st.statedb.CreateAccount(*to)
	}
	return *to
}

func (st *StateTransition) preCheck() error {
	msg := st.msg
	sender := st.from()

	// Make sure this transaction's nonce is correct
	if msg.CheckNonce() {
		nonce := st.statedb.GetNonce(sender)
		if nonce < msg.Nonce() {
			return ErrNonceTooHigh
		} else if nonce > msg.Nonce() {
			return errors.New("nonce too low")
		}
	}
	return nil
}

// TransitionDb will transition the state by applying the current message and returning the result
// including the required gas for the operation as well as the used gas. It returns an error if it
// failed. An error indicates a consensus issue.
func (st *StateTransition) TransitionDb() (ret []byte, failed bool, err error) {
	if err = st.preCheck(); err != nil {
		return
	}
	msg := st.msg
	sender := st.from()
	recipient := st.to()
	value := msg.Value()
	stateDB := st.statedb

	// 增加账户的交易数
	st.statedb.SetNonce(sender, st.statedb.GetNonce(sender)+1)

	// 检查发送者余额
	if !CanTransfer(stateDB, sender, value) {
		return nil, true, ErrInsufficientBalance
	}

	// 检查接收地址是否存在
	if !stateDB.Exist(recipient) {
		stateDB.CreateAccount(recipient)
	}

	// 转账
	Transfer(stateDB, sender, recipient, value)

	/*
		if msg.Type() == types.Binary {
			// 记录执行成功的交易
				// 测试时有空指针
				if pocContext != nil {
					// 记录最近执行成功的交易
					lastedTx:= types.AccountLastedTx{
						Account:    msg.From(),
						TxHash:     receipt.TxHash,
						RecordTime: big.NewInt(time.Now().Unix()),
					}
					if err = pocContext.SetLastedTx(lastedTx); err != nil {
						return nil, err
					}
					contractCreation := msg.To() == nil
					if contractCreation {
						// 给合约创建者 加贡献值
						c,err:=poc.AccumulateContribution(pocContext,msg.From())
						if err!=nil{

						}
						statedb.AddContribution(msg.From(),c)
						log.Info("Contract Creation Add 1e+18 Contribution")
						log.Info(fmt.Sprintf("Transition Sender %s",msg.From().String()))
					} else {
						// 给交易发起者 加贡献值
						c,err:=poc.AccumulateContribution(pocContext,msg.From())
						if err!=nil{

						}
						statedb.AddContribution(msg.From(),c)
						log.Info("Transition Add 2e+18 Contribution")
						log.Info(fmt.Sprintf("Transition Sender %s",msg.From().String()))
					}
				}
		}*/

	return ret, failed, err
}

// CanTransfer checks wether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db *state.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db *state.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

func applyPocMessage(pocContext *types.PocContext, msg types.Message) error {
	switch msg.Type() {
	case types.LoginCandidate:
		pocContext.BecomeCandidate(msg.From())
	case types.LogoutCandidate:
		pocContext.KickoutCandidate(msg.From())
	default:
		return types.ErrInvalidType
	}
	return nil
}

func ApplyDataMessage(txHash common.Hash, msg Message, statedb *state.StateDB, statedbRecord *state.StateDBRecord) (failed bool, err error) {
	st := NewStateTransition(msg, statedb)

	if err = st.preCheck(); err != nil {
		return
	}

	msg = st.msg
	sender := st.from()
	b, _ := rlp.EncodeToBytes(msg.Data())
	hash := common.BytesToHash(b)
	// 增加账户的交易数
	st.statedb.SetNonce(sender, st.statedb.GetNonce(sender)+1)

	switch msg.Type() {
	case types.ConfirmationData:
		// 检查数据唯一性
		if statedbRecord.Exist(hash) {
			return true, errors.New("")
		}

		// stateRecord 生成记录
		obj := statedbRecord.GetOrNewStateObject(hash)
		obj.SetOrigin(sender)
		obj.SetOwner(sender)
		obj.SetTxs([]common.Hash{txHash})

		// 添加账户的记录
		statedb.AddRecords(sender, hash)
		// 贡献值计算 先直接加2e+18
		statedb.AddContribution(sender, big.NewInt(2e+18))
		log.Info("ConfirmationData Add 1e+18 Contribution")
		log.Info(fmt.Sprintf("Transition Sender %s", sender))

	case types.AuthorizationData:
	case types.TransferData:
		// 检查是否可以进行转移
		if statedbRecord.GetOwner(hash).Hash() != sender.Hash() {
			return true, errors.New("")
		}

		// 状态
		if statedbRecord.GetStatus(hash) != 0 {
			return true, errors.New("")
		}

		// 转移拥有者
		statedbRecord.SetOwner(hash, st.to())

		// 添加交易记录
		statedbRecord.AddTxHash(hash, txHash)

		// 为账户添加删除记录
		statedb.AddRecords(st.to(), txHash)
		statedb.RemoveRecords(sender, txHash)
		// 贡献值
		statedb.AddContribution(sender, big.NewInt(1e+18))
		log.Info("TransferData Add 1e+18 Contribution")
		log.Info(fmt.Sprintf("Transition Sender %s", sender))

	}

	return false, err
}
