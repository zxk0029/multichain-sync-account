package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-transaction-syncs/common/tasks"
	"github.com/dapplink-labs/multichain-transaction-syncs/config"
	"github.com/dapplink-labs/multichain-transaction-syncs/database"
)

type TxManager struct {
	db             *database.DB
	chainNodeConf  *config.ChainNodeConfig
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewTxManager(cfg *config.Config, db *database.DB, shutdown context.CancelCauseFunc) (*TxManager, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &TxManager{
		db:             db,
		chainNodeConf:  &cfg.ChainNode,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in tx manager: %w", err))
		}},
	}, nil
}

func (tm *TxManager) Close() error {
	var result error
	tm.resourceCancel()
	if err := tm.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await deposit %w"), err)
		return result
	}
	return nil
}

func (tm *TxManager) Start() error {
	log.Info("start tx manager......")
	tickerDepositWorker := time.NewTicker(time.Second * 5)
	tm.tasks.Go(func() error {
		for range tickerDepositWorker.C {
			log.Info("start tx manager in worker")
		}
		return nil
	})
	return nil
}
