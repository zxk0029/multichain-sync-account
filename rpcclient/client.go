package rpcclient

import (
	"context"
	"github.com/dapplink-labs/multichain-sync-account/rpcclient/chain-account/account"
	"github.com/ethereum/go-ethereum/log"
)

type WalletChainAccountClient struct {
	Ctx             context.Context
	ChainName       string
	AccountRpClient account.WalletAccountServiceClient
}

func NewWalletChainAccountClient(ctx context.Context, rpc account.WalletAccountServiceClient, chainName string) (*WalletChainAccountClient, error) {
	return &WalletChainAccountClient{Ctx: ctx, AccountRpClient: rpc, ChainName: chainName}, nil
}

func (wac *WalletChainAccountClient) ExportAddressByPubKey(method, publicKey string) string {
	req := &account.ConvertAddressRequest{
		Chain:     wac.ChainName,
		Type:      method,
		PublicKey: publicKey,
	}
	address, err := wac.AccountRpClient.ConvertAddress(wac.Ctx, req)
	if err != nil || address.Code == 0 {
		log.Error("covert address fail", "err", err)
		return ""
	}
	return address.Address
}
