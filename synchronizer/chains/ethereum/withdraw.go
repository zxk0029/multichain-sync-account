package ethereum

import (
	"github.com/dapplink-labs/multichain-transaction-syncs/common/bigint"
	"github.com/dapplink-labs/multichain-transaction-syncs/database"
	"github.com/dapplink-labs/multichain-transaction-syncs/synchronizer/wallet-chain-node/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"math/big"
	"time"
)

// HandleWithdraw 处理提现
func HandleWithdraw(tx *wallet.BlockInfoTransactionList, receipt *wallet.TxReceiptByHashResponse, tokenAddress string) database.Withdraws {
	t, _ := time.Parse("2006-01-02 15:04:05", tx.Time)
	gasPrice := bigint.StringToBigInt(receipt.EffectiveGasPrice)
	transactionFee := new(big.Int).Mul(gasPrice, big.NewInt(int64(receipt.GasUsed)))

	return database.Withdraws{
		GUID:             uuid.New(),
		BlockHash:        common.HexToHash(receipt.BlockHash),
		BlockNumber:      bigint.StringToBigInt(receipt.BlockNumber),
		Hash:             common.HexToHash(tx.Hash),
		FromAddress:      common.HexToAddress(tx.From),
		ToAddress:        common.HexToAddress(tx.To),
		TokenAddress:     common.HexToAddress(tokenAddress),
		Fee:              transactionFee,
		Amount:           bigint.StringToBigInt(tx.Amount),
		Status:           3,
		TransactionIndex: big.NewInt(int64(receipt.TransactionIndex)),
		TxSignHex:        "rsv",
		Timestamp:        uint64(t.Unix()),
	}
}

// WriteDataWithdraw 写入到数据库
func WriteDataWithdraw(db *database.DB, withdrawMap map[string][]database.Withdraws) {
	for businessId, withdraws := range withdrawMap {
		err := db.Transaction(func(db *database.DB) error {
			if len(withdraws) == 0 {
				return nil
			}
			var tokenBalanceList []database.TokenBalance // 余额列表
			for _, withdraw := range withdraws {
				tokenBalanceList = append(tokenBalanceList, database.TokenBalance{
					Address:      withdraw.FromAddress,
					TokenAddress: withdraw.TokenAddress,
					Balance:      withdraw.Amount,
					TxType:       1,
				})
			}
			err := db.Withdraws.StoreWithdraws(businessId, withdraws, uint64(len(withdraws)))
			if err != nil {
				log.Error("Failed to store withdraws: %v", err)
				return err
			}
			err = db.Balances.UpdateOrCreate(businessId, tokenBalanceList)
			if err != nil {
				log.Error("Failed to store withdraws: %v", err)
				return err
			}
			err = db.Withdraws.UpdateTransactionStatus(businessId, withdraws)
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
