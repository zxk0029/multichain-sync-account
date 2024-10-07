package services

import (
	"context"
	"io"
	"time"

	"github.com/dapplink-labs/multichain-transaction-syncs/common/cache"
	"github.com/dapplink-labs/multichain-transaction-syncs/common/slices"
	"github.com/dapplink-labs/multichain-transaction-syncs/common/strings"
	"github.com/dapplink-labs/multichain-transaction-syncs/database"
	"github.com/dapplink-labs/multichain-transaction-syncs/database/dynamic"
	"github.com/dapplink-labs/multichain-transaction-syncs/protobuf/dal-wallet-go"
	"github.com/dgraph-io/ristretto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"

	"google.golang.org/grpc"
)

type ScanService struct {
	// 数据库操作
	db *database.DB
	// 缓存，基于bloom filter实现
	cache *ristretto.Cache[string, *database.Addresses]
}

// SetTokenAddress 设置代币合约地址
func (s ScanService) SetTokenAddress(stream grpc.ClientStreamingServer[dal_wallet_go.SetTokenAddressRequest, dal_wallet_go.BoilerplateResponse]) error {
	// 创建一个切片来保存所有接收到的 tokens
	var allTokens []database.Tokens
	for {
		// 从流中接收客户端发送的请求
		req, err := stream.Recv()
		// 客户端关闭连接，接收结束
		if err == io.EOF {
			// 检查是否有 tokens 要存储
			if len(allTokens) > 0 {
				// 一次性将所有 tokens 存入数据库
				err = s.db.Tokens.StoreTokens(allTokens, uint64(len(allTokens)))
				if err != nil {
					// 存入数据库失败，返回错误响应
					return stream.SendAndClose(&dal_wallet_go.BoilerplateResponse{
						Code: 0,
						Msg:  err.Error(),
					})
				}
			}
			// 存储成功，返回成功响应
			return stream.SendAndClose(&dal_wallet_go.BoilerplateResponse{
				Code: 1,
				Msg:  "success",
			})
		}
		// 处理其他错误
		if err != nil {
			// 客户端发送错误，返回错误响应
			return stream.SendAndClose(&dal_wallet_go.BoilerplateResponse{
				Code: 0,
				Msg:  err.Error(),
			})
		}
		// 遍历请求中的 TokenList，并将 token 信息存入 allTokens 切片
		for _, token := range req.TokenList {
			allTokens = append(allTokens, database.Tokens{
				GUID:         uuid.New(),
				TokenAddress: common.HexToAddress(token.Address),
				Decimals:     uint8(token.Decimals),
				TokenName:    token.TokenName,
				Timestamp:    uint64(time.Now().Unix()),
			})
		}
	}
}

// SetScanAddress 设置扫链地址
func (s ScanService) SetScanAddress(stream grpc.ClientStreamingServer[dal_wallet_go.SetScanAddressRequest, dal_wallet_go.BoilerplateResponse]) error {
	var allAddresses []database.Addresses
	for {
		// 从流中接收客户端发送的请求
		req, err := stream.Recv()
		// 客户端关闭连接
		if err == io.EOF {
			// 接收完毕，进行统一的存储操作
			if len(allAddresses) > 0 {
				// 将所有的地址存入数据库
				err = s.db.Addresses.StoreAddresses(allAddresses[0].BusinessUid, allAddresses, uint64(len(allAddresses)))
				if err != nil {
					// 如果存入数据库出错，清理缓存中的数据
					for _, addr := range allAddresses {
						s.cache.Del(addr.Address.Hex()) // 清除缓存
					}
					// 返回错误信息
					return stream.SendAndClose(&dal_wallet_go.BoilerplateResponse{
						Code: 0,
						Msg:  err.Error(),
					})
				}
			}
			// 如果所有操作成功，返回成功信息
			return stream.SendAndClose(&dal_wallet_go.BoilerplateResponse{
				Code: 1,
				Msg:  "success",
			})
		}

		// 如果在接收过程中发生其他错误
		if err != nil {
			return stream.SendAndClose(&dal_wallet_go.BoilerplateResponse{
				Code: 0,
				Msg:  err.Error(),
			})
		}

		// 校验 requestId
		if strings.IsValidTableName(req.RequestId) == false {
			return stream.SendAndClose(&dal_wallet_go.BoilerplateResponse{
				Code: 0,
				Msg:  "request id does not comply with the rules",
			})
		}

		// 过滤已经存在缓存中的地址，并暂时存储到内存中
		addresses := slices.Filter[*dal_wallet_go.Address](req.AddressList, func(item *dal_wallet_go.Address) bool {
			_, found := s.cache.Get(item.Address)
			return !found
		})

		// 临时保存有效地址到 allAddresses 切片
		for _, addr := range addresses {
			addressRecord := database.Addresses{
				GUID:        uuid.New(),
				BusinessUid: req.RequestId,
				Address:     common.HexToAddress(addr.Address),
				AddressType: uint8(addr.AddressType),
				Timestamp:   uint64(time.Now().Unix()),
			}
			// 将地址添加到临时切片中
			allAddresses = append(allAddresses, addressRecord)
			// 同时缓存地址，待后续统一存储
			s.cache.Set(addr.Address, &addressRecord, 1)
		}
	}
}

// SignUpScanService 注册扫链服务
func (s ScanService) SignUpScanService(ctx context.Context, request *dal_wallet_go.SignUpScanServiceRequest) (*dal_wallet_go.BoilerplateResponse, error) {
	if strings.IsValidTableName(request.RequestId) == false {
		// 客户端发送错误
		return &dal_wallet_go.BoilerplateResponse{
			Code: 0,
			Msg:  "request id does not comply with the rules",
		}, nil
	}
	// 创建表
	dynamic.CreateTableFromTemplate(request.RequestId, s.db)

	// 返回成功
	return &dal_wallet_go.BoilerplateResponse{
		Code: 1,
		Msg:  "success",
	}, nil
}

// RefreshCache 刷新缓存
func (s ScanService) RefreshCache(ctx context.Context, request *dal_wallet_go.RefreshCacheRequest) (*dal_wallet_go.BoilerplateResponse, error) {
	addresses, _ := s.db.Addresses.GetAllAddresses(request.RequestId)
	// 将地址存入缓存
	for _, addr := range addresses {
		s.cache.Set(addr.Address.Hex(), addr, 1)
	}
	return &dal_wallet_go.BoilerplateResponse{
		Code: 1,
		Msg:  "success",
	}, nil
}

//0xdAC17F958D2ee523a2206206994597C13D831ec7

// NewScanService 创建扫链服务
func NewScanService(db *database.DB) *ScanService {
	return &ScanService{
		db:    db,
		cache: cache.GetGlobalCache(),
	}
}
