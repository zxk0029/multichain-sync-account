package database

import (
	"fmt"
	"strings"
)

type TxStatus string

const (
	TxStatusCreateUnsigned TxStatus = "create_unsign"
	TxStatusSigned         TxStatus = "signed"
	TxStatusBroadcasted    TxStatus = "broadcasted"
	TxStatusWalletDone     TxStatus = "wallet_done"
	TxStatusNotified       TxStatus = "notified"
	TxStatusSuccess        TxStatus = "success"
)

type TokenType string

const (
	TokenTypeETH     TokenType = "ETH"
	TokenTypeERC20   TokenType = "ERC20"
	TokenTypeERC721  TokenType = "ERC721"
	TokenTypeERC1155 TokenType = "ERC1155"
)

type AddressType string

const (
	AddressTypeEOA  AddressType = "eoa"
	AddressTypeHot  AddressType = "hot"
	AddressTypeCold AddressType = "cold"
)

func (at AddressType) String() string {
	return string(at)
}

func ParseAddressType(s string) (AddressType, error) {
	switch strings.ToLower(s) {
	case string(AddressTypeEOA):
		return AddressTypeEOA, nil
	case string(AddressTypeHot):
		return AddressTypeHot, nil
	case string(AddressTypeCold):
		return AddressTypeCold, nil
	default:
		return "", fmt.Errorf("invalid address type: %s", s)
	}
}

type TransactionType string

const (
	TxTypeUnKnow     TransactionType = "unknow"
	TxTypeDeposit    TransactionType = "deposit"
	TxTypeWithdraw   TransactionType = "withdraw"
	TxTypeCollection TransactionType = "collection"
	TxTypeHot2Cold   TransactionType = "hot2cold"
	TxTypeCold2Hot   TransactionType = "cold2hot"
)

type DepositStatus string

const (
	DepositStatusPending      DepositStatus = "pending"
	DepositStatusWalletDone   DepositStatus = "walletDone"
	DepositStatusBusinessDone DepositStatus = "businessDone"
	DepositStatusCompleted    DepositStatus = "completed"
)
