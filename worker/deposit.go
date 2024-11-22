package worker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-sync-account/common/retry"
	"github.com/dapplink-labs/multichain-sync-account/common/tasks"
	"github.com/dapplink-labs/multichain-sync-account/config"
	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/dapplink-labs/multichain-sync-account/rpcclient"
	"github.com/dapplink-labs/multichain-sync-account/rpcclient/chain-account/account"
)

type Deposit struct {
	BaseSynchronizer

	confirms       uint8
	latestHeader   rpcclient.BlockHeader
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewDeposit(cfg *config.Config, db *database.DB, shutdown context.CancelCauseFunc) (*Deposit, error) {
	log.Info("New deposit", "ChainAccountRpc", cfg.ChainAccountRpc)
	conn, err := grpc.NewClient(cfg.ChainAccountRpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("Connect to da retriever fail", "err", err)
		return nil, err
	}
	client := account.NewWalletAccountServiceClient(conn)
	accountClient, err := rpcclient.NewWalletChainAccountClient(context.Background(), client, "Ethereum")
	if err != nil {
		log.Error("new wallet account client fail", "err", err)
		return nil, err
	}

	businessList, err := db.Business.QueryBusinessList()
	if err != nil {
		log.Error("query business list fail", "err", err)
		return nil, err
	}
	var businessIds []string
	for _, business := range businessList {
		businessIds = append(businessIds, business.BusinessUid)
	}

	dbLatestBlockHeader, err := db.Blocks.LatestBlocks()
	if err != nil {
		log.Error("get latest block from database fail")
		return nil, err
	}
	var fromHeader *rpcclient.BlockHeader

	if dbLatestBlockHeader != nil {
		log.Info("sync bock", "number", dbLatestBlockHeader.Number, "hash", dbLatestBlockHeader.Hash)
		fromHeader = dbLatestBlockHeader
	} else if cfg.ChainNode.StartingHeight > 0 {
		chainLatestBlockHeader, err := accountClient.GetBlockHeader(big.NewInt(int64(cfg.ChainNode.StartingHeight)))
		if err != nil {
			log.Error("get block from chain account fail", "err", err)
			return nil, err
		}
		fromHeader = chainLatestBlockHeader
	} else {
		chainLatestBlockHeader, err := accountClient.GetBlockHeader(nil)
		if err != nil {
			log.Error("get block from chain account fail", "err", err)
			return nil, err
		}
		fromHeader = chainLatestBlockHeader
	}

	businessTxChannel := make(chan map[string]*TransactionsChannel)

	baseSyncer := BaseSynchronizer{
		loopInterval:     cfg.ChainNode.SynchronizerInterval,
		headerBufferSize: cfg.ChainNode.BlocksStep,
		businessChannels: businessTxChannel,
		rpcClient:        accountClient,
		blockBatch:       rpcclient.NewBatchBlock(accountClient, fromHeader, big.NewInt(int64(cfg.ChainNode.Confirmations))),
		database:         db,
		businessIds:      businessIds,
	}

	resCtx, resCancel := context.WithCancel(context.Background())

	return &Deposit{
		BaseSynchronizer: baseSyncer,
		confirms:         uint8(cfg.ChainNode.Confirmations),
		resourceCtx:      resCtx,
		resourceCancel:   resCancel,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in deposit: %w", err))
		}},
	}, nil
}

func (deposit *Deposit) Close() error {
	var result error
	if err := deposit.BaseSynchronizer.Close(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to close internal base synchronizer: %w", err))
	}
	deposit.resourceCancel()
	if err := deposit.tasks.Wait(); err != nil {
		result = errors.Join(result, fmt.Errorf("failed to await batch handler completion: %w", err))
	}
	return result
}

func (deposit *Deposit) Start() error {
	log.Info("starting deposit...")
	if err := deposit.BaseSynchronizer.Start(); err != nil {
		return fmt.Errorf("failed to start internal Synchronizer: %w", err)
	}
	deposit.tasks.Go(func() error {
		log.Info("handle deposit task start")
		for batch := range deposit.businessChannels {
			log.Info("deposit business channel", "batch length", len(batch))
			if err := deposit.handleBatch(batch); err != nil {
				return fmt.Errorf("failed to handle batch, stopping L2 Synchronizer: %w", err)
			}
		}
		return nil
	})
	return nil
}

func (deposit *Deposit) handleBatch(batch map[string]*TransactionsChannel) error {
	for _, businessId := range deposit.businessIds {
		_, exists := batch[businessId]
		if !exists {
			continue
		}

		var (
			transactionFlowList []database.Transactions
			depositList         []database.Deposits
			withdrawList        []database.Withdraws
			internals           []database.Internals
			balances            []database.TokenBalance
		)

		log.Info("handle business flow", "businessId", businessId, "chainLatestBlock", batch[businessId].BlockHeight, "txn", len(batch[businessId].Transactions))

		for _, tx := range batch[businessId].Transactions {
			log.Info("Request transaction from chain account", "txHash", tx.Hash, "fromAddress", tx.FromAddress)
			txItem, err := deposit.rpcClient.GetTransactionByHash(tx.Hash)
			if err != nil {
				log.Info("get transaction by hash fail", "err", err)
				return err
			}
			amountBigInt, _ := new(big.Int).SetString(txItem.Values[0].Value, 10)
			log.Info("Transaction amount", "amountBigInt", amountBigInt, "FromAddress", tx.FromAddress, "TokenAddress", tx.TokenAddress, "TokenAddress", tx.ToAddress)
			balances = append(
				balances,
				database.TokenBalance{
					FromAddress:  common.HexToAddress(tx.FromAddress),
					ToAddress:    common.HexToAddress(txItem.Tos[0].Address),
					TokenAddress: common.HexToAddress(txItem.ContractAddress),
					Balance:      amountBigInt,
					TxType:       tx.TxType,
				},
			)

			log.Info("get transaction success", "txHash", txItem.Hash)
			transactionFlow, err := deposit.HandleTransaction(tx, txItem)
			if err != nil {
				log.Info("handle  transaction fail", "err", err)
				return err
			}
			transactionFlowList = append(transactionFlowList, transactionFlow)

			switch tx.TxType {
			case "deposit":
				depositItem, _ := deposit.HandleDeposit(tx, txItem)
				depositList = append(depositList, depositItem)
				break
			case "withdraw":
				withdrawItem, _ := deposit.HandleWithdraw(tx, txItem)
				withdrawList = append(withdrawList, withdrawItem)
				break
			case "collection", "hot2cold", "cold2hot":
				internelItem, _ := deposit.HandleInternalTx(tx, txItem)
				internals = append(internals, internelItem)
				break
			default:
				break
			}
		}
		retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
		if _, err := retry.Do[interface{}](deposit.resourceCtx, 10, retryStrategy, func() (interface{}, error) {
			if err := deposit.database.Transaction(func(tx *database.DB) error {
				if len(depositList) > 0 {
					log.Info("Store deposit transaction success", "totalTx", len(depositList))
					if err := tx.Deposits.StoreDeposits(businessId, depositList, uint64(len(depositList))); err != nil {
						return err
					}
				}

				if err := tx.Deposits.UpdateDepositsComfirms(businessId, batch[businessId].BlockHeight, uint64(deposit.confirms)); err != nil {
					log.Info("Handle confims fail", "totalTx", "err", err)
					return err
				}

				if len(balances) > 0 {
					log.Info("Handle balances success", "totalTx", len(balances))
					if err := tx.Balances.UpdateOrCreate(businessId, balances); err != nil {
						return err
					}
				}

				if len(withdrawList) > 0 {
					if err := tx.Withdraws.UpdateWithdrawStatus(businessId, 3, withdrawList); err != nil {
						return err
					}
				}

				if len(internals) > 0 {
					if err := tx.Internals.UpdateInternalstatus(businessId, 3, internals); err != nil {
						return err
					}
				}

				if len(transactionFlowList) > 0 {
					if err := tx.Transactions.StoreTransactions(businessId, transactionFlowList, uint64(len(transactionFlowList))); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				log.Error("unable to persist batch", "err", err)
				return nil, err
			}
			return nil, nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func (deposit *Deposit) HandleDeposit(tx *Transaction, txMsg *account.TxMessage) (database.Deposits, error) {
	txFee, _ := new(big.Int).SetString(txMsg.Fee, 10)
	txAmount, _ := new(big.Int).SetString(txMsg.Values[0].Value, 10)
	depositTx := database.Deposits{
		GUID:         uuid.New(),
		BlockHash:    common.Hash{},
		BlockNumber:  tx.BlockNumber,
		Hash:         common.HexToHash(tx.Hash),
		FromAddress:  common.HexToAddress(tx.FromAddress),
		ToAddress:    common.HexToAddress(tx.ToAddress),
		TokenAddress: common.HexToAddress(tx.TokenAddress),
		TokenId:      "0x00",
		TokenMeta:    "0x00",
		Fee:          txFee,
		Amount:       txAmount,
		Status:       0,
		Timestamp:    uint64(time.Now().Unix()),
	}
	return depositTx, nil
}

func (deposit *Deposit) HandleWithdraw(tx *Transaction, txMsg *account.TxMessage) (database.Withdraws, error) {
	txFee, _ := new(big.Int).SetString(txMsg.Fee, 10)
	txAmount, _ := new(big.Int).SetString(txMsg.Values[0].Value, 10)
	withdrawTx := database.Withdraws{
		GUID:         uuid.New(),
		BlockHash:    common.Hash{},
		BlockNumber:  tx.BlockNumber,
		Hash:         common.HexToHash(tx.Hash),
		FromAddress:  common.HexToAddress(tx.FromAddress),
		ToAddress:    common.HexToAddress(tx.ToAddress),
		TokenAddress: common.HexToAddress(tx.TokenAddress),
		TokenId:      "0x00",
		TokenMeta:    "0x00",
		Fee:          txFee,
		Amount:       txAmount,
		Status:       2,
		Timestamp:    uint64(time.Now().Unix()),
	}
	return withdrawTx, nil
}

func (deposit *Deposit) HandleTransaction(tx *Transaction, txMsg *account.TxMessage) (database.Transactions, error) {
	txFee, _ := new(big.Int).SetString(txMsg.Fee, 10)
	txAmount, _ := new(big.Int).SetString(txMsg.Values[0].Value, 10)
	transationTx := database.Transactions{
		GUID:         uuid.New(),
		BlockHash:    common.Hash{},
		BlockNumber:  tx.BlockNumber,
		Hash:         common.HexToHash(tx.Hash),
		FromAddress:  common.HexToAddress(tx.FromAddress),
		ToAddress:    common.HexToAddress(tx.ToAddress),
		TokenAddress: common.HexToAddress(tx.TokenAddress),
		TokenId:      "0x00",
		TokenMeta:    "0x00",
		Fee:          txFee,
		Status:       uint8(txMsg.Status),
		Amount:       txAmount,
		TxType:       tx.TxType,
		Timestamp:    uint64(time.Now().Unix()),
	}
	return transationTx, nil
}

func (deposit *Deposit) HandleInternalTx(tx *Transaction, txMsg *account.TxMessage) (database.Internals, error) {
	txFee, _ := new(big.Int).SetString(txMsg.Fee, 10)
	txAmount, _ := new(big.Int).SetString(txMsg.Values[0].Value, 10)
	internalTx := database.Internals{
		GUID:         uuid.New(),
		BlockHash:    common.Hash{},
		BlockNumber:  tx.BlockNumber,
		Hash:         common.HexToHash(tx.Hash),
		FromAddress:  common.HexToAddress(tx.FromAddress),
		ToAddress:    common.HexToAddress(tx.ToAddress),
		TokenAddress: common.HexToAddress(tx.TokenAddress),
		TokenId:      "0x00",
		TokenMeta:    "0x00",
		Fee:          txFee,
		Status:       uint8(txMsg.Status),
		Amount:       txAmount,
		TxType:       tx.TxType,
		Timestamp:    uint64(time.Now().Unix()),
	}
	return internalTx, nil
}
