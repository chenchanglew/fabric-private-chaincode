package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric-private-chaincode/tle_go/tlecore"
)

func readConfig() {
	// read config
	var configPath string
	fpcPath := os.Getenv("FPC_PATH")
	// if fpc_path exist, which mean it is running in local machine, else it is running in docker.
	if fpcPath != "" {
		configPath = filepath.Join(fpcPath, "samples/deployment/fabric-smart-client/the-simple-testing-network/testdata/fabric.default/peers/Org1.Org1_peer_0/core.yaml")
	} else {
		configPath = os.Getenv("CORE_CONFIG_PATH")
	}
	tlecore.SetupConfig(configPath)
}

func main() {
	fmt.Println("--- in TLE_go main.go start to create grpc server.---")
	readConfig()

	tlestate := &tlecore.Tlestate{}

	// serve block listener.
	go tlecore.ServePeer(tlestate)

	// serve metadata service.
	tlecore.ServeMeta("0.0.0.0:50051", tlestate)
}
