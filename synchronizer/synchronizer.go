package synchronizer

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

type Synchronizer struct {
	db             *database.DB
	chainConf      *config.ChainConfig
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewSynchronizer(cfg *config.Config, db *database.DB, shutdown context.CancelCauseFunc) (*Synchronizer, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &Synchronizer{
		db:             db,
		chainConf:      &cfg.Chain,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in synchronizer: %w", err))
		}},
	}, nil
}

func (sync *Synchronizer) Close() error {
	var result error
	sync.resourceCancel()
	if err := sync.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await synchronizer %w"), err)
		return result
	}
	return nil
}

func (sync *Synchronizer) Start() error {
	log.Info("start synchronizer......")
	tickerDepositWorker := time.NewTicker(time.Second * 5)
	sync.tasks.Go(func() error {
		for range tickerDepositWorker.C {
			log.Info("start synchronizer")
		}
		return nil
	})
	return nil
}
