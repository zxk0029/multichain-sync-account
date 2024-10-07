package ethereum

import (
	"context"
	"github.com/dapplink-labs/multichain-transaction-syncs/common/bigint"
	"github.com/dapplink-labs/multichain-transaction-syncs/database"
	"github.com/dapplink-labs/multichain-transaction-syncs/synchronizer/wallet-chain-node/wallet"
	"github.com/dgraph-io/ristretto"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"math/big"
	"strings"
	"time"
)

const (
	tokenABI = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
)

// TxHandler 交易处理
func TxHandler(client wallet.WalletServiceClient, db database.DB, cache *ristretto.Cache[string, *database.Addresses], tx *wallet.BlockInfoTransactionList) {
	from, fromIsExist := cache.Get(tx.From)
	to, toIsExist := cache.Get(tx.To)
	txType := GetTransactionType(fromIsExist, from, toIsExist, to, db, tx)
	if txType == "" {
		return
	}

	var depositList []database.Deposits
	// 获取交易回执，统一处理
	receipt, err := client.GetTxReceiptByHash(context.Background(), &wallet.TxReceiptByHashRequest{Chain: "Ethereum", Hash: tx.Hash})
	if err != nil {
		log.Error("Failed to get transaction receipt: %v", err)
		return
	}

	if txType == "deposit" {
		// eth充值
		depositList = append(depositList, HandleDeposit(tx, receipt, ""))
	} else if txType == "withdraw" {
		// 提现

	} else if txType == "collection" {
		// 归集
	} else if txType == "cold" {
		// 冷钱包
	} else if txType == "hot" {
		// 热钱包
	} else if txType == "token" {
		// 代币充值
		handleTokenTransfer(tx, receipt, fromIsExist, toIsExist, cache, db, &depositList)
	}

	// 如果有充值记录，写入数据库
	if len(depositList) > 0 {
		// 处理存入数据库或其他逻辑
	}
}

// GetTransactionType 判断条件是否满足
func GetTransactionType(fromIsExist bool, from *database.Addresses, toIsExist bool, to *database.Addresses, db database.DB, tx *wallet.BlockInfoTransactionList) string {
	if !fromIsExist && toIsExist && to.AddressType == 0 {
		// 充值：from 地址是外部地址，to 地址是系统内部用户地址
		return "deposit"
	} else if fromIsExist && from.AddressType == 1 && !toIsExist {
		// 提现：from 地址是热钱包地址，to 地址是外部地址
		return "withdraw"
	} else if fromIsExist && from.AddressType == 0 && toIsExist && to.AddressType == 1 {
		// 归集：from 地址是系统内部地址，to 地址是热钱包地址
		return "collection"
	} else if fromIsExist && from.AddressType == 1 && toIsExist && to.AddressType == 2 {
		// 转冷：from 地址是热钱包地址，to 地址是冷钱包地址
		return "cold"
	} else if fromIsExist && from.AddressType == 0 && toIsExist && to.AddressType == 2 {
		// 转热：from 地址是冷钱包地址，to 地址是热钱包地址
		return "hot"
	} else if token, _ := db.Tokens.TokensInfoByAddress(tx.To); token != nil {
		// 代币转账：如果 `to` 地址是合约地址
		return "token"
	} else {
		return ""
	}
}

// 处理代币转账逻辑
func handleTokenTransfer(tx *wallet.BlockInfoTransactionList, receipt *wallet.TxReceiptByHashResponse, fromIsExist, toIsExist bool, cache *ristretto.Cache[string, *database.Addresses], db database.DB, depositList *[]database.Deposits) {
	contractAbi, err := abi.JSON(strings.NewReader(tokenABI))
	if err != nil {
		log.Error("Failed to parse ABI: %v", err)
		return
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
				return
			}
			tx.Amount = value.String()
			tokenAddress := tx.To
			tx.To = to.Hex()
			// 检查目标地址
			dbTo, toIsExist := cache.Get(tx.To)
			if !fromIsExist && toIsExist && dbTo.AddressType == 0 {
				*depositList = append(*depositList, HandleDeposit(tx, receipt, tokenAddress))
			}
			break
		}
	}
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
		Status:           uint8(receipt.Status),
		TransactionIndex: big.NewInt(int64(receipt.TransactionIndex)),
		Timestamp:        uint64(t.Unix()),
		TokenAddress:     common.HexToAddress(tokenAddress),
		Fee:              transactionFee,
		Amount:           bigint.StringToBigInt(tx.Amount),
	}
}
