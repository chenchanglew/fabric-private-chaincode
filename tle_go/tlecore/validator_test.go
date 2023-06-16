package tlecore

import (
	"testing"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/bccsp/sw"
	"github.com/hyperledger/fabric/common/policydsl"
	"github.com/hyperledger/fabric/common/semaphore"
	tmocks "github.com/hyperledger/fabric/core/committer/txvalidator/mocks"
	txvalidatorplugin "github.com/hyperledger/fabric/core/committer/txvalidator/plugin"
	txvalidatorv20 "github.com/hyperledger/fabric/core/committer/txvalidator/v20"
	txvalidatormocks "github.com/hyperledger/fabric/core/committer/txvalidator/v20/mocks"
	plugindispatchermocks "github.com/hyperledger/fabric/core/committer/txvalidator/v20/plugindispatcher/mocks"
	ccp "github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/core/handlers/validation/builtin"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	mocktxvalidator "github.com/hyperledger/fabric/core/mocks/txvalidator"
	"github.com/hyperledger/fabric/core/scc/lscc"
	supportmocks "github.com/hyperledger/fabric/discovery/support/mocks"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const ccVersion = "1.0"

var signer msp.SigningIdentity

var signerSerialized []byte

func signedByAnyMember(ids []string) []byte {
	p := policydsl.SignedByAnyMember(ids)
	return protoutil.MarshalOrPanic(&peer.ApplicationPolicy{Type: &peer.ApplicationPolicy_SignaturePolicy{SignaturePolicy: p}})
}

func v20Capabilities() *tmocks.ApplicationCapabilities {
	ac := &tmocks.ApplicationCapabilities{}
	ac.On("V1_2Validation").Return(true)
	ac.On("V1_3Validation").Return(true)
	ac.On("V2_0Validation").Return(true)
	ac.On("PrivateChannelData").Return(true)
	ac.On("KeyLevelEndorsement").Return(true)
	return ac
}

func createRWset(t *testing.T, ccnames ...string) []byte {
	rwsetBuilder := rwsetutil.NewRWSetBuilder()
	for _, ccname := range ccnames {
		rwsetBuilder.AddToWriteSet(ccname, "key", []byte("value"))
	}
	rwset, err := rwsetBuilder.GetTxSimulationResults()
	require.NoError(t, err)
	rwsetBytes, err := rwset.GetPubSimulationBytes()
	require.NoError(t, err)
	return rwsetBytes
}

func getProposalWithType(ccID string, pType common.HeaderType) (*peer.Proposal, error) {
	cis := &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{Name: ccID, Version: ccVersion},
			Input:       &peer.ChaincodeInput{Args: [][]byte{[]byte("func")}},
			Type:        peer.ChaincodeSpec_GOLANG}}

	proposal, _, err := protoutil.CreateProposalFromCIS(pType, "testchannelid", cis, signerSerialized)
	return proposal, err
}

func getEnvWithType(ccID string, event []byte, res []byte, pType common.HeaderType, t *testing.T) *common.Envelope {
	// get a toy proposal
	prop, err := getProposalWithType(ccID, pType)
	require.NoError(t, err)

	response := &peer.Response{Status: 200}

	// endorse it to get a proposal response
	presp, err := protoutil.CreateProposalResponse(prop.Header, prop.Payload, response, res, event, &peer.ChaincodeID{Name: ccID, Version: ccVersion}, signer)
	require.NoError(t, err)

	// assemble a transaction from that proposal and endorsement
	tx, err := protoutil.CreateSignedTx(prop, signer, presp)
	require.NoError(t, err)

	return tx
}

func getEnvWithSigner(ccID string, event []byte, res []byte, sig msp.SigningIdentity, t *testing.T) *common.Envelope {
	// get a toy proposal
	pType := common.HeaderType_ENDORSER_TRANSACTION
	cis := &peer.ChaincodeInvocationSpec{
		ChaincodeSpec: &peer.ChaincodeSpec{
			ChaincodeId: &peer.ChaincodeID{Name: ccID, Version: ccVersion},
			Input:       &peer.ChaincodeInput{Args: [][]byte{[]byte("func")}},
			Type:        peer.ChaincodeSpec_GOLANG,
		},
	}

	sID, err := sig.Serialize()
	require.NoError(t, err)
	prop, _, err := protoutil.CreateProposalFromCIS(pType, "foochain", cis, sID)
	require.NoError(t, err)

	response := &peer.Response{Status: 200}

	// endorse it to get a proposal response
	presp, err := protoutil.CreateProposalResponse(prop.Header, prop.Payload, response, res, event, &peer.ChaincodeID{Name: ccID, Version: ccVersion}, sig)
	require.NoError(t, err)

	// assemble a transaction from that proposal and endorsement
	tx, err := protoutil.CreateSignedTx(prop, sig, presp)
	require.NoError(t, err)

	return tx
}

func getEnv(ccID string, event []byte, res []byte, t *testing.T) *common.Envelope {
	return getEnvWithType(ccID, event, res, common.HeaderType_ENDORSER_TRANSACTION, t)
}

func setupValidator() (*txvalidatorv20.TxValidator, *txvalidatormocks.QueryExecutor, *supportmocks.Identity, *txvalidatormocks.CollectionResources) {
	mspmgr := &supportmocks.MSPManager{}
	mockID := &supportmocks.Identity{}
	mockID.SatisfiesPrincipalReturns(nil)
	mockID.GetIdentifierReturns(&msp.IdentityIdentifier{})
	mspmgr.DeserializeIdentityReturns(mockID, nil)

	return setupValidatorWithMspMgr(mspmgr, mockID)
}

func setupValidatorWithMspMgr(mspmgr msp.MSPManager, mockID *supportmocks.Identity) (*txvalidatorv20.TxValidator, *txvalidatormocks.QueryExecutor, *supportmocks.Identity, *txvalidatormocks.CollectionResources) {
	pm := &plugindispatchermocks.Mapper{}
	factory := &plugindispatchermocks.PluginFactory{}
	pm.On("FactoryByName", txvalidatorplugin.Name("vscc")).Return(factory)
	factory.On("New").Return(&builtin.DefaultValidation{})

	mockQE := &txvalidatormocks.QueryExecutor{}
	mockQE.On("Done").Return(nil)
	mockQE.On("GetState", "lscc", "lscc").Return(nil, nil)
	mockQE.On("GetState", "lscc", "escc").Return(nil, nil)

	mockLedger := &txvalidatormocks.LedgerResources{}
	mockLedger.On("TxIDExists", mock.Anything).Return(false, nil)
	mockLedger.On("NewQueryExecutor").Return(mockQE, nil)

	mockCpmg := &plugindispatchermocks.ChannelPolicyManagerGetter{}
	mockCpmg.On("Manager", mock.Anything).Return(&txvalidatormocks.PolicyManager{})

	mockCR := &txvalidatormocks.CollectionResources{}

	cryptoProvider, _ := sw.NewDefaultSecurityLevelWithKeystore(sw.NewDummyKeyStore())
	v := txvalidatorv20.NewTxValidator(
		"",
		semaphore.New(10),
		&mocktxvalidator.Support{ACVal: v20Capabilities(), MSPManagerVal: mspmgr},
		mockLedger,
		&lscc.SCC{BCCSP: cryptoProvider},
		mockCR,
		pm,
		mockCpmg,
		cryptoProvider,
	)

	return v, mockQE, mockID, mockCR
}

func TestInvokeOK(t *testing.T) {
	ccID := "mycc"

	v, mockQE, _, _ := setupValidator()

	mockQE.On("GetState", "lscc", ccID).Return(protoutil.MarshalOrPanic(&ccp.ChaincodeData{
		Name:    ccID,
		Version: ccVersion,
		Vscc:    "vscc",
		Policy:  signedByAnyMember([]string{"SampleOrg"}),
	}), nil)
	mockQE.On("GetStateMetadata", ccID, "key").Return(nil, nil)

	tx := getEnv(ccID, nil, createRWset(t, ccID), t)
	b := &common.Block{Data: &common.BlockData{Data: [][]byte{protoutil.MarshalOrPanic(tx)}}, Header: &common.BlockHeader{Number: 2}}

	err := v.Validate(b)
	require.NoError(t, err)
	// assertValid(b, t)
}
