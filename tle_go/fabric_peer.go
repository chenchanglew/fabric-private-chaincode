package main

import (
	"fmt"
	"runtime"

	"github.com/hyperledger/fabric/core/committer/txvalidator/plugin"
	"github.com/hyperledger/fabric/core/handlers/library"
	validation "github.com/hyperledger/fabric/core/handlers/validation/api"
	ledgermocks "github.com/hyperledger/fabric/core/ledger/mock"
	"github.com/hyperledger/fabric/core/peer"
)

func InitializeFabricPeer(peerInstance *peer.Peer) error {
	libConf, err := library.LoadConfig()
	if err != nil {
		return fmt.Errorf("could not decode peer handlers configuration [%s]", err)
	}

	reg := library.InitRegistry(libConf)
	validationPluginsByName := reg.Lookup(library.Validation).(map[string]validation.PluginFactory)
	peerInstance.Initialize(
		func(cid string) {
			fmt.Printf("--- peerInstance Init function with cid: %s ---\n", cid)
		},
		nil,
		plugin.MapBasedMapper(validationPluginsByName),
		&ledgermocks.DeployedChaincodeInfoProvider{},
		nil,
		nil,
		runtime.NumCPU(),
	)
	return nil
}
