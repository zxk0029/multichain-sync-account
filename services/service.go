package services

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/ethereum/go-ethereum/log"

	"github.com/dapplink-labs/multichain-transaction-syncs/database"
	"github.com/dapplink-labs/multichain-transaction-syncs/protobuf/dal-wallet-go"
)

const MaxRecvMessageSize = 1024 * 1024 * 300

type RpcServerConfig struct {
	GrpcHostname string
	GrpcPort     int
}

type RpcServer struct {
	*RpcServerConfig
	db *database.DB

	// wallet.UnimplementedWalletServiceServer
	stopped atomic.Bool
}

func (s *RpcServer) Stop(ctx context.Context) error {
	s.stopped.Store(true)
	return nil
}

func (s *RpcServer) Stopped() bool {
	return s.stopped.Load()
}

func NewRpcServer(db *database.DB, config *RpcServerConfig) (*RpcServer, error) {
	return &RpcServer{
		RpcServerConfig: config,
		db:              db,
	}, nil
}

func (s *RpcServer) Start(ctx context.Context) error {
	go func(s *RpcServer) {
		addr := fmt.Sprintf("%s:%d", s.GrpcHostname, s.GrpcPort)
		log.Info("start rpc server", "addr", addr)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Error("Could not start tcp listener. ")
		}

		// 注册gRPC服务
		gs := grpc.NewServer(
			// 用于设置gRPC客户端或服务器能够接收的最大消息大小
			grpc.MaxRecvMsgSize(MaxRecvMessageSize),
			// 用于设置gRPC服务器的拦截器
			grpc.ChainUnaryInterceptor(
				nil,
			),
		)
		// 为gRPC服务器启用了反射功能，使客户端能够动态查询服务器提供的服务和接口。
		reflection.Register(gs)

		// 注册接口服务
		dal_wallet_go.RegisterScanChainServer(gs, NewScanService(s.db))

		log.Info("Grpc info", "port", s.GrpcPort, "address", listener.Addr())
		if err := gs.Serve(listener); err != nil {
			log.Error("Could not GRPC server")
		}
	}(s)
	return nil
}
