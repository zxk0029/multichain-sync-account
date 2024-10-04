package database

import (
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type Balances struct {
	GUID         uuid.UUID      `gorm:"primaryKey" json:"guid"`
	Address      common.Address `json:"address" gorm:"serializer:bytes"`
	TokenAddress common.Address `json:"token_address" gorm:"serializer:bytes"`
	AddressType  uint8          `json:"address_type"` //0:用户地址；1:热钱包地址(归集地址)；2:冷钱包地址
	Balance      *big.Int       `gorm:"serializer:u256;column:balance" db:"balance" json:"Balance" form:"balance"`
	LockBalance  *big.Int       `gorm:"serializer:u256;column:lock_balance" db:"lock_balance" json:"LockBalance" form:"lock_balance"`
	Timestamp    uint64
}

type BalancesView interface {
	QueryWalletBalanceByTokenAndAddress(requestId string, address, tokenAddress common.Address) (*Balances, error)
	UnCollectionList(requestId string, amount *big.Int) ([]Balances, error)
	QueryHotWalletBalances(requestId string, amount *big.Int) ([]Balances, error)
	QueryBalancesByToAddress(requestId string, address *common.Address) (*Balances, error)
}

type BalancesDB interface {
	BalancesView

	UpdateOrCreate(string, []TokenBalance) error
	StoreBalances(string, []Balances, uint64) error
	UpdateBalances(string, []Balances, bool) error
}

type balancesDB struct {
	gorm *gorm.DB
}

func NewBalancesDB(db *gorm.DB) BalancesDB {
	return &balancesDB{gorm: db}
}

func (db *balancesDB) StoreBalances(requestId string, balanceList []Balances, balanceListLength uint64) error {
	result := db.gorm.Table("balances"+requestId).CreateInBatches(&balanceList, int(balanceListLength))
	return result.Error
}

func (db *balancesDB) UpdateBalances(requestId string, balanceList []Balances, isCollection bool) error {
	for i := 0; i < len(balanceList); i++ {
		var balance = Balances{}
		result := db.gorm.Table("balances" + requestId).Where(&Balances{Address: balanceList[i].Address, TokenAddress: balanceList[i].TokenAddress}).Take(&balance)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil
			}
			return result.Error
		}
		if isCollection {
			balance.LockBalance = balance.Balance
			balance.Balance = big.NewInt(0)
		} else {
			balance.Balance = new(big.Int).Sub(balance.Balance, balanceList[i].LockBalance)
			balance.LockBalance = balanceList[i].LockBalance
		}
		err := db.gorm.Save(&balance).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *balancesDB) QueryBalancesByToAddress(requestId string, address *common.Address) (*Balances, error) {
	var balanceEntry Balances
	err := db.gorm.Table("balances"+requestId).Where("address", strings.ToLower(address.String())).Take(&balanceEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &balanceEntry, nil
}

func (db *balancesDB) QueryHotWalletBalances(requestId string, amount *big.Int) ([]Balances, error) {
	var balanceList []Balances
	err := db.gorm.Table("balances"+requestId).Where("address_type = ? and balance >=?", 1, amount.Uint64()).Find(&balanceList).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return balanceList, nil
}

func (db *balancesDB) UnCollectionList(requestId string, amount *big.Int) ([]Balances, error) {
	var balanceList []Balances
	err := db.gorm.Table("balances"+requestId).Where("balance >=?", amount.Uint64()).Find(&balanceList).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return balanceList, nil
}

func (db *balancesDB) QueryWalletBalanceByTokenAndAddress(requestId string, address, tokenAddress common.Address) (*Balances, error) {
	var balanceEntry Balances
	err := db.gorm.Table("balances"+requestId).Where("address = ? and token_address = ?", strings.ToLower(address.String()), strings.ToLower(tokenAddress.String())).Take(&balanceEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &balanceEntry, nil
}

func (db *balancesDB) UpdateOrCreate(requestId string, balanceList []TokenBalance) error {
	return nil
}
