/*
Copyright IBM Corp. All Rights Reserved.
Copyright 2020 Intel Corporation

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-private-chaincode/samples/application/simple-cli-go/pkg"
)

func initConfig() *pkg.Config {

	getStrEnv := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			panic(fmt.Sprintf("%s not set", key))
		}
		return val
	}

	getBoolEnv := func(key string) bool {
		val := getStrEnv(key)
		ret, err := strconv.ParseBool(val)
		if err != nil {
			if val == "" {
				panic(fmt.Sprintf("invalid bool value for %s", key))
			}
		}
		return ret
	}

	config := &pkg.Config{
		CorePeerAddress:         getStrEnv("CORE_PEER_ADDRESS"),
		CorePeerId:              getStrEnv("CORE_PEER_ID"),
		CorePeerLocalMSPID:      getStrEnv("CORE_PEER_LOCALMSPID"),
		CorePeerMSPConfigPath:   getStrEnv("CORE_PEER_MSPCONFIGPATH"),
		CorePeerTLSCertFile:     getStrEnv("CORE_PEER_TLS_CERT_FILE"),
		CorePeerTLSEnabled:      getBoolEnv("CORE_PEER_TLS_ENABLED"),
		CorePeerTLSKeyFile:      getStrEnv("CORE_PEER_TLS_KEY_FILE"),
		CorePeerTLSRootCertFile: getStrEnv("CORE_PEER_TLS_ROOTCERT_FILE"),
		OrdererCA:               getStrEnv("ORDERER_CA"),
		ChaincodeId:             getStrEnv("CC_NAME"),
		ChannelId:               getStrEnv("CHANNEL_NAME"),
		GatewayConfigPath:       getStrEnv("GATEWAY_CONFIG"),
	}
	return config
}

func main() {

	config := initConfig()

	client := pkg.NewClient(config)
	// res := client.Invoke("initSecretKeeper")
	// fmt.Println("> " + res)

	res := client.Query("revealSecret", "Alice")
	fmt.Println("> " + res)

	res = client.Invoke("LockSecret", "Bob", "NewSecret2")
	fmt.Println("> " + res)

	start := time.Now()
	for i := 0; i < 10; i++ {
		res = client.Invoke("revealSecret", "Alice")
		fmt.Println("> " + res)
	}
	fmt.Println(time.Since(start))
}
