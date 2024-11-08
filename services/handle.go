package services

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"

	"github.com/dapplink-labs/multichain-sync-account/database"
	dal_wallet_go "github.com/dapplink-labs/multichain-sync-account/protobuf/dal-wallet-go"
)

func (bws *BusinessMiddleWireServices) BusinessRegister(ctx context.Context, request *dal_wallet_go.BusinessRegisterRequest) (*dal_wallet_go.BusinessRegisterResponse, error) {
	if request.RequestId == "" || request.DepositNotify == "" || request.WithdrawNotify == "" {
		return &dal_wallet_go.BusinessRegisterResponse{
			Code: dal_wallet_go.ReturnCode_ERROR,
			Msg:  "invalid params",
		}, nil
	}
	business := &database.Business{
		GUID:           uuid.New(),
		BusinessUid:    request.RequestId,
		DepositNotify:  request.DepositNotify,
		WithdrawNotify: request.WithdrawNotify,
		TxFlowNotify:   request.TxFlowNotify,
		Timestamp:      uint64(time.Now().Unix()),
	}
	err := bws.db.Business.StoreBusiness(business)
	if err != nil {
		log.Error("store business fail", "err", err)
		return &dal_wallet_go.BusinessRegisterResponse{
			Code: dal_wallet_go.ReturnCode_ERROR,
			Msg:  "invalid params",
		}, nil
	}
	return &dal_wallet_go.BusinessRegisterResponse{
		Code: dal_wallet_go.ReturnCode_SUCCESS,
		Msg:  "config business success",
	}, nil
}

func (bws *BusinessMiddleWireServices) ExportAddressesByPublicKeys(ctx context.Context, request *dal_wallet_go.ExportAddressesRequest) (*dal_wallet_go.ExportAddressesResponse, error) {
	var retAddressess []*dal_wallet_go.Address
	var dbAddresses []database.Addresses
	for _, value := range request.PublicKeys {
		address := bws.accountClient.ExportAddressByPubKey(strconv.Itoa(int(value.Type)), value.PublicKey)
		item := &dal_wallet_go.Address{
			Type:    value.Type,
			Address: address,
		}
		dbAddress := database.Addresses{
			GUID:        uuid.New(),
			Address:     common.HexToAddress(address),
			AddressType: uint8(value.Type),
			PublicKey:   value.PublicKey,
			Timestamp:   uint64(time.Now().Unix()),
		}
		dbAddresses = append(dbAddresses, dbAddress)
		retAddressess = append(retAddressess, item)
	}
	err := bws.db.Addresses.StoreAddresses(request.RequestId, dbAddresses)
	if err != nil {
		return &dal_wallet_go.ExportAddressesResponse{
			Code: dal_wallet_go.ReturnCode_ERROR,
			Msg:  "store address to db fail",
		}, nil
	}
	return &dal_wallet_go.ExportAddressesResponse{
		Code:      dal_wallet_go.ReturnCode_SUCCESS,
		Msg:       "generate address success",
		Addresses: retAddressess,
	}, nil
}

func (bws *BusinessMiddleWireServices) CreateUnSignTransaction(ctx context.Context, request *dal_wallet_go.UnSignTransactionRequest) (*dal_wallet_go.UnSignTransactionResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (bws *BusinessMiddleWireServices) BuildSignedTransaction(ctx context.Context, request *dal_wallet_go.SignedTransactionRequest) (*dal_wallet_go.SignedTransactionResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (bws *BusinessMiddleWireServices) SetTokenAddress(ctx context.Context, request *dal_wallet_go.SetTokenAddressRequest) (*dal_wallet_go.SetTokenAddressResponse, error) {
	//TODO implement me
	panic("implement me")
}
