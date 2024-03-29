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
	"AQChainRe/pkg/core/state"
	"AQChainRe/pkg/core/types"
	"math/big"
)

// Validator is an interface which defines the standard for block validation. It
// is only responsible for validating block contents, as the header validation is
// done by the specific consensus engines.
//
type Validator interface {
	// ValidateBody validates the given block's content.
	ValidateBody(block *types.Block) error

	// ValidateState validates the given stateDB and optionally the receipts and
	// gas used.
	ValidateState(block, parent *types.Block, state *state.StateDB, stateRecord *state.StateDBRecord, receipts types.Receipts, usedGas *big.Int) error
	// ValidatePocState validates the given poc state
	ValidatePocState(block *types.Block) error
}

// Processor is an interface for processing blocks using a given initial state.
//
// Process takes the block to be processed and the stateDB upon which the
// initial state is based. It should return the receipts generated, amount
// of gas used in the process and return an error if any of the internal rules
// failed.
type Processor interface {
	Process(block *types.Block, statedb *state.StateDB, statedbRecord *state.StateDBRecord) (types.Receipts, []*types.Log, *big.Int, error)
}
