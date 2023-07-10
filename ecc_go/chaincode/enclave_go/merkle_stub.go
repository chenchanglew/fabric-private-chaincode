/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package enclave_go

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-private-chaincode/internal/protos"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/core/endorser"
	merkle "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/mtreecomp/mtreegrpc"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/mtreecomp/mtreeimpl"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/mtreecomp/types"
	"github.com/pkg/errors"
)

type MerkleStubInterface struct {
	*FpcStubInterface
	merkleRoot []byte
}

func NewMerkleStubInterface(stub shim.ChaincodeStubInterface, chaincodeRequest *protos.CleartextChaincodeRequest, rwset *readWriteSet, sep StateEncryptionFunctions) shim.ChaincodeStubInterface {
	logger.Debugf("==== Get New Merkle Interface =====")
	fpcStub := NewFpcStubInterface(stub, chaincodeRequest, rwset, sep)
	MerkleStub := MerkleStubInterface{fpcStub.(*FpcStubInterface), nil}

	merkleRootHashes := chaincodeRequest.GetMerkleRootHashes()
	logger.Debugf("MerkleRootHashes are: ", merkleRootHashes)
	MerkleStub.InitMerkleStub(merkleRootHashes)
	return &MerkleStub
}

// Verify signature & remove duplication,
// Avoid single peer sending multiple response to cover majority.
func (s *MerkleStubInterface) getUniqueHashes(merkleRootHashes [][]byte) map[string][]byte {
	rootsMap := map[string][]byte{}

	for _, signedRootBytes := range merkleRootHashes {
		var signedRoot merkle.SignedMerkleRootResponse
		err := proto.Unmarshal(signedRootBytes, &signedRoot)
		if err != nil {
			fmt.Println("Proto failed to Unmarshal signedRootbytes to SignedMerkleRootResponse")
		}
		merkleBytes := signedRoot.GetSerializedMerkleRootResponse()
		signature := signedRoot.GetSignature()
		var merkleRootResponse merkle.MerkleRootResponse
		err = proto.Unmarshal(merkleBytes, &merkleRootResponse)
		if err != nil {
			fmt.Println("Proto failed to Unmarshal merkleBytes to MerkleRootResponse")
		}
		merkleRoot := merkleRootResponse.GetData()

		// TODO verify signature here
		hexSignature := hex.EncodeToString(signature)
		rootsMap[hexSignature] = merkleRoot
	}
	return rootsMap
}

func (s *MerkleStubInterface) InitMerkleStub(merkleRootHashes [][]byte) error {
	rootsMap := s.getUniqueHashes(merkleRootHashes)

	counterMap := make(map[string]int)
	for _, value := range rootsMap {
		key := hex.EncodeToString(value)
		counterMap[key]++
	}

	// TODO: Use real total number of peers, instead of hardcoded
	totalPeers := 1
	threshold := (totalPeers * 2) / 3

	// Decide on the majority, and make sure it is more than 2/3 peers.
	for hexHash, count := range counterMap {
		if count > threshold {
			merkleRoot, err := hex.DecodeString(hexHash)
			if err != nil {
				return err
			}
			s.merkleRoot = merkleRoot
		}
	}

	return nil
}

func (s *MerkleStubInterface) extractMerkleValue(mv []byte) ([]types.MerklePath, []byte, error) {
	var merkleValue endorser.MerkleValue
	err := json.Unmarshal(mv, &merkleValue)
	if err != nil {
		return nil, nil, err
	}
	return merkleValue.Merklepath, merkleValue.Value, nil
}

func (s *MerkleStubInterface) GetState(key string) ([]byte, error) {
	byteMerkleValue, err := s.GetPublicState(key)
	if err != nil {
		return nil, err
	}
	if len(byteMerkleValue) == 0 {
		return nil, errors.New("Merkle Solution, KVS key not found")
	}
	if s.merkleRoot == nil {
		return nil, errors.New("Merkle Solution, Merkle Root not yet decided.")
	}
	// extract merklepath from encValue
	mpath, encValue, err := s.extractMerkleValue(byteMerkleValue)
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
	valid, err := mtreeimpl.VerifyMerklePath(c, mpath, s.merkleRoot, nil)
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
