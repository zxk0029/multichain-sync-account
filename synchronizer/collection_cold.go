package synchronizer

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-account/common/tasks"
	"github.com/dapplink-labs/multichain-sync-account/config"
	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/dapplink-labs/multichain-sync-account/synchronizer/wallet-chain-node/wallet"
)

var (
	CollectionFunding           = big.NewInt(10000000000000000)   // 0.01 ETH
	ColdFunding                 = big.NewInt(2000000000000000000) // 2 ETH
	EthGasLimit          uint64 = 21000
	TokenGasLimit        uint64 = 120000
	maxFeePerGas                = big.NewInt(2900000000) // 2.9 Gwei
	maxPriorityFeePerGas        = big.NewInt(2600000000) // 2.6 Gwei
)

type CollectionCold struct {
	db             *database.DB
	rpcClient      wallet.WalletServiceClient
	chainNodeConf  *config.ChainNodeConfig
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
	ticker         *time.Ticker
}

func NewCollectionCold(cfg *config.Config, db *database.DB, rpcClient wallet.WalletServiceClient, shutdown context.CancelCauseFunc) (*CollectionCold, error) {
	resCtx, resCancel := context.WithCancel(context.Background())

	return &CollectionCold{
		db:             db,
		chainNodeConf:  &cfg.ChainNode,
		rpcClient:      rpcClient,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		ticker:         time.NewTicker(5 * time.Second),
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in collection cold: %w", err))
		}},
	}, nil
}

func (cc *CollectionCold) Close() error {
	cc.resourceCancel()
	cc.ticker.Stop()
	log.Info("Stopping collection and cold tasks...")
	if err := cc.tasks.Wait(); err != nil {
		return fmt.Errorf("failed to await tasks completion: %w", err)
	}
	log.Info("Successfully stopped collection and cold tasks")
	return nil
}

func (cc *CollectionCold) Start() error {
	log.Info("Starting collection and cold tasks...")

	cc.tasks.Go(func() error {
		for {
			select {
			case <-cc.ticker.C:
				if err := cc.Collection(); err != nil {
					log.Error("Collection failed", "err", err)
				}
			case <-cc.resourceCtx.Done():
				log.Info("Stopping collection due to context cancellation")
				return nil
			}
		}
	})

	cc.tasks.Go(func() error {
		for {
			select {
			case <-cc.ticker.C:
				if err := cc.ToCold(); err != nil {
					log.Error("ToCold operation failed", "err", err)
				}
			case <-cc.resourceCtx.Done():
				log.Info("Stopping cold tasks due to context cancellation")
				return nil
			}
		}
	})

	return nil
}

func (cc *CollectionCold) ToCold() error {
	// Add logic for the "ToCold" functionality if needed
	return nil
}

// Collection 归集
func (cc *CollectionCold) Collection() error {
	//globalCache := cache.GetGlobalCache()
	//businessList, err := cc.db.Business.QueryBusinessAll()
	//if err != nil {
	//	log.Error("Failed to query business list", "err", err)
	//	return err
	//}
	//
	//for _, business := range businessList {
	//	unCollectionList, err := cc.db.Balances.UnCollectionList(business.BusinessUid, CollectionFunding)
	//	if err != nil {
	//		log.Error("Failed to query uncollected balances", "err", err)
	//		continue
	//	}
	//
	//	hotWalletInfo, err := cc.db.Addresses.QueryHotWalletInfo(business.BusinessUid)
	//	if err != nil {
	//		log.Error("Failed to query hot wallet info", "err", err)
	//		return err
	//	}
	//
	//	var txList []database.Transactions
	//	for _, uncollect := range unCollectionList {
	//		accountInfo, found := globalCache.Get(uncollect.Address.Hex())
	//		if !found {
	//			log.Error("Account info not found in cache", "address", uncollect.Address.Hex())
	//			continue
	//		}
	//
	//		txCountResp, err := cc.rpcClient.GetTxCountByAddress(context.Background(), &wallet.TxCountByAddressRequest{
	//			Chain:   cc.chainNodeConf.ChainName,
	//			Address: uncollect.Address.Hex(),
	//			Network: "mainnet",
	//		})
	//		if err != nil {
	//			log.Error("Failed to get transaction count by address", "err", err)
	//			continue
	//		}
	//
	//		nonce := txCountResp.Count
	//		var (
	//			buildData []byte
	//			gasLimit  uint64
	//			toAddress *common.Address
	//			amount    *big.Int
	//		)
	//
	//		if uncollect.TokenAddress.Hex() != "0x0000000000000000000000000000000000000000" {
	//			//buildData = ethereum.BuildErc20Data(hotWalletInfo.Address, uncollect.Balance)
	//			toAddress = &uncollect.TokenAddress
	//			gasLimit = TokenGasLimit
	//			amount = big.NewInt(0)
	//		} else {
	//			collectAmount := new(big.Int).Sub(uncollect.Balance, big.NewInt(1000000000000000)) // 0.001 ETH fee
	//			if collectAmount.Sign() <= 0 {
	//				log.Warn("Insufficient balance to collect", "address", uncollect.Address.Hex())
	//				continue
	//			}
	//			toAddress = &hotWalletInfo.Address
	//			gasLimit = EthGasLimit
	//			amount = collectAmount
	//		}
	//
	//		dFeeTx := &types.DynamicFeeTx{
	//			ChainID:   big.NewInt(int64(cc.chainNodeConf.ChainId)),
	//			Nonce:     uint64(nonce),
	//			GasTipCap: maxPriorityFeePerGas,
	//			GasFeeCap: maxFeePerGas,
	//			Gas:       gasLimit,
	//			To:        toAddress,
	//			Value:     amount,
	//			Data:      buildData,
	//		}
	//
	//		rawTx, txHash, err := ethereum.OfflineSignTx(dFeeTx, "", big.NewInt(int64(cc.chainNodeConf.ChainId)))
	//		if err != nil {
	//			log.Error("Offline transaction signing failed", "err", err)
	//			continue
	//		}
	//
	//		log.Info("Offline transaction signed", "rawTx", rawTx, "fromAddress", accountInfo.Address, "balance", uncollect.Balance, "amount", amount)
	//
	//		_, err = cc.rpcClient.SendTx(context.Background(), &wallet.SendTxRequest{
	//			Chain: cc.chainNodeConf.ChainName,
	//			RawTx: rawTx,
	//		})
	//		if err != nil {
	//			log.Error("Failed to send raw transaction", "err", err)
	//			continue
	//		}
	//
	//		guid := uuid.New()
	//		collection := database.Transactions{
	//			GUID:             guid,
	//			BlockHash:        common.Hash{},
	//			BlockNumber:      big.NewInt(1),
	//			Hash:             common.HexToHash(txHash),
	//			FromAddress:      uncollect.Address,
	//			ToAddress:        hotWalletInfo.Address,
	//			TokenAddress:     uncollect.TokenAddress,
	//			Fee:              big.NewInt(1000000000000000), // 0.001 ETH fee
	//			Amount:           uncollect.Balance,
	//			Status:           0,
	//			TxType:           2,
	//			TransactionIndex: big.NewInt(time.Now().Unix()),
	//			Timestamp:        uint64(time.Now().Unix()),
	//		}
	//		txList = append(txList, collection)
	//	}
	//
	//	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20000, MaxJitter: 250}
	//	if _, err := retry.Do[interface{}](cc.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
	//		return nil, cc.db.Transaction(func(tx *database.DB) error {
	//			if len(unCollectionList) > 0 {
	//				if err := tx.Balances.UpdateBalances(business.BusinessUid, unCollectionList, true); err != nil {
	//					return err
	//				}
	//			}
	//
	//			if err := tx.Transactions.StoreTransactions(business.BusinessUid, txList, uint64(len(txList))); err != nil {
	//				return err
	//			}
	//
	//			return nil
	//		})
	//	}); err != nil {
	//		log.Error("Failed to persist transactions", "err", err)
	//		return err
	//	}
	//}
	return nil
}
