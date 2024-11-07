package ethereum

import (
	"github.com/dapplink-labs/multichain-sync-account/common/bigint"
	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/dapplink-labs/multichain-sync-account/synchronizer/wallet-chain-node/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"math/big"
	"time"
)

// HandleCollection 归集处理
func HandleCollection(tx *wallet.BlockInfoTransactionList, receipt *wallet.TxReceiptByHashResponse, tokenAddress string) database.Deposits {
	t, _ := time.Parse("2006-01-02 15:04:05", tx.Time)
	gasPrice := bigint.StringToBigInt(receipt.EffectiveGasPrice)
	transactionFee := new(big.Int).Mul(gasPrice, big.NewInt(int64(receipt.GasUsed)))

	return database.Deposits{
		GUID:             uuid.New(),
		BlockHash:        common.HexToHash(receipt.BlockHash),
		BlockNumber:      bigint.StringToBigInt(receipt.BlockNumber),
		Hash:             common.HexToHash(tx.Hash),
		FromAddress:      common.HexToAddress(tx.From),
		ToAddress:        common.HexToAddress(tx.To),
		Status:           0,
		TransactionIndex: big.NewInt(int64(receipt.TransactionIndex)),
		Timestamp:        uint64(t.Unix()),
		TokenAddress:     common.HexToAddress(tokenAddress),
		Fee:              transactionFee,
		Amount:           bigint.StringToBigInt(tx.Amount),
	}
}

// WriteDataCollection 写入到数据库
func WriteDataCollection(db *database.DB, collectionMap map[string][]database.Deposits) {
	for businessId, collections := range collectionMap {
		err := db.Transaction(func(db *database.DB) error {
			if len(collections) == 0 {
				return nil
			}
			var tokenBalanceList []database.TokenBalance // 余额列表
			var transactionList []database.Transactions
			for _, collection := range collections {
				tokenBalanceList = append(tokenBalanceList, database.TokenBalance{
					Address:      collection.FromAddress,
					TokenAddress: collection.TokenAddress,
					LockBalance:  collection.Amount,
					TxType:       2,
				})
				transactionList = append(transactionList, database.Transactions{
					GUID:             uuid.New(),
					BlockHash:        collection.BlockHash,
					BlockNumber:      collection.BlockNumber,
					Hash:             collection.Hash,
					FromAddress:      collection.FromAddress,
					ToAddress:        collection.ToAddress,
					TokenAddress:     collection.TokenAddress,
					Fee:              collection.Fee,
					Amount:           collection.Amount,
					Status:           collection.Status,
					TransactionIndex: collection.TransactionIndex,
					Timestamp:        collection.Timestamp,
					TxType:           2,
				})
			}
			err := db.Balances.UpdateOrCreate(businessId, tokenBalanceList)
			if err != nil {
				log.Error("Failed to store withdraws: %v", err)
				return err
			}
			err = db.Transactions.StoreTransactions(businessId, transactionList, uint64(len(collections)))
			if err != nil {
				log.Error("Failed to store withdraws: %v", err)
				return err
			}
			return nil
		})
		if err != nil {
			log.Error("Failed to store withdraws: %v", err)
		}
	}
}
