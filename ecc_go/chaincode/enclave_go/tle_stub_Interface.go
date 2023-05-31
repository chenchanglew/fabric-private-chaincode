/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package enclave_go

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log"

	"google.golang.org/grpc"

	tle "github.com/hyperledger/fabric-private-chaincode/tle_go/tlegrpc"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
)

type TleStubInterface struct {
	*FpcStubInterface
	LastCommitHash []byte
	Address        string
}

func NewTleStubInterface(stub shim.ChaincodeStubInterface, input *pb.ChaincodeInput, rwset *readWriteSet, sep StateEncryptionFunctions) shim.ChaincodeStubInterface {
	logger.Warning("==== Get New TLE Interface =====")
	fpcStub := NewFpcStubInterface(stub, input, rwset, sep)
	tleStub := TleStubInterface{fpcStub.(*FpcStubInterface), []byte{}, "localhost:50051"}
	err := tleStub.InitTleStub()
	if err != nil {
		logger.Warningf("Error!! Initializing SKVS failed")
	}
	return &tleStub
}

func (s *TleStubInterface) InitTleStub() error {
	logger.Warningf(" === Initializing Tle Stub === ")

	// establish secure connection to TLE (TODO)
	conn, err := grpc.Dial(s.Address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	// Create a new gRPC clien
	client := tle.NewTleServiceClient(conn)
	response, err := client.GetSession(context.Background(), &tle.Empty{})
	if err != nil {
		return err
	}
	s.LastCommitHash = response.GetLastCommitHash()
	logger.Warningf("tle Stub Init finish, lastCommitHash: %s", s.LastCommitHash)
	return nil
}

func (s *TleStubInterface) ValidateMeta(metadata []byte, encValue []byte) error {
	// validate meta data
	hash := sha256.Sum256(encValue)
	if bytes.Equal(hash[:], metadata) {
		return nil
	}
	// TODO
	fmt.Printf("Validate Metadata failed, metadata: %x != hash: %x\n", metadata, hash)
	// return errors.Errorf("Validate Metadata failed, metadata: %x != hash: %x", metadata, hash)
	return nil
}

func (s *TleStubInterface) GetMeta(key string) ([]byte, error) {
	logger.Warningf("Calling Get Meta from TLE, key: %s", key)
	// Q: How to get the namespace? or we dont need namespace?
	// establish secure connection to TLE (TODO)
	conn, err := grpc.Dial(s.Address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Create a new gRPC clien
	client := tle.NewTleServiceClient(conn)

	request := &tle.MetaRequest{
		Namespace: "fpc-secret-keeper-go",
		Key:       key,
	}
	response, err := client.GetMeta(context.Background(), request)
	if err != nil {
		log.Fatalf("Failed to call GetMeta: %v", err)
	}

	// Process the response
	data := response.GetData()
	lastCommitHash := response.GetLastCommitHash()
	fmt.Printf("Received data: %s, lastCommitHash: %s\n", string(data), string(lastCommitHash))

	if bytes.Equal(s.LastCommitHash, lastCommitHash) {
		return data, nil
	}
	return nil, errors.Errorf("Get Metadata failed, lastCommitHash recv: %x != session lastCommitHash: %x", lastCommitHash, s.LastCommitHash)
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

	err = s.ValidateMeta(metadata, encValue)
	if err != nil {
		return s.sep.DecryptState(encValue)
	}
	return nil, err
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
