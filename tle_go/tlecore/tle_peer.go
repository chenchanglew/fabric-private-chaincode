package tlecore

import (
	"crypto/sha256"
	"fmt"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/committer"
	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/protoutil"
)

type TlePeer struct {
	tleState      *Tlestate
	blockListener BlockListener

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
	for tIdx := range block.Data.Data {
		// TODO: continue if current txn is invalid.
		txsfltr := ValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
		fmt.Printf("blockNum %d, tIdx: %d, validationCode: %v\n", tIdx, block.Header.Number, txsfltr.Flag(tIdx))
		if txsfltr.IsInvalid(tIdx) {
			fmt.Println("The current txn is not valid!")
			continue
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

func (p *TlePeer) ProcessBlock(block *common.Block, blockNum int) error {
	err := VerifyBlock(p.policyMgr, []byte(p.channelName), uint64(blockNum), block)
	if err != nil {
		return err
	}
	fmt.Printf("--- Verify Block %d success, start verify txn ---\n", uint64(blockNum))
	err = p.validator.Validate(block)
	if err != nil {
		return err
	}

	err = StoreBlock(p.lc, block)
	if err != nil {
		return err
	}

	// update state
	return p.UpdateState(block)
}

func (p *TlePeer) InitFabricPart(genesisBlock *common.Block) func() {
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
	genesisBlock, err := p.blockListener.GetNextBlock()
	if err != nil {
		fmt.Println("Get genesis block failed: ", err)
	}
	p.blockListener.NotifySuccess()

	cleanup := p.InitFabricPart(genesisBlock)
	defer cleanup()

	for {
		blocknum := p.blockListener.GetNextBlockNum()
		block, err := p.blockListener.GetNextBlock()
		if err != nil {
			fmt.Printf("TlePeer GetBlock Failed, %v\n", err)
			continue
		}
		err = p.ProcessBlock(block, blocknum)
		if err != nil {
			fmt.Printf("TlePeer Process Block error, %v\n", err)
			continue
		}
		p.blockListener.NotifySuccess()
	}
}

func ServePeer(tleState *Tlestate) {
	// TODO change the logic here.
	blockListener := NewFileBlockGetter()
	// blockListener := NewOrdererBlockGetter()
	peer := &TlePeer{
		tleState:      tleState,
		blockListener: blockListener,
	}
	go peer.Start()
}
