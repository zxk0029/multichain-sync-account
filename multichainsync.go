package multichain_transaction_syncs

import (
	"context"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-transaction-syncs/config"
	"github.com/dapplink-labs/multichain-transaction-syncs/database"
	"github.com/dapplink-labs/multichain-transaction-syncs/synchronizer"
	"github.com/dapplink-labs/multichain-transaction-syncs/worker"
)

type MultiChainSync struct {
	txManager    *worker.TxManager
	synchronizer *synchronizer.Synchronizer

	shutdown context.CancelCauseFunc
	stopped  atomic.Bool
}

func NewMultiChainSync(ctx context.Context, cfg *config.Config, shutdown context.CancelCauseFunc) (*MultiChainSync, error) {
	db, err := database.NewDB(ctx, cfg.MasterDB)
	if err != nil {
		log.Error("init database fail", err)
		return nil, err
	}

	txManager, _ := worker.NewTxManager(cfg, db, shutdown)
	_synchronizer, _ := synchronizer.NewSynchronizer(cfg, db, shutdown)

	out := &MultiChainSync{
		txManager:    txManager,
		synchronizer: _synchronizer,
		shutdown:     shutdown,
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
