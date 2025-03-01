package database

import (
	"errors"
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

// ChainConfig defines the configuration for a blockchain
type ChainConfig struct {
	Native        TokenType // Native token type for the chain
	Default       TokenType // Default token type for the chain
	IsEVM         bool      // Whether this is an EVM compatible chain
	NativeAddress string    // Native token contract address
}

// ChainTokenTypes defines the mapping of chain names to their configurations
var ChainTokenTypes = map[string]ChainConfig{
	// EVM compatible chains (all use same zero address format)
	"ethereum": {
		Native:        "ETH",
		Default:       "ERC20",
		IsEVM:         true,
		NativeAddress: "0x0000000000000000000000000000000000000000",
	},
	"bsc": {
		Native:        "BNB",
		Default:       "BEP20",
		IsEVM:         true,
		NativeAddress: "0x0000000000000000000000000000000000000000",
	},
	"polygon": {
		Native:        "MATIC",
		Default:       "ERC20",
		IsEVM:         true,
		NativeAddress: "0x0000000000000000000000000000000000000000",
	},
	"avalanche-c": {
		Native:        "AVAX",
		Default:       "ERC20",
		IsEVM:         true,
		NativeAddress: "0x0000000000000000000000000000000000000000",
	},
	"arbitrum": {
		Native:        "ETH",
		Default:       "ERC20",
		IsEVM:         true,
		NativeAddress: "0x0000000000000000000000000000000000000000",
	},
	"optimism": {
		Native:        "ETH",
		Default:       "ERC20",
		IsEVM:         true,
		NativeAddress: "0x0000000000000000000000000000000000000000",
	},

	// Non-EVM chains
	"cosmos": {
		Native:        "ATOM",
		Default:       "CosmosCoin", // CW20
		IsEVM:         false,
		NativeAddress: "",
	},
	"solana": {
		Native:        "SOL",
		Default:       "SPL",
		IsEVM:         false,
		NativeAddress: "11111111111111111111111111111111",
	},
	"ton": {
		Native:        "TON",
		Default:       "JettonToken", // Added specific token type for TON
		IsEVM:         false,
		NativeAddress: "-1:0000000000000000000000000000000000000000000000000000000000000000",
	},
	"tron": {
		Native:        "TRX",
		Default:       "TRC20",
		IsEVM:         false,
		NativeAddress: "",
	},
	"xrp": {
		Native:        "XRP",
		Default:       "XRP",
		IsEVM:         false,
		NativeAddress: "",
	},
	"bitcoin": {
		Native:        "BTC",
		Default:       "BTC",
		IsEVM:         false,
		NativeAddress: "0000000000000000000000000000000000000000",
	},
}

type TokenType string

func (t TokenType) String() string {
	return string(t)
}

// GetTokenType returns the appropriate token type based on chain name and whether it's a native token
func GetTokenType(chainName string, isNative bool) TokenType {
	chainName = strings.ToLower(chainName)
	if config, ok := ChainTokenTypes[chainName]; ok {
		if isNative {
			return config.Native
		}
		return config.Default
	}
	// Default to ERC20 for unknown chains
	return "ERC20"
}

// GetNativeAddress returns the native token contract address for the given chain
func GetNativeAddress(chainName string) string {
	if config, ok := ChainTokenTypes[strings.ToLower(chainName)]; ok {
		return config.NativeAddress
	}
	return "0x0000000000000000000000000000000000000000" // Default to EVM zero address
}

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

func ParseTransactionType(s string) (TransactionType, error) {
	switch s {
	case string(TxTypeDeposit):
		return TxTypeDeposit, nil
	case string(TxTypeWithdraw):
		return TxTypeWithdraw, nil
	case string(TxTypeCollection):
		return TxTypeCollection, nil
	case string(TxTypeHot2Cold):
		return TxTypeHot2Cold, nil
	case string(TxTypeCold2Hot):
		return TxTypeCold2Hot, nil
	default:
		return TxTypeUnKnow, errors.New("unknown transaction type")
	}
}
