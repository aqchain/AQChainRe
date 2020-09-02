// Copyright 2015 The go-ethereum Authors
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
	"AQChainRe/pkg/consensus"
	"AQChainRe/pkg/consensus/misc"
	"AQChainRe/pkg/core/state"
	"AQChainRe/pkg/core/types"
	"AQChainRe/pkg/params"
	"math/big"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for block rewards
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the stateDB and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB,statedbRecord *state.StateDBRecord) (types.Receipts, []*types.Log, *big.Int, error) {
	var (
		receipts     types.Receipts
		totalUsedGas = big.NewInt(0)
		header       = block.Header()
		allLogs      []*types.Log
	)
	// Mutate the the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	// Set block poc context
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		statedb.Prepare(tx.Hash(), block.Hash(), i)
		receipt, err := ApplyTransaction(p.config, block.PocCtx(), p.bc, nil,statedb,statedbRecord, header, tx)
		if err != nil {
			return nil, nil, nil, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles(), receipts, block.PocCtx())

	return receipts, allLogs, totalUsedGas, nil
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
// 交易执行的过程
func ApplyTransaction(config *params.ChainConfig, pocContext *types.PocContext, bc *BlockChain, coinbase *common.Address, statedb *state.StateDB, statedbRecord *state.StateDBRecord,header *types.Header, tx *types.Transaction) (*types.Receipt, error) {
	// 转换成Message类型
	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number))
	if err != nil {
		return nil, err
	}

	// 验证
	err = tx.Validate()
	if err != nil {
		return nil, err
	}

	var failed bool

	// 没有了evm 直接区分三种交易类型进行分别处理 转账 共识 数据记录
	if msg.Type() == types.Binary{
		_, failed ,err = ApplyMessage(msg,statedb)
		if err != nil {
			return nil, err
		}
	}

	if msg.Type() == types.ConfirmationData ||  msg.Type() == types.AuthorizationData || msg.Type() == types.TransferData{
		failed, err = ApplyDataMessage(msg,statedb,statedbRecord)
		if err != nil {
			return nil, err
		}
	}

	if msg.Type() == types.LoginCandidate ||  msg.Type() == types.LogoutCandidate{
		err = applyPocMessage(pocContext,msg)
		if err != nil {
			return nil, err
		}
	}

	// Update the state with pending changes
	var root []byte
	var rootRecord []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
		statedbRecord.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
		rootRecord = statedbRecord.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing wether the root touch-delete accounts.
	receipt := types.NewReceipt(root,rootRecord,failed)
	receipt.TxHash = tx.Hash()

	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return receipt, err
}
