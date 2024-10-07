package ethereum

import (
	"context"
	"github.com/dapplink-labs/multichain-transaction-syncs/common/bigint"
	"github.com/dapplink-labs/multichain-transaction-syncs/database"
	"github.com/dapplink-labs/multichain-transaction-syncs/synchronizer/wallet-chain-node/wallet"
	"github.com/dgraph-io/ristretto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"math/big"
	"time"
)

// TxHandler 交易处理
func TxHandler(client wallet.WalletServiceClient, cache *ristretto.Cache[string, *database.Addresses], tx *wallet.BlockInfoTransactionList) {
	from, fromIsExist := cache.Get(tx.From)
	to, toIsExist := cache.Get(tx.To)
	// 充值列表
	var depositList []database.Deposits
	// 提现列表
	//var withdrawList []database.Withdraws
	if fromIsExist == false && toIsExist == true && to.AddressType == 0 {
		// 充值：form 地址等于外部地址，to地址等于系统内部用户地址

		// 如果to地址是合约地址说明是代币转账

		// 如果不是说明是原生币转账
		receipt, err := client.GetTxReceiptByHash(context.Background(), &wallet.TxReceiptByHashRequest{Chain: "Ethereum", Hash: tx.Hash})
		if err != nil {
			return
		}
		depositList = append(depositList, HandleDeposit(tx, receipt))

	} else if fromIsExist == true && from.AddressType == 1 && toIsExist == false {
		// 提现：form 地址等于热钱包地址，to地址等于系统外部地址

	} else if fromIsExist == true && from.AddressType == 0 && toIsExist == true && to.AddressType == 1 {
		// 归集：form地址是系统内部地址，to地址是热钱包地址

	} else if fromIsExist == true && from.AddressType == 1 && toIsExist == true && to.AddressType == 2 {
		// 转冷：form地址是热钱包地址，to地址是冷钱包地址

	} else if fromIsExist == true && from.AddressType == 0 && toIsExist == true && to.AddressType == 2 {
		// 转热：form地址是冷钱包地址，to地址是热钱包地址

	}
}

func HandleDeposit(tx *wallet.BlockInfoTransactionList, receipt *wallet.TxReceiptByHashResponse) database.Deposits {
	// 交易时间
	t, _ := time.Parse("2006-01-02 15:04:05", tx.Time)
	// 单个Gas 价格
	gasPrice := bigint.StringToBigInt(receipt.EffectiveGasPrice)
	// 交易费
	transactionFee := new(big.Int)
	// 计算交易费
	transactionFee.Mul(gasPrice, big.NewInt(int64(receipt.GasUsed)))

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
		//TokenAddress:     tokenAddress,
		Fee:    transactionFee,
		Amount: bigint.StringToBigInt(tx.Amount),
	}
}
