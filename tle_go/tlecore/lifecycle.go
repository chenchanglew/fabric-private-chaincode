package tlecore

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/aclmgmt"
	"github.com/hyperledger/fabric/core/chaincode/lifecycle"
	"github.com/hyperledger/fabric/core/chaincode/persistence"
	"github.com/hyperledger/fabric/core/committer/txvalidator/v20/plugindispatcher"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	coreconfig "github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/core/container"
	"github.com/hyperledger/fabric/core/container/externalbuilder"
	"github.com/hyperledger/fabric/core/ledger"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/policy"
	"github.com/hyperledger/fabric/core/scc/lscc"
	gossipprivdata "github.com/hyperledger/fabric/gossip/privdata"
	"github.com/hyperledger/fabric/msp/mgmt"
)

// externalVMAdapter adapts coerces the result of Build to the
// container.Interface type expected by the VM interface.
type externalVMAdapter struct {
	detector *externalbuilder.Detector
}

func (e externalVMAdapter) Build(
	ccid string,
	mdBytes []byte,
	codePackage io.Reader,
) (container.Instance, error) {
	i, err := e.detector.Build(ccid, mdBytes, codePackage)
	if err != nil {
		return nil, err
	}

	// ensure <nil> is returned instead of (*externalbuilder.Instance)(nil)
	if i == nil {
		return nil, nil
	}
	return i, err
}

type disabledDockerBuilder struct{}

func (disabledDockerBuilder) Build(string, *persistence.ChaincodePackageMetadata, io.Reader) (container.Instance, error) {
	return nil, errors.New("docker build is disabled")
}

func createLifecycleValidation(peerInstance *peer.Peer) (plugindispatcher.LifecycleResources, plugindispatcher.CollectionAndLifecycleResources, ledger.DeployedChaincodeInfoProvider) {
	//obtain coreConfiguration
	coreConfig, err := peer.GlobalConfig()
	if err != nil {
		fmt.Printf("obtain coreConfiguration failed, err: %s\n", err)
		panic("!!!err!!!")
	}

	mspID := coreConfig.LocalMSPID
	chaincodeInstallPath := filepath.Join(coreconfig.GetPath("peer.fileSystemPath"), "lifecycle", "chaincodes")

	externalBuilderOutput := filepath.Join(coreconfig.GetPath("peer.fileSystemPath"), "externalbuilder", "builds")
	err = os.MkdirAll(externalBuilderOutput, 0700)
	if err != nil {
		fmt.Printf("could not create externalbuilder build output dir: %s\n", err)
		panic("!!err!!")
	}

	builtinSCCs := map[string]struct{}{
		"lscc":       {},
		"qscc":       {},
		"cscc":       {},
		"_lifecycle": {},
	}

	policyChecker := policy.NewPolicyChecker(
		policies.PolicyManagerGetterFunc(peerInstance.GetPolicyManager),
		mgmt.GetLocalMSP(factory.GetDefault()),
		mgmt.NewLocalMSPPrincipalGetter(factory.GetDefault()),
	)

	aclProvider := aclmgmt.NewACLProvider(
		aclmgmt.ResourceGetter(peerInstance.GetStableChannelConfig),
		policyChecker,
	)

	buildRegistry := &container.BuildRegistry{}
	dockerBuilder := &disabledDockerBuilder{}

	externalVM := &externalbuilder.Detector{
		Builders:    externalbuilder.CreateBuilders(coreConfig.ExternalBuilders, mspID),
		DurablePath: externalBuilderOutput,
	}

	containerRouter := &container.Router{
		DockerBuilder:   dockerBuilder,
		ExternalBuilder: externalVMAdapter{externalVM},
		PackageProvider: &persistence.FallbackPackageLocator{
			ChaincodePackageLocator: &persistence.ChaincodePackageLocator{
				ChaincodeDir: chaincodeInstallPath,
			},
			LegacyCCPackageLocator: &ccprovider.CCInfoFSImpl{GetHasher: factory.GetDefault()},
		},
	}

	ebMetadataProvider := &externalbuilder.MetadataProvider{
		DurablePath: externalBuilderOutput,
	}

	lsccInst := &lscc.SCC{
		BuiltinSCCs: builtinSCCs,
		Support: &lscc.SupportImpl{
			GetMSPIDs: peerInstance.GetMSPIDs,
		},
		SCCProvider:        &lscc.PeerShim{Peer: peerInstance},
		ACLProvider:        aclProvider,
		GetMSPIDs:          peerInstance.GetMSPIDs,
		PolicyChecker:      policyChecker,
		BCCSP:              factory.GetDefault(),
		BuildRegistry:      buildRegistry,
		ChaincodeBuilder:   containerRouter,
		EbMetadataProvider: ebMetadataProvider,
	}

	fmt.Println("--- successfully generate lsccInst ---")

	ccStore := persistence.NewStore(chaincodeInstallPath)
	ccPackageParser := &persistence.ChaincodePackageParser{
		MetadataProvider: ccprovider.PersistenceAdapter(ccprovider.MetadataAsTarEntries),
	}

	lifecycleResources := &lifecycle.Resources{
		Serializer:          &lifecycle.Serializer{},
		ChannelConfigSource: peerInstance,
		ChaincodeStore:      ccStore,
		PackageParser:       ccPackageParser,
	}

	privdataConfig := gossipprivdata.GlobalConfig()
	lifecycleValidatorCommitter := &lifecycle.ValidatorCommitter{
		CoreConfig:                   coreConfig,
		PrivdataConfig:               privdataConfig,
		Resources:                    lifecycleResources,
		LegacyDeployedCCInfoProvider: &lscc.DeployedCCInfoProvider{},
	}

	fmt.Println("--- successfully generate LifecycleValidatorCommitter ---")

	return lsccInst, lifecycleValidatorCommitter, lifecycleValidatorCommitter
}
