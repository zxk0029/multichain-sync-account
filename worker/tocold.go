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

type ToCold struct {
	db             *database.DB
	chainNodeConf  *config.ChainNodeConfig
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker
}

func NewToCold(cfg *config.Config, db *database.DB, shutdown context.CancelCauseFunc) (*ToCold, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &ToCold{
		db:             db,
		chainNodeConf:  &cfg.ChainNode,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in to cold: %w", err))
		}},
		ticker: time.NewTicker(time.Second * 5),
	}, nil
}

func (tm *ToCold) Close() error {
	var result error
	tm.resourceCancel()
	tm.ticker.Stop()
	log.Info("stop to cold......")
	if err := tm.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await to cold %w"), err)
		return result
	}
	log.Info("stop to cold success")
	return nil
}

func (tm *ToCold) Start() error {
	log.Info("start to cold......")
	tm.tasks.Go(func() error {
		for {
			select {
			case <-tm.ticker.C:
				log.Info("start to cold in worker")
			case <-tm.resourceCtx.Done():
				log.Info("stop to cold in worker")
				return nil
			}
		}
	})
	return nil
}
