package worker

import (
	"context"
	"errors"
	"fmt"
	wallet2 "github.com/dapplink-labs/multichain-sync-account/synchronizer/wallet-chain-node/wallet"
	"time"

	"github.com/dapplink-labs/multichain-sync-account/common/tasks"
	"github.com/dapplink-labs/multichain-sync-account/config"
	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/ethereum/go-ethereum/log"
)

type TxManager struct {
	db             *database.DB
	chainNodeConf  *config.ChainNodeConfig
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	// 定时任务
	ticker *time.Ticker
}

func NewTxManager(cfg *config.Config, db *database.DB, rpcClient wallet2.WalletServiceClient, shutdown context.CancelCauseFunc) (*TxManager, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &TxManager{
		db:             db,
		chainNodeConf:  &cfg.ChainNode,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in tx manager: %w", err))
		}},
		ticker: time.NewTicker(time.Second * 5),
	}, nil
}

func (tm *TxManager) Close() error {
	var result error
	tm.resourceCancel()
	tm.ticker.Stop()
	log.Info("stop txmanager......")
	if err := tm.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await deposit %w"), err)
		return result
	}
	log.Info("stop txmanager success")
	return nil
}

func (tm *TxManager) Start() error {
	log.Info("start tx manager......")
	tm.tasks.Go(func() error {
		for {
			select {
			case <-tm.ticker.C:
				log.Info("start tx manager in worker")
			case <-tm.resourceCtx.Done():
				log.Info("stop tx manager in worker")
				return nil
			}
		}
	})
	return nil
}
