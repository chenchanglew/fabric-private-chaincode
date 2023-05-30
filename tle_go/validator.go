/*
Copyright IBM Corp. All Rights Reserved.
Copyright 2020 Intel Corporation

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"fmt"

	pcommon "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/semaphore"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/committer/txvalidator/plugin"
	validatorv14 "github.com/hyperledger/fabric/core/committer/txvalidator/v14"
	validatorv20 "github.com/hyperledger/fabric/core/committer/txvalidator/v20"
	"github.com/hyperledger/fabric/core/committer/txvalidator/v20/plugindispatcher"
	vir "github.com/hyperledger/fabric/core/committer/txvalidator/v20/valinforetriever"
	"github.com/hyperledger/fabric/core/handlers/library"
	validation "github.com/hyperledger/fabric/core/handlers/validation/api"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/protoutil"
)

func VerifyBlock(channelPolicyManagerGetter policies.ChannelPolicyManagerGetter, chainID common.ChannelID, seqNum uint64, block *pcommon.Block) error {
	fmt.Println("start verifyBlock")
	if block.Header == nil {
		return fmt.Errorf("Invalid Block on channel [%s]. Header must be different from nil.", chainID)
	}

	blockSeqNum := block.Header.Number
	if seqNum != blockSeqNum {
		return fmt.Errorf("Claimed seqNum is [%d] but actual seqNum inside block is [%d]", seqNum, blockSeqNum)
	}

	// - Extract channelID and compare with chainID
	channelID, err := protoutil.GetChannelIDFromBlock(block)
	if err != nil {
		return fmt.Errorf("Failed getting channel id from block with id [%d] on channel [%s]: [%s]", block.Header.Number, chainID, err)
	}

	if channelID != string(chainID) {
		return fmt.Errorf("Invalid block's channel id. Expected [%s]. Given [%s]", chainID, channelID)
	}

	// - Unmarshal medatada
	if block.Metadata == nil || len(block.Metadata.Metadata) == 0 {
		return fmt.Errorf("Block with id [%d] on channel [%s] does not have metadata. Block not valid.", block.Header.Number, chainID)
	}

	metadata, err := protoutil.GetMetadataFromBlock(block, pcommon.BlockMetadataIndex_SIGNATURES)
	if err != nil {
		return fmt.Errorf("Failed unmarshalling medatata for signatures [%s]", err)
	}

	// - Verify that Header.DataHash is equal to the hash of block.Data
	// This is to ensure that the header is consistent with the data carried by this block
	if !bytes.Equal(protoutil.BlockDataHash(block.Data), block.Header.DataHash) {
		return fmt.Errorf("Header.DataHash is different from Hash(block.Data) for block with id [%d] on channel [%s]", block.Header.Number, chainID)
	}

	// - Get Policy for block validation

	// Get the policy manager for channelID
	cpm := channelPolicyManagerGetter.Manager(channelID)
	if cpm == nil {
		return fmt.Errorf("Could not acquire policy manager for channel %s", channelID)
	}

	// Get block validation policy
	policy, ok := cpm.GetPolicy(policies.BlockValidation)
	// ok is true if it was the policy requested, or false if it is the default policy
	fmt.Printf("Got block validation policy for channel [%s] with flag [%t], policy [%s]\n", channelID, ok, policy)

	// - Prepare SignedData
	signatureSet := []*protoutil.SignedData{}
	for _, metadataSignature := range metadata.Signatures {
		shdr, err := protoutil.UnmarshalSignatureHeader(metadataSignature.SignatureHeader)
		if err != nil {
			return fmt.Errorf("Failed unmarshalling signature header for block with id [%d] on channel [%s]: [%s]", block.Header.Number, chainID, err)
		}
		signatureSet = append(
			signatureSet,
			&protoutil.SignedData{
				Identity:  shdr.Creator,
				Data:      util.ConcatenateBytes(metadata.Value, metadataSignature.SignatureHeader, protoutil.BlockHeaderBytes(block.Header)),
				Signature: metadataSignature.Signature,
			},
		)
	}

	// - Evaluate policy
	fmt.Println("Start evaluateSignedData")
	// if len(signatureSet) > 0 {
	// 	fmt.Printf("signatureSet[0].Identity = [%x]\n", signatureSet[0].Identity)
	// 	fmt.Printf("signatureSet[0].Data = [%x]\n", signatureSet[0].Data)
	// 	fmt.Printf("signatureSet[0].Signature = [%x]\n", signatureSet[0].Signature)
	// }
	return policy.EvaluateSignedData(signatureSet)
}

func CreateTxValidatorViaPeer(peerInstance *peer.Peer, cid string, legacyLifecycleValidation plugindispatcher.LifecycleResources, newLifecycleValidation plugindispatcher.CollectionAndLifecycleResources) (*txvalidator.ValidationRouter, error) {
	libConf, err := library.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("could not decode peer handlers configuration [%s]", err)
	}
	reg := library.InitRegistry(libConf)
	validationPluginsByName := reg.Lookup(library.Validation).(map[string]validation.PluginFactory)

	nWorkers := 1
	cryptoProvider := peerInstance.CryptoProvider
	// cryptoProvider, err := sw.NewDefaultSecurityLevelWithKeystore(sw.NewDummyKeyStore())
	// require.NoError(t, err)
	channel := peerInstance.Channel(cid)
	// bundle := (*channelconfig.Bundle)(nil)
	pluginMapper := plugin.MapBasedMapper(validationPluginsByName)
	policyManagerFunc := peerInstance.GetPolicyManager

	validator, err := CreateTxValidator(
		nWorkers,
		cid,
		cryptoProvider,
		channel,
		// bundle,
		pluginMapper,
		policyManagerFunc,
		legacyLifecycleValidation,
		newLifecycleValidation)
	return validator, err
}

func CreateTxValidator(
	nWorkers int,
	cid string,
	cryptoProvider bccsp.BCCSP,
	channel *peer.Channel,
	// bundle *channelconfig.Bundle,
	pluginMapper plugin.MapBasedMapper,
	policyManagerFunc func(cid string) policies.Manager,
	legacyLifecycleValidation plugindispatcher.LifecycleResources,
	newLifecycleValidation plugindispatcher.CollectionAndLifecycleResources,
) (*txvalidator.ValidationRouter, error) {

	validationWorkersSemaphore := semaphore.New(nWorkers)

	validator := &txvalidator.ValidationRouter{
		CapabilityProvider: channel,
		V14Validator: validatorv14.NewTxValidator(
			cid,
			validationWorkersSemaphore,
			channel,
			pluginMapper,
			cryptoProvider,
		),
		V20Validator: validatorv20.NewTxValidator(
			cid,
			validationWorkersSemaphore,
			channel,
			channel.Ledger(),
			&vir.ValidationInfoRetrieveShim{
				New:    newLifecycleValidation,
				Legacy: legacyLifecycleValidation,
			},
			&peer.CollectionInfoShim{
				CollectionAndLifecycleResources: newLifecycleValidation,
				// ChannelID:                       bundle.ConfigtxValidator().ChannelID(),
				ChannelID: cid,
			},
			pluginMapper,
			policies.PolicyManagerGetterFunc(policyManagerFunc),
			cryptoProvider,
		),
	}

	return validator, nil
}
