package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"

	"github.com/hyperledger/fabric-private-chaincode/tle_go/mocks"
	"github.com/hyperledger/fabric-protos-go/common"
	protospeer "github.com/hyperledger/fabric-protos-go/peer"
	configtxtest "github.com/hyperledger/fabric/common/configtx/test"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/committer/txvalidator/plugin"
	"github.com/hyperledger/fabric/core/committer/txvalidator/v20/plugindispatcher"
	validation "github.com/hyperledger/fabric/core/handlers/validation/api"
	ledgermocks "github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func mockBlock(t *testing.T, channel string, seqNum uint64, localSigner *mocks.SignerSerializer, dataHash []byte) (*common.Block, []byte) {
	block := protoutil.NewBlock(seqNum, nil)

	// Add a fake transaction to the block referring channel "C"
	sProp, _ := protoutil.MockSignedEndorserProposalOrPanic(channel, &protospeer.ChaincodeSpec{}, []byte("transactor"), []byte("transactor's signature"))
	sPropRaw, err := protoutil.Marshal(sProp)
	require.NoError(t, err, "Failed marshalling signed proposal")
	block.Data.Data = [][]byte{sPropRaw}

	// Compute hash of block.Data and put into the Header
	if len(dataHash) != 0 {
		block.Header.DataHash = dataHash
	} else {
		block.Header.DataHash = protoutil.BlockDataHash(block.Data)
	}

	// Add signer's signature to the block
	shdr, err := protoutil.NewSignatureHeader(localSigner)
	require.NoError(t, err, "Failed generating signature header")

	blockSignature := &common.MetadataSignature{
		SignatureHeader: protoutil.MarshalOrPanic(shdr),
	}

	// Note, this value is intentionally nil, as this metadata is only about the signature, there is no additional metadata
	// information required beyond the fact that the metadata item is signed.
	blockSignatureValue := []byte(nil)

	msg := util.ConcatenateBytes(blockSignatureValue, blockSignature.SignatureHeader, protoutil.BlockHeaderBytes(block.Header))
	localSigner.SignReturns(msg, nil)
	blockSignature.Signature, err = localSigner.Sign(msg)
	require.NoError(t, err, "Failed signing block")

	block.Metadata.Metadata[common.BlockMetadataIndex_SIGNATURES] = protoutil.MarshalOrPanic(&common.Metadata{
		Value: blockSignatureValue,
		Signatures: []*common.MetadataSignature{
			blockSignature,
		},
	})

	return block, msg
}

func TestVerifyBlock(t *testing.T) {
	aliceSigner := &mocks.SignerSerializer{}
	aliceSigner.SerializeReturns([]byte("Alice"), nil)
	policyManagerGetter := &mocks.ChannelPolicyManagerGetterWithManager{
		Managers: map[string]policies.Manager{
			"A": &mocks.ChannelPolicyManager{
				Policy: &mocks.Policy{Deserializer: &mocks.IdentityDeserializer{Identity: []byte("Bob"), Msg: []byte("msg2"), Mock: mock.Mock{}}},
			},
			"B": &mocks.ChannelPolicyManager{
				Policy: &mocks.Policy{Deserializer: &mocks.IdentityDeserializer{Identity: []byte("Charlie"), Msg: []byte("msg3"), Mock: mock.Mock{}}},
			},
			"C": &mocks.ChannelPolicyManager{
				Policy: &mocks.Policy{Deserializer: &mocks.IdentityDeserializer{Identity: []byte("Alice"), Msg: []byte("msg1"), Mock: mock.Mock{}}},
			},
			"D": &mocks.ChannelPolicyManager{
				Policy: &mocks.Policy{Deserializer: &mocks.IdentityDeserializer{Identity: []byte("Alice"), Msg: []byte("msg1"), Mock: mock.Mock{}}},
			},
		},
	}

	// - Prepare testing valid block, Alice signs it.
	blockRaw, msg := mockBlock(t, "C", 42, aliceSigner, nil)
	policyManagerGetter.Managers["C"].(*mocks.ChannelPolicyManager).Policy.(*mocks.Policy).Deserializer.(*mocks.IdentityDeserializer).Msg = msg
	blockRaw2, msg2 := mockBlock(t, "D", 42, aliceSigner, nil)
	policyManagerGetter.Managers["D"].(*mocks.ChannelPolicyManager).Policy.(*mocks.Policy).Deserializer.(*mocks.IdentityDeserializer).Msg = msg2

	// - Verify block
	// require.NoError(t, msgCryptoService.VerifyBlock([]byte("C"), 42, blockRaw))
	// // Wrong sequence number claimed
	// err = msgCryptoService.VerifyBlock([]byte("C"), 43, blockRaw)
	// require.Error(t, err)
	// require.Contains(t, err.Error(), "but actual seqNum inside block is")
	// delete(policyManagerGetter.Managers, "D")
	// nilPolMgrErr := msgCryptoService.VerifyBlock([]byte("D"), 42, blockRaw2)
	// require.Contains(t, nilPolMgrErr.Error(), "Could not acquire policy manager")
	// require.Error(t, nilPolMgrErr)
	// require.Error(t, msgCryptoService.VerifyBlock([]byte("A"), 42, blockRaw))
	// require.Error(t, msgCryptoService.VerifyBlock([]byte("B"), 42, blockRaw))

	require.NoError(t, VerifyBlock(policyManagerGetter, []byte("C"), 42, blockRaw))
	// Wrong sequence number claimed
	err := VerifyBlock(policyManagerGetter, []byte("C"), 43, blockRaw)
	require.Error(t, err)
	require.Contains(t, err.Error(), "but actual seqNum inside block is")
	delete(policyManagerGetter.Managers, "D")
	nilPolMgrErr := VerifyBlock(policyManagerGetter, []byte("D"), 42, blockRaw2)
	require.Contains(t, nilPolMgrErr.Error(), "Could not acquire policy manager")
	require.Error(t, nilPolMgrErr)
	require.Error(t, VerifyBlock(policyManagerGetter, []byte("A"), 42, blockRaw))
	require.Error(t, VerifyBlock(policyManagerGetter, []byte("B"), 42, blockRaw))

	// - Prepare testing invalid block (wrong data has), Alice signs it.
	blockRaw, msg = mockBlock(t, "C", 42, aliceSigner, []byte{0})
	policyManagerGetter.Managers["C"].(*mocks.ChannelPolicyManager).Policy.(*mocks.Policy).Deserializer.(*mocks.IdentityDeserializer).Msg = msg

	// - Verify block
	require.Error(t, VerifyBlock(policyManagerGetter, []byte("C"), 42, blockRaw))

	// Check invalid args
	require.Error(t, VerifyBlock(policyManagerGetter, []byte("C"), 42, &common.Block{}))
}

func TestCreateValidator(t *testing.T) {
	// require fake peer.
	peerInstance, cleanup := peer.NewTestPeer2(t)
	defer cleanup()

	var initArg string
	peerInstance.Initialize(
		func(cid string) { initArg = cid },
		nil,
		plugin.MapBasedMapper(map[string]validation.PluginFactory{}),
		&ledgermocks.DeployedChaincodeInfoProvider{},
		nil,
		nil,
		runtime.NumCPU(),
	)

	testChannelID := fmt.Sprintf("mytestchannelid-%d", rand.Int())
	block, err := configtxtest.MakeGenesisBlock(testChannelID)
	if err != nil {
		fmt.Printf("Failed to create a config block, %s err %s\n,", initArg, err)
		t.FailNow()
	}

	err = peerInstance.CreateChannel(testChannelID, block, &ledgermocks.DeployedChaincodeInfoProvider{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create chain %s", err)
	}

	// Initialize gossip service
	nWorkers := 10
	cid := testChannelID
	cryptoProvider := peerInstance.CryptoProvider
	// cryptoProvider, err := sw.NewDefaultSecurityLevelWithKeystore(sw.NewDummyKeyStore())
	// require.NoError(t, err)
	channel := peerInstance.Channel(cid)
	// bundle := (*channelconfig.Bundle)(nil)
	pluginMapper := plugin.MapBasedMapper(map[string]validation.PluginFactory{})
	policyManagerFunc := peerInstance.GetPolicyManager
	legacyLifecycleValidation := (plugindispatcher.LifecycleResources)(nil)
	newLifecycleValidation := (plugindispatcher.CollectionAndLifecycleResources)(nil)

	_, err = CreateTxValidator(
		nWorkers,
		cid,
		cryptoProvider,
		channel,
		// bundle,
		pluginMapper,
		policyManagerFunc,
		legacyLifecycleValidation,
		newLifecycleValidation)

	require.NoError(t, err)
}
