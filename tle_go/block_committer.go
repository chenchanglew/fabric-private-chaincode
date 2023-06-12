package main

import (
	"errors"
	"fmt"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/core/committer"
	"github.com/hyperledger/fabric/core/ledger"
)

func StoreBlock(lc *committer.LedgerCommitter, block *common.Block) error {
	if block.Data == nil {
		return errors.New("Block data is empty")
	}
	if block.Header == nil {
		return errors.New("Block header is nil")
	}

	blockAndPvtData := &ledger.BlockAndPvtData{
		Block:          block,
		PvtData:        make(ledger.TxPvtDataMap),
		MissingPvtData: make(ledger.TxMissingPvtData),
	}

	err := CommitLegacy(lc, blockAndPvtData, &ledger.CommitOptions{})
	if err != nil {
		return errors.New(fmt.Sprintf("commit failed with error %s", err))
	}
	return err
}

func CommitLegacy(lc *committer.LedgerCommitter, blockAndPvtData *ledger.BlockAndPvtData, commitOpts *ledger.CommitOptions) error {
	// Committing new block
	if err := lc.PeerLedgerSupport.CommitLegacy(blockAndPvtData, commitOpts); err != nil {
		return err
	}
	return nil
}
