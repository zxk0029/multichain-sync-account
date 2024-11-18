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

type Collection struct {
	db             *database.DB
	chainNodeConf  *config.ChainNodeConfig
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker
}

func NewCollection(cfg *config.Config, db *database.DB, shutdown context.CancelCauseFunc) (*Collection, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &Collection{
		db:             db,
		chainNodeConf:  &cfg.ChainNode,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in collection: %w", err))
		}},
		ticker: time.NewTicker(time.Second * 5),
	}, nil
}

func (tm *Collection) Close() error {
	var result error
	tm.resourceCancel()
	tm.ticker.Stop()
	log.Info("stop to cold......")
	if err := tm.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await collection%w"), err)
		return result
	}
	log.Info("stop Collection success")
	return nil
}

func (tm *Collection) Start() error {
	log.Info("start to cold......")
	tm.tasks.Go(func() error {
		for {
			select {
			case <-tm.ticker.C:
				log.Info("start collection in worker")
			case <-tm.resourceCtx.Done():
				log.Info("stop collection to worker")
				return nil
			}
		}
	})
	return nil
}
