package services

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dapplink-labs/multichain-sync-account/config"
	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/dapplink-labs/multichain-sync-account/rpcclient"
	"github.com/dapplink-labs/multichain-sync-account/rpcclient/chain-account/account"
)

const (
	NotifyUrl        = "http://127.0.0.1:8001"
	CurrentRequestId = "1"

	// 链的类型常量
	ChainTypeEthereum = "Ethereum"
	ChainTypeSolana   = "Solana"
)

// TestChainType 控制要测试的链类型, 可以设置为 ChainTypeEthereum 或 ChainTypeSolana 来测试特定链, 为空时测试所有链
var TestChainType string

// ChainConfig 定义链相关的配置
type ChainConfig struct {
	ChainId   string
	ChainName string
	// 添加链特定的测试数据
	TestAddresses struct {
		From string
		To   string
	}
	TestPublicKey   string
	ContractAddress string
}

var (
	ChainConfigs = map[string]ChainConfig{
		ChainTypeEthereum: {
			ChainId:   "17000",
			ChainName: ChainTypeEthereum,
			TestAddresses: struct {
				From string
				To   string
			}{
				From: "0x82565b64e8063674CAea7003979280f4dbC3aAE7",
				To:   "0xDf894d39f6b33763bf55582Bb7A8b5515bccD982",
			},
			TestPublicKey:   "048846b3ce4376e8d58c83c1c6420a784caa675d7f26c496f499585d09891af8fc9167a4b658b57b28211783cdee651caa8b5341b753fa39c995317670123f12d8",
			ContractAddress: "0x0000000000000000000000000000000000000000",
		},
		ChainTypeSolana: {
			ChainId:   "501",
			ChainName: ChainTypeSolana,
			TestAddresses: struct {
				From string
				To   string
			}{
				From: "C3QtZA7tXqn8EjHLG3AyDQvuuNuYUh2D7PxRDQSTSNrt",
				To:   "Cc7ZvTv9H2aEWyRzZik8k75Z3nPphEtgNPHCjnBz3F6N",
			},
			TestPublicKey:   "92730a92fd64ffa5826a5bf8cc72265ed6c57a8b7b7c71feaa0774971b181598",
			ContractAddress: "11111111111111111111111111111111",
		},
	}
)

// shouldRunChainTest 判断是否应该运行特定链的测试
func shouldRunChainTest(t *testing.T, chainType string) bool {
	// 如果 TestChainType 变量已设置，只运行指定链的测试
	if TestChainType != "" {
		shouldRun := TestChainType == chainType
		if shouldRun {
			t.Logf("Running tests for chain: %s", chainType)
		}
		return shouldRun
	}

	// 如果 TestChainType 未设置，测试所有链
	t.Logf("No specific chain type set, running tests for: %s", chainType)
	return true
}

// SetTestChain 设置要测试的链类型
func SetTestChain(chainType string) {
	TestChainType = chainType
}

func setupDb() *database.DB {
	// 使用默认的 PostgreSQL 配置，简化测试设置
	dbConfig := config.DBConfig{
		Host:     "127.0.0.1",
		Port:     5432,
		Name:     "multichain",
		User:     "sinco-z",
		Password: "",
	}

	fmt.Println("Using default database configuration for testing:")
	fmt.Printf("  Host: %s, Port: %d, DB: %s, User: %s\n",
		dbConfig.Host, dbConfig.Port, dbConfig.Name, dbConfig.User)

	// 创建数据库连接
	newDB, err := database.NewDB(context.Background(), dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	return newDB
}

// CleanupDB closes the database connection
func CleanupDB(db *database.DB) {
	if db != nil {
		db.Close()
	}
}

func setupService(t *testing.T, chainConfig ChainConfig) (*BusinessMiddleWireServices, func()) {
	fmt.Println("Setting up test service")
	db := setupDb()

	// 使用默认的 GRPC 配置
	grpcHost := "127.0.0.1"
	grpcPort := 50051
	grpcAddr := "127.0.0.1:8189"

	fmt.Println("Using default GRPC configuration for testing:")
	fmt.Printf("  Host: %s, Port: %d, Address: %s\n", grpcHost, grpcPort, grpcAddr)

	bConfig := &BusinessMiddleConfig{
		GrpcHostname: grpcHost,
		GrpcPort:     grpcPort,
	}

	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)

	client := account.NewWalletAccountServiceClient(conn)
	accountClient, err := rpcclient.NewWalletChainAccountClient(context.Background(), client, chainConfig.ChainName)
	assert.NoError(t, err)

	bws, err := NewBusinessMiddleWireServices(db, bConfig, accountClient)
	assert.NoError(t, err)

	// Return a cleanup function that closes resources
	cleanup := func() {
		CleanupDB(db)
		conn.Close()
	}

	return bws, cleanup
}
