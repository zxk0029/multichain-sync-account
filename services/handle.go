package services

import (
	"context"
	"github.com/dapplink-labs/multichain-transaction-syncs/common/slices"
	"github.com/dapplink-labs/multichain-transaction-syncs/common/strings"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"io"
	"time"

	"github.com/dapplink-labs/multichain-transaction-syncs/database"
	"github.com/dapplink-labs/multichain-transaction-syncs/database/dynamic"
	"github.com/dapplink-labs/multichain-transaction-syncs/protobuf/dal-wallet-go"

	"github.com/dgraph-io/ristretto"
	"google.golang.org/grpc"
)

type ScanService struct {
	// 数据库操作
	db *database.DB
	// 缓存，基于bloom filter实现
	cache *ristretto.Cache[string, database.Addresses]
}

// SetScanAddress 设置扫链地址
func (s ScanService) SetScanAddress(stream grpc.ClientStreamingServer[dal_wallet_go.SetScanAddressRequest, dal_wallet_go.BoilerplateResponse]) error {
	for {
		// 从流中接收客户端发送的请求
		req, err := stream.Recv()
		// 客户端关闭连接
		if err == io.EOF {
			return stream.SendAndClose(&dal_wallet_go.BoilerplateResponse{
				Code: 1,
				Msg:  "success",
			})
		}
		if err != nil {
			// 客户端发送错误
			return stream.SendAndClose(&dal_wallet_go.BoilerplateResponse{
				Code: 0,
				Msg:  err.Error(),
			})
		}
		if strings.IsValidTableName(req.RequestId) == false {
			// 客户端发送错误
			return stream.SendAndClose(&dal_wallet_go.BoilerplateResponse{
				Code: 0,
				Msg:  "request id does not comply with the rules",
			})
		}
		// 过滤已经存在缓存中的地址
		addresses := slices.Filter[*dal_wallet_go.Address](req.AddressList, func(item *dal_wallet_go.Address) bool {
			_, found := s.cache.Get(req.RequestId + item.Address)
			return !found
		})
		sa := make([]database.Addresses, len(addresses))
		// 将地址存入缓存
		for i, addr := range addresses {
			sa[i] = database.Addresses{
				GUID:        uuid.New(),
				BusinessUid: req.RequestId,
				Address:     common.HexToAddress(addr.Address),
				AddressType: uint8(addr.AddressType),
				Timestamp:   uint64(time.Now().Unix()),
			}
			s.cache.Set(req.RequestId+addr.Address, sa[i], 1)
		}
		// 将地址存入数据库
		err = s.db.Addresses.StoreAddresses(req.RequestId, sa, uint64(len(sa)))
		// 存入数据库出错
		if err != nil {
			// 删除缓存
			for _, addr := range addresses {
				s.cache.Del(req.RequestId + addr.Address)
			}
			// 给客户端发送错误
			return stream.SendAndClose(&dal_wallet_go.BoilerplateResponse{
				Code: 0,
				Msg:  err.Error(),
			})
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

// NewScanService 创建扫链服务
func NewScanService(db *database.DB) *ScanService {
	// 创建一个新的 Ristretto 缓存实例
	cache, err := ristretto.NewCache[string, database.Addresses](&ristretto.Config[string, database.Addresses]{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		log.Error("create ristretto cache failed", "err", err)
	}
	return &ScanService{
		db:    db,
		cache: cache,
	}
}
