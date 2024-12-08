package worker

import (
	"context"
	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strconv"
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

func TestDeposit_SendTransaction(t *testing.T) {
	// 准备测试环境
	deposit := setupDeposit(t)

	depositTxId := "818e6568-17ee-463b-ad29-ea05adcc664d"

	dbDeposit, err := deposit.database.Deposits.QueryDepositsById(strconv.Itoa(CurrentRequestId), depositTxId)
	assert.NoError(t, err)

	// 模拟发送交易上链
	sendTx, err := deposit.rpcClient.SendTx(dbDeposit.TxSignHex)
	assert.NoError(t, err)

	dbDeposit.TxHash = common.HexToHash(sendTx)
	dbDeposit.Status = database.TxStatusBroadcasted

	err = deposit.database.Deposits.UpdateDepositListById(strconv.Itoa(CurrentRequestId), []*database.Deposits{dbDeposit})
	assert.NoError(t, err)
}
