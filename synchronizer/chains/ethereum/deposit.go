package ethereum

import (
	"github.com/dapplink-labs/multichain-sync-account/common/bigint"
	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/dapplink-labs/multichain-sync-account/synchronizer/wallet-chain-node/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"math/big"
	"strings"
	"time"
)

// HandleTokenTransfer 处理代币充值逻辑
func HandleTokenTransfer(tx *wallet.BlockInfoTransactionList, receipt *wallet.TxReceiptByHashResponse) (string, string, string) {
	contractAbi, err := abi.JSON(strings.NewReader(tokenABI))
	if err != nil {
		log.Error("Failed to parse ABI: %v", err)
		return "", "", ""
	}
	// 遍历日志，查找 Transfer 事件
	for _, vLog := range receipt.Logs {
		transferEventHash := contractAbi.Events["Transfer"].ID.Hex()
		if vLog.Topics[0] == transferEventHash {
			// 提取日志中的 to 地址和 amount
			to := common.HexToAddress(vLog.Topics[2])
			var value big.Int
			err := contractAbi.UnpackIntoInterface(&value, "Transfer", vLog.Data)
			if err != nil {
				log.Error("Failed to decode ABI: %v", err)
				return "", "", ""
			}
			return to.Hex(), tx.To, value.String()
		}
	}
	return "", "", ""
}

// HandleDeposit 处理充值的具体逻辑
func HandleDeposit(tx *wallet.BlockInfoTransactionList, receipt *wallet.TxReceiptByHashResponse, tokenAddress string) database.Deposits {
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

// WriteDataDeposit 写入到数据库
func WriteDataDeposit(db *database.DB, depositMap map[string][]database.Deposits) {
	for businessId, deposits := range depositMap {
		err := db.Transaction(func(db *database.DB) error {
			if len(deposits) == 0 {
				return nil
			}
			var tokenBalanceList []database.TokenBalance // 余额列表
			var transactionList []database.Transactions
			for _, deposit := range deposits {
				tokenBalanceList = append(tokenBalanceList, database.TokenBalance{
					Address:      deposit.FromAddress,
					TokenAddress: deposit.TokenAddress,
					LockBalance:  deposit.Amount,
					TxType:       0,
				})
				transactionList = append(transactionList, database.Transactions{
					GUID:             uuid.New(),
					BlockHash:        deposit.BlockHash,
					BlockNumber:      deposit.BlockNumber,
					Hash:             deposit.Hash,
					FromAddress:      deposit.FromAddress,
					ToAddress:        deposit.ToAddress,
					TokenAddress:     deposit.TokenAddress,
					Fee:              deposit.Fee,
					Amount:           deposit.Amount,
					Status:           deposit.Status,
					TransactionIndex: deposit.TransactionIndex,
					Timestamp:        deposit.Timestamp,
					TxType:           0,
				})
			}
			err := db.Deposits.StoreDeposits(businessId, deposits, uint64(len(deposits)))
			if err != nil {
				log.Error("Failed to store deposits: %v", err)
				return err
			}
			err = db.Balances.UpdateOrCreate(businessId, tokenBalanceList)
			if err != nil {
				log.Error("Failed to store deposits: %v", err)
				return err
			}
			err = db.Transactions.StoreTransactions(businessId, transactionList, uint64(len(deposits)))
			if err != nil {
				log.Error("Failed to store withdraws: %v", err)
				return err
			}
			return nil
		})
		if err != nil {
			log.Error("Failed to store deposits: %v", err)
		}
	}
}
