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

// HandleCold 转冷处理
func HandleCold(tx *wallet.BlockInfoTransactionList, receipt *wallet.TxReceiptByHashResponse, tokenAddress string) database.Deposits {
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

// WriteDataCold 写入到数据库
func WriteDataCold(db *database.DB, coldMap map[string][]database.Deposits) {
	for businessId, colds := range coldMap {
		err := db.Transaction(func(db *database.DB) error {
			if len(colds) == 0 {
				return nil
			}
			var tokenBalanceList []database.TokenBalance // 余额列表
			var transactionList []database.Transactions
			for _, cold := range colds {
				tokenBalanceList = append(tokenBalanceList, database.TokenBalance{
					Address:      cold.FromAddress,
					TokenAddress: cold.TokenAddress,
					LockBalance:  cold.Amount,
					TxType:       3,
				})
				transactionList = append(transactionList, database.Transactions{
					GUID:             uuid.New(),
					BlockHash:        cold.BlockHash,
					BlockNumber:      cold.BlockNumber,
					Hash:             cold.Hash,
					FromAddress:      cold.FromAddress,
					ToAddress:        cold.ToAddress,
					TokenAddress:     cold.TokenAddress,
					Fee:              cold.Fee,
					Amount:           cold.Amount,
					Status:           cold.Status,
					TransactionIndex: cold.TransactionIndex,
					Timestamp:        cold.Timestamp,
					TxType:           3,
				})
			}
			err := db.Balances.UpdateOrCreate(businessId, tokenBalanceList)
			if err != nil {
				log.Error("Failed to store withdraws: %v", err)
				return err
			}
			err = db.Transactions.StoreTransactions(businessId, transactionList, uint64(len(colds)))
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
