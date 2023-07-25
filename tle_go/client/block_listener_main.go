package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hyperledger/fabric-config/protolator"
	"github.com/hyperledger/fabric-private-chaincode/tle_go/listener"
	"github.com/hyperledger/fabric-protos-go/common"
)

func listener_main() {
	channelID := "testchannel"
	serverAddr := "127.0.0.1:20000"
	seek := -2    // -2 is load from oldest, -1 load from newest, other int -> load from the int.
	quiet := true // true = only print block number, false = print whole block.
	caCertPath := "/Users/lew/go/src/github.com/hyperledger/fabric-private-chaincode/samples/deployment/fabric-smart-client/the-simple-testing-network/testdata/fabric.default/crypto/ca-certs.pem"
	blockChan := make(chan common.Block, 1)

	go listener.ListenBlock(channelID, serverAddr, seek, quiet, caCertPath, blockChan)

	for {
		select {
		case block := <-blockChan:
			fmt.Println("Receiving Block:")
			fmt.Scanln()
			err := protolator.DeepMarshalJSON(os.Stdout, &block)
			if err != nil {
				fmt.Printf("  Error pretty printing block: %s", err)
			}
		case <-time.After(5 * time.Second):
			fmt.Println("Still waiting for new block...")
		}
	}
}
