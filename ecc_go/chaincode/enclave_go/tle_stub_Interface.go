/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package enclave_go

import (
	"bytes"
	"crypto/sha256"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
)

type TleStubInterface struct {
	*FpcStubInterface
	sessionCookie string
}

func NewTleStubInterface(stub shim.ChaincodeStubInterface, input *pb.ChaincodeInput, rwset *readWriteSet, sep StateEncryptionFunctions) shim.ChaincodeStubInterface {
	logger.Warning("==== Get New TLE Interface =====")
	fpcStub := NewFpcStubInterface(stub, input, rwset, sep)
	tleStub := TleStubInterface{fpcStub.(*FpcStubInterface), ""}
	err := tleStub.InitTleStub()
	if err != nil {
		logger.Warningf("Error!! Initializing SKVS failed")
	}
	return &tleStub
}

func (s *TleStubInterface) InitTleStub() error {
	logger.Warningf(" === Initializing Tle Stub === ")

	// establish secure connection to TLE and get sesssion cookie.
	// TODO
	s.sessionCookie = "XXX"

	logger.Warningf("tle Stub Init finish, session_cookie: %s", s.sessionCookie)
	return nil
}

func (s *TleStubInterface) GetMeta(key string) ([]byte, error) {
	logger.Warningf("Calling Get Meta from TLE, key: %s", key)
	// TODO
	return []byte{}, nil
}

func (s *TleStubInterface) GetState(key string) ([]byte, error) {
	logger.Warningf("Calling Get State (Start), key: %s", key)

	// getmeta meta from TLE
	metadata, err := s.GetMeta(key)
	if err != nil {
		return nil, err
	}
	if metadata == nil {
		return nil, errors.New("TLE metadata key not found")
	}

	// getdata from state
	encValue, err := s.GetPublicState(key)
	if err != nil {
		return nil, err
	}
	if len(encValue) == 0 {
		return nil, errors.New("KVS key not found")
	}

	logger.Warningf("Calling Get State, encValue: %x, Metadata: %x", encValue, metadata)

	// validate meta data
	hash := sha256.Sum256(encValue)
	if bytes.Equal(hash[:], metadata) {
		return s.sep.DecryptState(encValue)
	}
	return nil, errors.Errorf("Validate Metadata failed, metadata: %x != hash: %x", metadata, hash)
}

func (s *TleStubInterface) PutState(key string, value []byte) error {
	return s.FpcStubInterface.PutState(key, value)
}

func (s *TleStubInterface) GetStateByRange(startKey string, endKey string) (shim.StateQueryIteratorInterface, error) {
	panic("not implemented") // TODO: Implement
}

func (s *TleStubInterface) GetStateByRangeWithPagination(startKey string, endKey string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	panic("not implemented") // TODO: Implement
}
