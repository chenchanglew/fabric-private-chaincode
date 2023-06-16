package tlecore

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"strconv"
	"sync"
	"time"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/committer"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/protoutil"
)

type TlePeer struct {
	tleState     *Tlestate
	nextBlockNum uint
	mutex        sync.Mutex

	channelName string
	lc          *committer.LedgerCommitter
	policyMgr   policies.PolicyManagerGetterFunc
	validator   *txvalidator.ValidationRouter
}

func (p *TlePeer) vsccExtractRwsetRaw(block *common.Block, txPosition int, actionPosition int) ([]byte, error) {
	// get the envelope...
	env, err := protoutil.GetEnvelopeFromBlock(block.Data.Data[txPosition])
	if err != nil {
		err = fmt.Errorf("VSCC error: GetEnvelope failed, err %s", err)
		return nil, err
	}

	// ...and the payload...
	payl, err := protoutil.UnmarshalPayload(env.Payload)
	if err != nil {
		err = fmt.Errorf("VSCC error: GetPayload failed, err %s", err)
		return nil, err
	}

	tx, err := protoutil.UnmarshalTransaction(payl.Data)
	if err != nil {
		err = fmt.Errorf("VSCC error: GetTransaction failed, err %s", err)
		return nil, err
	}

	cap, err := protoutil.UnmarshalChaincodeActionPayload(tx.Actions[actionPosition].Payload)
	if err != nil {
		err = fmt.Errorf("VSCC error: GetChaincodeActionPayload failed, err %s", err)
		return nil, err
	}

	pRespPayload, err := protoutil.UnmarshalProposalResponsePayload(cap.Action.ProposalResponsePayload)
	if err != nil {
		err = fmt.Errorf("GetProposalResponsePayload error %s", err)
		return nil, err
	}
	if pRespPayload.Extension == nil {
		err = fmt.Errorf("nil pRespPayload.Extension")
		return nil, err
	}
	respPayload, err := protoutil.UnmarshalChaincodeAction(pRespPayload.Extension)
	if err != nil {
		err = fmt.Errorf("GetChaincodeAction error %s", err)
		return nil, err
	}
	return respPayload.Results, nil
}

func (p *TlePeer) UpdateState(block *common.Block) error {
	for tIdx, _ := range block.Data.Data {
		// TODO: continue if current txn is invalid.
		txsfltr := ValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
		fmt.Printf("blockNum %d, tIdx: %d, validationCode: %v\n", tIdx, block.Header.Number, txsfltr.Flag(tIdx))
		if txsfltr.IsInvalid(tIdx) {
			fmt.Println("The current txn is not valid!")
			// continue
		}

		// extract rwset
		rwsetRaw, err := p.vsccExtractRwsetRaw(block, tIdx, 0)
		if err != nil {
			fmt.Printf("failed to generate ReadWriteSetRaw for tIdx: %d, blockNum: %d\n", tIdx, block.Header.Number)
			continue
		}
		rwset := &rwsetutil.TxRwSet{}
		if err := rwset.FromProtoBytes(rwsetRaw); err != nil {
			fmt.Printf("failed to generate ReadWriteSet for tIdx: %d, blockNum: %d\n", tIdx, block.Header.Number)
			continue
		}

		for _, nsRWSet := range rwset.NsRwSets {
			fmt.Printf("namespace: %v, read: %v, writes: %v\n", nsRWSet.NameSpace, nsRWSet.KvRwSet.Reads, nsRWSet.KvRwSet.Writes)

			// Q: can we use Metadata?
			// metadataWriteSet := nsRwset.KvRwSet.MetadataWrites

			for _, kvWrite := range nsRWSet.KvRwSet.Writes {
				metaData := sha256.Sum256(kvWrite.Value)

				fmt.Printf("Saving namespace: %v, key: %v, metadata: %x\n", nsRWSet.NameSpace, kvWrite.Key, metaData)

				// Update tleState using PutMeta
				err := p.tleState.PutMeta(nsRWSet.NameSpace, kvWrite.Key, metaData[:])
				if err != nil {
					fmt.Printf("Error updating tleState: %v\n", err)
				}
			}
		}
	}
	return nil
}

func (p *TlePeer) GetBlock() (*common.Block, error) {
	// Simulating data retrieval from somewhere
	fmt.Printf("Start to get block num: %d\n", p.GetNextBlockNum())
	rawBlock, err := ioutil.ReadFile("tmpBlocks/t" + strconv.Itoa(int(p.GetNextBlockNum())) + ".block")
	if err != nil {
		return nil, err
	}
	return protoutil.UnmarshalBlock(rawBlock)
}

func (p *TlePeer) ProcessBlock(block *common.Block) error {
	err := VerifyBlock(p.policyMgr, []byte(p.channelName), uint64(p.GetNextBlockNum()), block)
	if err != nil {
		return err
	}
	fmt.Printf("--- Verify Block %d success, start verify txn ---\n", p.GetNextBlockNum())
	err = p.validator.Validate(block)
	if err != nil {
		return err
	}

	err = StoreBlock(p.lc, block)
	if err != nil {
		return err
	}

	// update state
	p.IncrementNextBlockNum()
	return p.UpdateState(block)
}

func (p *TlePeer) GetNextBlockNum() uint {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.nextBlockNum
}

func (p *TlePeer) IncrementNextBlockNum() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.nextBlockNum += 1
}

func (p *TlePeer) InitFabricPart() func() {
	genesisBlock := GetGenesisBlock()
	peerInstance, cleanup := peer.NewFabricPeer()

	err := InitializeFabricPeer(peerInstance)
	if err != nil {
		panic(err)
	}
	legacyLifecycleValidation, NewLifecycleValidation, dcip := createLifecycleValidation(peerInstance)
	channelName := "testchannel"

	err = peerInstance.CreateChannel(channelName, genesisBlock, dcip, legacyLifecycleValidation, NewLifecycleValidation)
	if err != nil {
		panic(err)
	}
	ledger := peerInstance.Channel(channelName).Ledger()
	lc := committer.NewLedgerCommitter(ledger)
	policyMgr := policies.PolicyManagerGetterFunc(peerInstance.GetPolicyManager)
	validator, err := CreateTxValidatorViaPeer(peerInstance, channelName, legacyLifecycleValidation, NewLifecycleValidation)
	if err != nil {
		panic(err)
	}
	p.lc = lc
	p.policyMgr = policyMgr
	p.validator = validator
	p.channelName = channelName

	return cleanup
}

func (p *TlePeer) Start() {
	cleanup := p.InitFabricPart()
	defer cleanup()

	waitTime := 1
	for {
		// wait several second to update one block
		time.Sleep(time.Duration(waitTime) * time.Second)

		block, err := p.GetBlock()
		if err != nil {
			fmt.Printf("TlePeer GetBlock Failed, %v\n", err)
			waitTime = waitTime * 2
			continue
		}
		err = p.ProcessBlock(block)
		if err != nil {
			fmt.Printf("TlePeer Process Block error, %v\n", err)
		}
		waitTime = 1
	}
}

func GetGenesisBlock() *common.Block {
	// TODO get from somewhere else.
	rawBlock0, err := ioutil.ReadFile("tmpBlocks/t0.block")
	if err != nil {
		panic("read genesis block error")
	}
	fmt.Println("Finish reading genesis block!!!")
	genesisBlock, err := protoutil.UnmarshalBlock(rawBlock0)
	if err != nil {
		panic("Unmarshal genesis block error")
	}
	return genesisBlock
}

func ServePeer(tleState *Tlestate) {
	// TODO change the logic here.
	peer := &TlePeer{
		tleState:     tleState,
		nextBlockNum: 1,
	}
	go peer.Start()
}
