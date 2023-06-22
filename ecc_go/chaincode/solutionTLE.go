/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chaincode

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-private-chaincode/ecc/chaincode"
	"github.com/hyperledger/fabric-private-chaincode/ecc/chaincode/ercc"
	"github.com/hyperledger/fabric-private-chaincode/ecc_go/chaincode/enclave_go"
	"github.com/hyperledger/fabric-private-chaincode/internal/endorsement"
)

func NewTleChaincode(cc shim.Chaincode) *chaincode.EnclaveChaincode {
	newStubInterfaceFunc := enclave_go.NewTleStubInterface
	ecc := &chaincode.EnclaveChaincode{
		Enclave:   enclave_go.NewEnclaveStub(cc, newStubInterfaceFunc),
		Validator: endorsement.NewValidator(),
		Extractor: &chaincode.ExtractorImpl{},
		Ercc:      &ercc.StubImpl{},
	}
	return ecc
}
