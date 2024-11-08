package synchronizer

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
	"time"

	"github.com/dapplink-labs/multichain-sync-account/common/tasks"
	"github.com/dapplink-labs/multichain-sync-account/config"
	"github.com/dapplink-labs/multichain-sync-account/database"
)

// GetStartHeight 获取开始扫描的区块高度
func GetStartHeight(cfg *config.Config, db *database.DB) *big.Int {
	var lastScannedBlock *big.Int
	// 获取DB里最新块高
	latestHeader, err := db.Blocks.LatestBlocks()
	if err != nil {
		log.Error("get latest block from database fail: ", "error", err)
	}
	if latestHeader != nil {
		// 数据库最新块高不为空那么就用数据库里的最新块高扫链
		log.Info("sync detected latest index block", "number")
	} else if cfg.ChainNode.StartingHeight > 0 {
		// 如果配置的起始块高不为0，那就按照配置的最新高度开始扫描
		lastScannedBlock = big.NewInt(int64(cfg.ChainNode.StartingHeight))
	}
	return lastScannedBlock
}

type Synchronizer struct {
	db               *database.DB
	chainNodeConf    *config.ChainNodeConfig
	resourceCtx      context.Context
	resourceCancel   context.CancelFunc
	tasks            tasks.Group
	lastScannedBlock *big.Int
	ticker           *time.Ticker
}

// NewSynchronizer 创建同步器
func NewSynchronizer(cfg *config.Config, db *database.DB, shutdown context.CancelCauseFunc) (*Synchronizer, error) {
	resCtx, resCancel := context.WithCancel(context.Background())
	return &Synchronizer{
		db:             db,
		chainNodeConf:  &cfg.ChainNode,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in synchronizer: %w", err))
		}},

		lastScannedBlock: GetStartHeight(cfg, db),
		ticker:           time.NewTicker(time.Second * 5),
	}, nil
}

func (sync *Synchronizer) Close() error {
	var result error
	sync.resourceCancel()
	sync.ticker.Stop()
	log.Info("stop synchronizer......")
	if err := sync.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await synchronizer %w"), err)
		return result
	}
	log.Info("stop synchronizer success")
	return nil
}

func (sync *Synchronizer) Start() error {
	log.Info("start synchronizer......")
	sync.tasks.Go(func() error {
		for {
			select {
			case <-sync.ticker.C:
				log.Info("start task")
			case <-sync.resourceCtx.Done():
				log.Info("stop synchronizer scan chain in worker")
				return nil
			}
		}
	})
	return nil
}
