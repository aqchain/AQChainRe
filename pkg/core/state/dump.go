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

package state

import (
	"AQChainRe/pkg/common"
	"AQChainRe/pkg/rlp"
	"AQChainRe/pkg/trie"
	"encoding/json"
	"fmt"
)

type DumpAccount struct {
	Balance      string            `json:"balance"`
	Contribution string            `json:"prev"`
	Nonce        uint64            `json:"nonce"`
	Root         string            `json:"root"`
	CodeHash     string            `json:"codeHash"`
	Code         string            `json:"code"`
	Storage      map[string]string `json:"storage"`
}

type Dump struct {
	Root     string                 `json:"root"`
	Accounts map[string]DumpAccount `json:"accounts"`
}

func (self *StateDB) RawDump() Dump {
	dump := Dump{
		Root:     fmt.Sprintf("%x", self.trie.Hash()),
		Accounts: make(map[string]DumpAccount),
	}

	it := trie.NewIterator(self.trie.NodeIterator(nil))
	for it.Next() {
		addr := self.trie.GetKey(it.Key)
		var data Account
		if err := rlp.DecodeBytes(it.Value, &data); err != nil {
			panic(err)
		}

		obj := newObject(nil, common.BytesToAddress(addr), data, nil)
		account := DumpAccount{
			Balance:      data.Balance.String(),
			Contribution: data.Contribution.String(),
			Nonce:        data.Nonce,
			Root:         common.Bytes2Hex(data.Root[:]),
			CodeHash:     common.Bytes2Hex(data.CodeHash),
			Code:         common.Bytes2Hex(obj.Code(self.db)),
			Storage:      make(map[string]string),
		}
		storageIt := trie.NewIterator(obj.getTrie(self.db).NodeIterator(nil))
		for storageIt.Next() {
			account.Storage[common.Bytes2Hex(self.trie.GetKey(storageIt.Key))] = common.Bytes2Hex(storageIt.Value)
		}
		dump.Accounts[common.Bytes2Hex(addr)] = account
	}
	return dump
}

func (self *StateDB) Dump() []byte {
	json, err := json.MarshalIndent(self.RawDump(), "", "    ")
	if err != nil {
		fmt.Println("dump err", err)
	}
	return json
}

type DumpRecord struct {
	Origin  string            `json:"origin"`
	Owner   string            `json:"owner"`
	Status  uint8             `json:"status"`
	Root    string            `json:"root"`
	Storage map[string]string `json:"storage"`
}

type Dump2 struct {
	Root    string                `json:"root"`
	Records map[string]DumpRecord `json:"records"`
}

func (self *StateDBRecord) RawDump() Dump2 {
	dump := Dump2{
		Root:    fmt.Sprintf("%x", self.trie.Hash()),
		Records: make(map[string]DumpRecord),
	}

	it := trie.NewIterator(self.trie.NodeIterator(nil))
	for it.Next() {
		fmt.Println(it.Key)
		addr := self.trie.GetKey(it.Key)
		fmt.Println(common.ToHex(addr))
		var data Record
		if err := rlp.DecodeBytes(it.Value, &data); err != nil {
			//panic(err)
		}

		obj := newObjectRecord(nil, common.BytesToHash(addr), data, nil)
		account := DumpRecord{
			Origin:  data.Origin.String(),
			Owner:   data.Owner.String(),
			Root:    common.Bytes2Hex(data.Root[:]),
			Storage: make(map[string]string),
		}
		storageIt := trie.NewIterator(obj.getTrie(self.db).NodeIterator(nil))
		for storageIt.Next() {
			account.Storage[common.Bytes2Hex(self.trie.GetKey(storageIt.Key))] = common.Bytes2Hex(storageIt.Value)
		}
		dump.Records[common.Bytes2Hex(addr)] = account
	}
	return dump
}

func (self *StateDBRecord) Dump() []byte {
	json, err := json.MarshalIndent(self.RawDump(), "", "    ")
	if err != nil {
		fmt.Println("dump err", err)
	}
	return json
}
