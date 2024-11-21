package database

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type TokenBalance struct {
	FromAddress  common.Address `json:"from_address"`
	ToAddress    common.Address `json:"to_address"`
	TokenAddress common.Address `json:"to_ken_address"`
	Balance      *big.Int       `json:"balance"`
	TxType       string         `json:"tx_type"` // deposit:充值；withdraw:提现；collection:归集；hot2cold:热转冷；cold2hot:冷转热
}
