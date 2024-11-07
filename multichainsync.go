package multichain_transaction_syncs

import (
	"context"
	"github.com/dapplink-labs/multichain-sync-account/synchronizer/wallet-chain-node"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-account/config"
	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/dapplink-labs/multichain-sync-account/synchronizer"
	"github.com/dapplink-labs/multichain-sync-account/worker"
)

type MultiChainSync struct {
	txManager      *worker.TxManager
	synchronizer   *synchronizer.Synchronizer
	collectionCold *synchronizer.CollectionCold

	shutdown context.CancelCauseFunc
	stopped  atomic.Bool
}

func NewMultiChainSync(ctx context.Context, cfg *config.Config, shutdown context.CancelCauseFunc) (*MultiChainSync, error) {
	db, err := database.NewDB(ctx, cfg.MasterDB)
	if err != nil {
		log.Error("init database fail", err)
		return nil, err
	}
	rpcClient := wallet_chain_node.InitRpcClient(cfg.ChainNode.RpcUrl)
	txManager, _ := worker.NewTxManager(cfg, db, rpcClient, shutdown)
	_synchronizer, _ := synchronizer.NewSynchronizer(cfg, db, rpcClient, shutdown)
	collectionCold, _ := synchronizer.NewCollectionCold(cfg, db, rpcClient, shutdown)

	out := &MultiChainSync{
		txManager:      txManager,
		synchronizer:   _synchronizer,
		collectionCold: collectionCold,
		shutdown:       shutdown,
	}

	return out, nil
}

func (mcs *MultiChainSync) Start(ctx context.Context) error {
	err := mcs.txManager.Start()
	if err != nil {
		return err
	}
	err = mcs.synchronizer.Start()
	if err != nil {
		return err
	}
	return nil
}

func (mcs *MultiChainSync) Stop(ctx context.Context) error {
	err := mcs.txManager.Close()
	if err != nil {
		return err
	}
	err = mcs.synchronizer.Close()

	if err != nil {
		return err
	}
	return nil
}

func (mcs *MultiChainSync) Stopped() bool {
	return mcs.stopped.Load()
}
