package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric-private-chaincode/tle_go/tlecore"
)

func main() {
	fmt.Println("--- in TLE_go main.go start to create grpc server.---")
	tlestate := &tlecore.Tlestate{}
	// read config
	fpcPath := os.Getenv("FPC_PATH")
	configPath := filepath.Join(fpcPath, "samples/deployment/fabric-smart-client/the-simple-testing-network/testdata/fabric.default/peers/Org1.Org1_peer_0/core.yaml")
	// configPath := "/Users/lew/go/src/github.com/hyperledger/fabric-private-chaincode/samples/deployment/fabric-smart-client/the-simple-testing-network/testdata/fabric.default/peers/Org1.Org1_peer_0/core.yaml"
	tlecore.SetupConfig(configPath)

	// serve block listener.
	go tlecore.ServePeer(tlestate)

	// serve metadata service.
	tlecore.ServeMeta("127.0.0.1:50051", tlestate)
}
