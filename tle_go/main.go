package main

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func PrintConfig() {
	fmt.Println("--- viper config ---")
	settings := viper.AllSettings()
	for key, value := range settings {
		fmt.Printf("%s: %v\n", key, value)
	}
	fmt.Println("--finish viper config--")
}

func SetupConfig(configPath string) {
	viper.SetConfigFile(configPath)
	viper.ReadInConfig()
	viper.SetEnvPrefix("core")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
}

func main() {
	fmt.Println("--- in TLE_go main.go start to create grpc server.---")
	tlestate := &Tlestate{}
	// read config
	configPath := "/Users/lew/go/src/github.com/hyperledger/fabric-private-chaincode/samples/deployment/fabric-smart-client/the-simple-testing-network/testdata/fabric.default/peers/Org1.Org1_peer_0/core.yaml"
	SetupConfig(configPath)

	// serve block listener.
	go ServePeer(tlestate)

	// serve metadata service.
	ServeMeta("127.0.0.1:50051", tlestate)
}
