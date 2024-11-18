package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dapplink-labs/multichain-sync-account/common/tasks"
	"github.com/dapplink-labs/multichain-sync-account/config"
	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/ethereum/go-ethereum/log"
)

type Deposit struct {
	db             *database.DB
	chainNodeConf  *config.ChainNodeConfig
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker
}

func NewDeposit(cfg *config.Config, db *database.DB, shutdown context.CancelCauseFunc) (*Deposit, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &Deposit{
		db:             db,
		chainNodeConf:  &cfg.ChainNode,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in deposit: %w", err))
		}},
		ticker: time.NewTicker(time.Second * 5),
	}, nil
}

func (tm *Deposit) Close() error {
	var result error
	tm.resourceCancel()
	tm.ticker.Stop()
	log.Info("stop to cold......")
	if err := tm.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await deposit%w"), err)
		return result
	}
	log.Info("stop deposit success")
	return nil
}

func (tm *Deposit) Start() error {
	log.Info("start to cold......")
	tm.tasks.Go(func() error {
		for {
			select {
			case <-tm.ticker.C:
				log.Info("start deposit in worker")
			case <-tm.resourceCtx.Done():
				log.Info("stop deposit to worker")
				return nil
			}
		}
	})
	return nil
}
