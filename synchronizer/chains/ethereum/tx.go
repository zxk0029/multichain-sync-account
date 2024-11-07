package ethereum

import (
	"context"
	cache2 "github.com/dapplink-labs/multichain-sync-account/common/cache"
	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/dapplink-labs/multichain-sync-account/synchronizer/wallet-chain-node/wallet"
	"github.com/dgraph-io/ristretto"
	"github.com/ethereum/go-ethereum/log"
)

const (
	tokenABI = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
)

// ProcessBatch 批量交易处理
func ProcessBatch(client wallet.WalletServiceClient, db *database.DB, txs []*wallet.BlockInfoTransactionList) {
	cache := cache2.GetGlobalCache()
	depositMap := make(map[string][]database.Deposits)    // 充值列表
	withdrawMap := make(map[string][]database.Withdraws)  // 提现列表
	collectionMap := make(map[string][]database.Deposits) // 归集列表
	coldMap := make(map[string][]database.Deposits)       // 转冷列表
	hotMap := make(map[string][]database.Deposits)        // 转热列表

	for _, tx := range txs {
		txType, businessId, list := txHandler(client, db, tx, cache, "")
		if txType == "" || businessId == "" || list == nil {
			continue
		}
		if txType == "deposit" {
			depositMap[businessId] = append(depositMap[businessId], list.(database.Deposits))
		} else if txType == "withdraw" {
			withdrawMap[businessId] = append(withdrawMap[businessId], list.(database.Withdraws))
		} else if txType == "collection" {
			collectionMap[businessId] = append(collectionMap[businessId], list.(database.Deposits))
		} else if txType == "cold" {
			coldMap[businessId] = append(coldMap[businessId], list.(database.Deposits))
		} else if txType == "hot" {
			hotMap[businessId] = append(hotMap[businessId], list.(database.Deposits))
		}
	}

	// 如果有充值记录，写入数据库
	WriteDataDeposit(db, depositMap)
	// 如果有提现记录，写入数据库
	WriteDataWithdraw(db, withdrawMap)
	// 如果有归集记录，写入数据库
	WriteDataCollection(db, collectionMap)
	// 如果有转冷记录，写入数据库
	WriteDataCold(db, coldMap)
	// 如果有转热记录，写入数据库
	WriteDataHot(db, hotMap)
}

// txHandler 交易处理
func txHandler(client wallet.WalletServiceClient, db *database.DB, tx *wallet.BlockInfoTransactionList, cache *ristretto.Cache[string, *database.Addresses], tokenAddress string) (string, string, any) {
	from, fromIsExist := cache.Get(tx.From)
	to, toIsExist := cache.Get(tx.To)
	txType := GetTransactionType(fromIsExist, from, toIsExist, to, *db, tx)
	if txType == "" {
		return "", "", nil
	}
	// 获取交易回执，统一处理
	receipt, err := client.GetTxReceiptByHash(context.Background(), &wallet.TxReceiptByHashRequest{Chain: "Ethereum", Hash: tx.Hash})
	if err != nil {
		log.Error("Failed to get transaction receipt: %v", err)
		return txType, "", nil
	}
	if txType == "deposit" {
		// eth充值
		return txType, to.BusinessUid, HandleDeposit(tx, receipt, tokenAddress)
	} else if txType == "withdraw" {
		// 提现
		return txType, from.BusinessUid, HandleWithdraw(tx, receipt, tokenAddress)
	} else if txType == "collection" {
		// 归集
		return txType, from.BusinessUid, HandleCollection(tx, receipt, tokenAddress)
	} else if txType == "cold" {
		// 转冷
		return txType, from.BusinessUid, HandleCold(tx, receipt, tokenAddress)
	} else if txType == "hot" {
		// 转热
		return txType, from.BusinessUid, HandleHot(tx, receipt, tokenAddress)
	} else if txType == "token" {
		// 如果是代币转账的话，解析出正确的to地址、token地址和amount，再调用TxHandler
		tokenTo, tokenContractAddress, amount := HandleTokenTransfer(tx, receipt)
		tx.To = tokenTo
		tx.Amount = amount
		return txHandler(client, db, tx, cache, tokenContractAddress)
	}
	return txType, "", nil
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
	} else if fromIsExist && from.AddressType == 2 && toIsExist && to.AddressType == 1 {
		// 转热：from 地址是冷钱包地址，to 地址是热钱包地址
		return "hot"
	} else if token, _ := db.Tokens.TokensInfoByAddress(tx.To); token != nil {
		// 代币转账：如果 `to` 地址是合约地址
		return "token"
	} else {
		return ""
	}
}
