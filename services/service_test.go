package services

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dapplink-labs/multichain-sync-account/common/json2"
	"github.com/dapplink-labs/multichain-sync-account/config"
	"github.com/dapplink-labs/multichain-sync-account/database"
	dal_wallet_go "github.com/dapplink-labs/multichain-sync-account/protobuf/dal-wallet-go"
	"github.com/dapplink-labs/multichain-sync-account/rpcclient"
	"github.com/dapplink-labs/multichain-sync-account/rpcclient/chain-account/account"
)

func setupDb() *database.DB {
	dbConfig := config.DBConfig{
		Host:     "127.0.0.1",
		Port:     5432,
		Name:     "multichain",
		User:     "postgres",
		Password: "123456",
	}

	newDB, _ := database.NewDB(context.Background(), dbConfig)
	return newDB
}

func setup(t *testing.T) *BusinessMiddleWireServices {
	db := setupDb()

	bConfig := &BusinessMiddleConfig{
		GrpcHostname: "localhost",
		GrpcPort:     50051,
	}

	conn, err := grpc.NewClient("127.0.0.1:8189", grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)

	client := account.NewWalletAccountServiceClient(conn)
	accountClient, err := rpcclient.NewWalletChainAccountClient(context.Background(), client, "Ethereum")

	bws, err := NewBusinessMiddleWireServices(db, bConfig, accountClient)
	assert.NoError(t, err)

	return bws
}

func TestBusinessMiddleWireServices_CreateUnSignTransaction_ETHTransfer(t *testing.T) {
	// 准备测试环境
	bws := setup(t)
	ctx := context.Background()

	// 构造请求
	request := &dal_wallet_go.UnSignWithdrawTransactionRequest{
		ConsumerToken: "test_token",
		RequestId:     "1",
		ChainId:       "17000", // 主网
		Chain:         "ethereum",
		From:          "0xD79053a14BC465d9C1434d4A4fAbdeA7b6a2A94b",
		To:            "0xDf894d39f6b33763bf55582Bb7A8b5515bccD982",
		//Value:         "1000000000000000000", // 1 ETH
		Value:           "10000000000000000", // 0.01 ETH
		ContractAddress: "0x00",
		TokenId:         "",
		TokenMeta:       "",
		TxType:          "withdraw",
	}

	// 执行测试
	response, err := bws.CreateUnSignTransaction(ctx, request)

	// 打印响应详情
	respJSON := json2.ToPrettyJSON(response)
	t.Logf("Response:\n%s", respJSON)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, dal_wallet_go.ReturnCode_SUCCESS, response.Code)
	assert.NotEmpty(t, response.TransactionId)
	assert.NotEmpty(t, response.UnSignTx)

	// 可以添加更详细的验证
	// 例如验证 UnSignTx 的格式是否正确
	assert.True(t, strings.HasPrefix(response.UnSignTx, "0x"))

	//respJSON2, _ := json.Marshal(response)
	//t.Logf("Response:\n%s", string(respJSON2))
}
