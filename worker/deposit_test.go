package worker

import (
	"context"
	"github.com/dapplink-labs/multichain-sync-account/database"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dapplink-labs/multichain-sync-account/config"
	"github.com/dapplink-labs/multichain-sync-account/rpcclient"
	"github.com/dapplink-labs/multichain-sync-account/rpcclient/chain-account/account"
)

const (
	depositBusinessId = "1"
	depositTxId       = "1e7e508b-5ad0-4ba7-b92c-b8bc1555fd9b"

	CurrentRequestId = 1
	CurrentChainId   = 17000
	CurrentChain     = "ethereum"
)

var CurrentBlockNumber = new(big.Int).SetUint64(200000)

func setupDeposit(t *testing.T) *Deposit {
	// 设置数据库
	db := database.SetupDb()

	// 设置 gRPC 连接
	conn, err := grpc.NewClient("127.0.0.1:8189", grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)

	// 设置账户客户端
	client := account.NewWalletAccountServiceClient(conn)
	accountClient, err := rpcclient.NewWalletChainAccountClient(context.Background(), client, "Ethereum")
	assert.NoError(t, err)

	// 设置配置
	cfg := &config.Config{
		ChainNode: config.ChainNodeConfig{
			ChainId:   CurrentChainId,
			ChainName: CurrentChain,

			StartingHeight: 2849348,
			Confirmations:  64,
			BlocksStep:     1,

			WorkerInterval:       10 * time.Second,
			SynchronizerInterval: 10 * time.Second,
		},
	}

	// 创建 shutdown 函数
	shutdown := func(cause error) {
		t.Logf("Shutdown called with cause: %v", cause)
	}

	// 创建 Deposit worker
	deposit, err := NewDeposit(cfg, db, accountClient, shutdown)
	assert.NoError(t, err)

	return deposit
}

func TestDeposit_Start(t *testing.T) {
	// 准备测试环境
	deposit := setupDeposit(t)

	// 启动 worker
	err := deposit.Start()
	assert.NoError(t, err)

	// 等待一段时间让 worker 处理交易
	time.Sleep(1000 * time.Second)

	// 清理资源
	err = deposit.Close()
	assert.NoError(t, err)
}

func TestDeposit_HandleTransaction(t *testing.T) {
	deposit := setupDeposit(t)

	// 准备测试数据
	tx := &Transaction{
		BlockNumber:  CurrentBlockNumber,
		Hash:         "0x123...",
		FromAddress:  "0x456...",
		ToAddress:    "0x789...",
		TokenAddress: "0xabc...",
		TxType:       "deposit",
	}

	txMsg := &account.TxMessage{
		Hash:   "0x123...",
		Fee:    "1000000000",
		Status: 1,
		Values: []*account.Value{
			{
				Value: "1000000000000000000",
			},
		},
	}

	// 测试处理交易
	result, err := deposit.BuildTransaction(tx, txMsg)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, tx.TxType, result.TxType)
}

func TestDeposit_HandleDeposit(t *testing.T) {
	deposit := setupDeposit(t)

	// 准备测试数据
	tx := &Transaction{
		BlockNumber:  CurrentBlockNumber,
		Hash:         "0x123...",
		FromAddress:  "0x456...",
		ToAddress:    "0x789...",
		TokenAddress: "0xabc...",
	}

	txMsg := &account.TxMessage{
		Hash: "0x123...",
		Fee:  "1000000000",
		Values: []*account.Value{
			{
				Value: "1000000000000000000",
			},
		},
	}

	// 测试处理存款
	result, err := deposit.HandleDeposit(tx, txMsg)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, tx.Hash, result.Hash.Hex())
}

func TestDeposit_HandleBatch(t *testing.T) {
	deposit := setupDeposit(t)

	// 准备测试数据
	batch := map[string]*TransactionsChannel{
		depositBusinessId: {
			BlockHeight: 12345,
			Transactions: []*Transaction{
				{
					BlockNumber:  CurrentBlockNumber,
					Hash:         "0x123...",
					FromAddress:  "0x456...",
					ToAddress:    "0x789...",
					TokenAddress: "0xabc...",
					TxType:       "deposit",
				},
			},
		},
	}

	// 测试处理批次
	err := deposit.handleBatch(batch)
	assert.NoError(t, err)
}
