package listener

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/protoutil"
)

type BlockListener interface {
	GetNextBlockNum() int
	GetNextBlock() (*common.Block, error)
	NotifySuccess() error
}

type FileBlockGetter struct {
	nextBlockNum int
}

func NewFileBlockGetter() BlockListener {
	return &FileBlockGetter{
		nextBlockNum: 0,
	}
}

func (f *FileBlockGetter) GetNextBlockNum() int {
	return f.nextBlockNum
}

func (f *FileBlockGetter) GetNextBlock() (*common.Block, error) {
	// Simulating data retrieval from somewhere
	fmt.Printf("Start to get block num: %d\n", f.nextBlockNum)
	blockPath := os.Getenv("BLOCK_PATH")
	if blockPath == "" {
		blockPath = "tmpBlocks"
	}
	waitTime := 5
	maxWaitTime := 120
	for {
		rawBlock, err := ioutil.ReadFile(filepath.Join(blockPath, "t"+strconv.Itoa(int(f.nextBlockNum))+".block"))
		if err != nil {
			fmt.Printf("FileBlockGetter GetBlock Failed, %v, wait for a while...\n", err)

			time.Sleep(time.Duration(waitTime) * time.Second)
			waitTime += 5
			if waitTime > maxWaitTime {
				waitTime = maxWaitTime
			}
			continue
		}
		return protoutil.UnmarshalBlock(rawBlock)
	}
}

func (f *FileBlockGetter) NotifySuccess() error {
	f.nextBlockNum += 1
	return nil
}

type OrdererBlockGetter struct {
	blockChan    chan common.Block
	nextBlockNum int
}

func NewOrdererBlockGetter(channelID, serverAddr, caCertPath string) BlockListener {
	// channelID := "testchannel"
	// serverAddr := "127.0.0.1:20000"
	seek := -2    // -2 is load from oldest, -1 load from newest, other int -> load from the int.
	quiet := true // true = only print block number, false = print whole block.
	// caCertPath := "/Users/lew/go/src/github.com/hyperledger/fabric-private-chaincode/samples/deployment/fabric-smart-client/the-simple-testing-network/testdata/fabric.default/crypto/ca-certs.pem"
	blockChan := make(chan common.Block, 1)

	go ListenBlock(channelID, serverAddr, seek, quiet, caCertPath, blockChan)

	return &OrdererBlockGetter{
		blockChan:    blockChan,
		nextBlockNum: 0,
	}
}

func (o *OrdererBlockGetter) GetNextBlockNum() int {
	return o.nextBlockNum
}

func (o *OrdererBlockGetter) GetNextBlock() (*common.Block, error) {
	// TODO will need to get the same block if previous failed.
	waitTime := 5
	maxWaitTime := 120
	for {
		select {
		case block := <-o.blockChan:
			return &block, nil
		case <-time.After(time.Duration(waitTime) * time.Second):
			fmt.Println("Still waiting for new block: block", o.nextBlockNum)
		}
		waitTime += 5
		if waitTime > maxWaitTime {
			waitTime = maxWaitTime
		}
	}
}

func (o *OrdererBlockGetter) NotifySuccess() error {
	o.nextBlockNum += 1
	return nil
}
