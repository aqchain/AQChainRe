// Copyright 2017 The go-ethereum Authors
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

package poc

import (
	"AQChainRe/pkg/common"
	"AQChainRe/pkg/consensus"
	"AQChainRe/pkg/core/types"
	"AQChainRe/pkg/rpc"
	"math/big"
)

// API is a user facing RPC API to allow controlling the delegate and voting
// mechanisms of the delegated-proof-of-stake
type API struct {
	chain consensus.ChainReader
	poc   *Poc
}

// GetValidators retrieves the list of the validators at specified block
func (api *API) GetValidators(number *rpc.BlockNumber) ([]common.Address, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	if header == nil {
		return nil, errUnknownBlock
	}

	epochTrie, err := types.NewEpochTrie(header.PocContext.EpochHash, api.poc.db)
	if err != nil {
		return nil, err
	}
	pocContext := types.PocContext{}
	pocContext.SetEpoch(epochTrie)
	validators, err := pocContext.GetValidators()
	if err != nil {
		return nil, err
	}
	return validators, nil
}

// GetConfirmedBlockNumber retrieves the latest irreversible block
func (api *API) GetConfirmedBlockNumber() (*big.Int, error) {
	var err error
	header := api.poc.confirmedBlockHeader
	if header == nil {
		header, err = api.poc.loadConfirmedBlockHeader(api.chain)
		if err != nil {
			return nil, err
		}
	}
	return header.Number, nil
}

// 获取贡献值
func (api *API) GetContributions(number *rpc.BlockNumber) ([]types.AccountContribution, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	if header == nil {
		return nil, errUnknownBlock
	}

	contributionTrie, err := types.NewCandidateTrie(header.PocContext.ContributionHash, api.poc.db)
	if err != nil {
		return nil, err
	}
	pocContext := types.PocContext{}
	pocContext.SetContribution(contributionTrie)
	contributions, err := pocContext.GetContributions()
	if err != nil {
		return nil, err
	}
	return contributions, nil
}

// 获取验证者
func (api *API) GetCandidates(number *rpc.BlockNumber) ([]common.Address, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	if header == nil {
		return nil, errUnknownBlock
	}

	candidateTrie, err := types.NewCandidateTrie(header.PocContext.CandidateHash, api.poc.db)
	if err != nil {
		return nil, err
	}
	pocContext := types.PocContext{}
	pocContext.SetCandidate(candidateTrie)
	candidates, err := pocContext.GetCandidates()
	if err != nil {
		return nil, err
	}
	return candidates, nil
}

// 获取最后一次交易
func (api *API) GetLastedTx(account common.Address) (types.AccountLastedTx, error) {

	var tx types.AccountLastedTx
	header := api.chain.CurrentHeader()
	if header == nil {
		return tx, errUnknownBlock
	}

	latestTxTrie, err := types.NewLatestTxTrie(header.PocContext.LatestTxHash, api.poc.db)
	if err != nil {
		return tx, err
	}
	pocContext := types.PocContext{}
	pocContext.SetCandidate(latestTxTrie)
	tx, err = pocContext.GetLastedTx(account)
	if err != nil {
		return tx, err
	}
	return tx, nil
}
