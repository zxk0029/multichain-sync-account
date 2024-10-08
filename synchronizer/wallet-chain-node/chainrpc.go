package wallet_chain_node

import (
	"github.com/dapplink-labs/multichain-transaction-syncs/synchronizer/wallet-chain-node/wallet"
	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// InitRpcClient 初始化RPC
func InitRpcClient(addr string) *wallet.WalletServiceClient {
	// 创建 gRPC 不安全的连接
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("did not connect: %v", err)
	}
	// 创建 gRPC 客户端
	client := wallet.NewWalletServiceClient(conn)
	// 准备调用服务
	return client
}
