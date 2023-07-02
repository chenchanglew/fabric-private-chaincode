/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package enclave_go

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/mtreecomp/mtreeimpl"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/mtreecomp/types"
	"github.com/pkg/errors"
)

type MerkleStubInterface struct {
	*FpcStubInterface
	merkleRoot []byte
}

func NewMerkleStubInterface(stub shim.ChaincodeStubInterface, input *pb.ChaincodeInput, rwset *readWriteSet, sep StateEncryptionFunctions) shim.ChaincodeStubInterface {
	logger.Debugf("==== Get New Merkle Interface =====")
	fpcStub := NewFpcStubInterface(stub, input, rwset, sep)
	MerkleStub := MerkleStubInterface{fpcStub.(*FpcStubInterface), nil}
	return &MerkleStub
}

func (s *MerkleStubInterface) DecideMerkleRoot(merkleRootHashes [][]byte) error {
	// TODO: Extract merkle roots from hashes

	// TODO: Verify signature and remove duplicate

	// TODO: Decide on the majority, and make sure it is more than 2/3 peers.
	// Q: How to get total number of peers?

	return nil
}

func (s *MerkleStubInterface) extractMerklePath([]byte) ([]types.MerklePath, error) {
	panic("Not Implemented")
}

func (s *MerkleStubInterface) GetState(key string) ([]byte, error) {
	encValue, err := s.GetPublicState(key)
	if err != nil {
		return nil, err
	}
	if len(encValue) == 0 {
		return nil, errors.New("Merkle Solution, KVS key not found")
	}
	if s.merkleRoot == nil {
		return nil, errors.New("Merkle Solution, Merkle Root not yet decided.")
	}
	// TODO: extract merklepath from encValue, Q: How?
	merklePath, err := s.extractMerklePath(encValue)
	if err != nil {
		return nil, err
	}
	data, err := s.sep.DecryptState(encValue)
	if err != nil {
		return nil, err
	}

	c := types.KVScontent{
		Key:   key,
		Value: data,
	}

	// verify merklepath
	valid, err := mtreeimpl.VerifyMerklePath(c, merklePath, s.merkleRoot, nil)
	if !valid {
		return nil, errors.New("Verify MerklePath Failed, merkleRoot not match.")
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *MerkleStubInterface) PutState(key string, value []byte) error {
	return s.FpcStubInterface.PutState(key, value)
}

func (s *MerkleStubInterface) GetStateByRange(startKey string, endKey string) (shim.StateQueryIteratorInterface, error) {
	panic("not implemented") // TODO: Implement
}

func (s *MerkleStubInterface) GetStateByRangeWithPagination(startKey string, endKey string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	panic("not implemented") // TODO: Implement
}
