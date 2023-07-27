package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric-private-chaincode/tle_go/listener"
	"github.com/hyperledger/fabric-private-chaincode/tle_go/tlecore"
	tleconfig "github.com/hyperledger/fabric-private-chaincode/tle_go/tlecore/config"
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
	tleconfig.SetupConfig(configPath)
}

func main() {

	// serve block listener.
	channelID := "testchannel"
	serverAddr := "host.docker.internal:20000"
	caCertPath := "/project/src/github.com/hyperledger/fabric-private-chaincode/samples/deployment/fabric-smart-client/the-simple-testing-network/testdata/fabric.default/crypto/ca-certs.pem"
	// if fpc_path exist, which mean it is running in local machine.
	fpcPath := os.Getenv("FPC_PATH")
	if fpcPath != "" {
		serverAddr = "127.0.0.1:20000"
		caCertPath = filepath.Join(fpcPath, "samples/deployment/fabric-smart-client/the-simple-testing-network/testdata/fabric.default/crypto/ca-certs.pem")
	}
	blockListener := listener.NewOrdererBlockGetter(channelID, serverAddr, caCertPath)
	// blockListener := listener.NewFileBlockGetter()
	fmt.Scanln()

	// fmt.Println("--- in TLE_go main.go start to create grpc server.---")

	readConfig()
	tlestate := &tlecore.Tlestate{}
	go tlecore.ServePeer(tlestate, blockListener)

	// serve metadata service.
	tlecore.ServeMeta("0.0.0.0:50051", tlestate)
}
