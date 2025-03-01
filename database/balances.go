package database

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dapplink-labs/multichain-sync-account/database/utils"
)

type Balances struct {
	GUID         uuid.UUID   `gorm:"primary_key" json:"guid"`
	Address      string      `gorm:"type:varchar;not null" json:"address"`
	TokenAddress string      `gorm:"type:varchar;not null" json:"token_address"`
	AddressType  AddressType `gorm:"type:varchar(10);not null;default:'eoa'" json:"address_type"`
	Balance      *big.Int    `gorm:"type:numeric;not null;default:0;check:balance >= 0;serializer:u256" json:"balance"`
	LockBalance  *big.Int    `gorm:"type:numeric;not null;default:0;serializer:u256" json:"lock_balance"`
	Timestamp    uint64      `gorm:"type:bigint;not null;check:timestamp > 0" json:"timestamp"`
}

type BalancesView interface {
	QueryWalletBalanceByTokenAndAddress(
		requestId string,
		chainName string,
		addressType AddressType,
		address,
		tokenAddress string,
	) (*Balances, error)
}

type BalancesDB interface {
	BalancesView

	UpdateOrCreate(requestId string, chainName string, balances []*TokenBalance) error
	StoreBalances(requestId string, chainName string, balances []*Balances) error
	UpdateBalanceListByTwoAddress(requestId string, chainName string, balances []*Balances) error
	UpdateBalance(requestId string, chainName string, balance *Balances) error
}

type balancesDB struct {
	gorm *gorm.DB
}

func NewBalancesDB(db *gorm.DB) BalancesDB {
	return &balancesDB{gorm: db}
}

func (db *balancesDB) StoreBalances(requestId string, chainName string, balanceList []*Balances) error {
	valueList := make([]Balances, len(balanceList))
	for i, balance := range balanceList {
		if balance != nil {
			valueList[i] = *balance
		}
	}
	tableName := utils.GetTableName("balances", requestId, chainName)
	return db.gorm.Table(tableName).CreateInBatches(&valueList, len(valueList)).Error
}

func (db *balancesDB) UpdateBalance(requestId string, chainName string, balance *Balances) error {
	if balance == nil {
		return fmt.Errorf("balance cannot be nil")
	}

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		tableName := utils.GetTableName("balances", requestId, chainName)
		return db.UpdateAndSaveBalance(tx, tableName, balance)
	})
}

func (db *balancesDB) UpdateAndSaveBalance(tx *gorm.DB, tableName string, balance *Balances) error {
	if balance == nil {
		return fmt.Errorf("balance cannot be nil")
	}

	var currentBalance Balances
	result := tx.Table(tableName).
		Where("address = ? AND token_address = ?",
			strings.ToLower(balance.Address),
			strings.ToLower(balance.TokenAddress),
		).
		Take(&currentBalance)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Debug("Balance record not found",
				"tableName", tableName,
				"address", balance.Address,
				"tokenAddress", balance.TokenAddress)
			return nil
		}
		return fmt.Errorf("query balance failed: %w", result.Error)
	}

	currentBalance.Balance = balance.Balance //上游修改这里不做重复计算
	currentBalance.LockBalance = new(big.Int).Add(currentBalance.LockBalance, balance.LockBalance)
	currentBalance.Timestamp = uint64(time.Now().Unix())

	if err := tx.Table(tableName).Save(&currentBalance).Error; err != nil {
		log.Error("Failed to save balance",
			"tableName", tableName,
			"address", balance.Address,
			"error", err)
		return fmt.Errorf("save balance failed: %w", err)
	}

	log.Debug("Balance updated and saved successfully",
		"tableName", tableName,
		"address", balance.Address,
		"tokenAddress", balance.TokenAddress,
		"newBalance", currentBalance.Balance.String(),
		"lockBalance", currentBalance.LockBalance.String())

	return nil
}

func (db *balancesDB) UpdateBalanceListByTwoAddress(requestId string, chainName string, balanceList []*Balances) error {
	if len(balanceList) == 0 {
		return nil
	}

	tableName := utils.GetTableName("balances", requestId, chainName)
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, balance := range balanceList {
			var currentBalance Balances
			result := tx.Table(tableName).
				Where("address = ? AND token_address = ?",
					balance.Address,
					balance.TokenAddress).
				Take(&currentBalance)

			if result.Error != nil {
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					continue
				}
				return fmt.Errorf("query balance failed: %w", result.Error)
			}

			currentBalance.Balance = new(big.Int).Sub(currentBalance.Balance, balance.LockBalance)
			currentBalance.LockBalance = balance.LockBalance
			currentBalance.Timestamp = uint64(time.Now().Unix())

			if err := tx.Table(tableName).Save(&currentBalance).Error; err != nil {
				return fmt.Errorf("save balance failed: %w", err)
			}
		}
		return nil
	})
}

func (db *balancesDB) QueryWalletBalanceByTokenAndAddress(
	requestId string,
	chainName string,
	addressType AddressType,
	address,
	tokenAddress string,
) (*Balances, error) {
	balance, err := db.queryBalance(requestId, chainName, address, tokenAddress)
	if err == nil {
		return balance, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.createInitialBalance(requestId, chainName, addressType, address, tokenAddress)
	}

	return nil, fmt.Errorf("query balance failed: %w", err)
}

func (db *balancesDB) queryBalance(
	requestId string,
	chainName string,
	address,
	tokenAddress string,
) (*Balances, error) {
	var balance Balances
	tableName := utils.GetTableName("balances", requestId, chainName)
	err := db.gorm.Table(tableName).
		Where("address = ? AND token_address = ?",
			strings.ToLower(address),
			strings.ToLower(tokenAddress),
		).
		Take(&balance).
		Error

	if err != nil {
		return nil, err
	}

	return &balance, nil
}

func (db *balancesDB) createInitialBalance(
	requestId string,
	chainName string,
	addressType AddressType,
	address,
	tokenAddress string,
) (*Balances, error) {
	balance := &Balances{
		GUID:         uuid.New(),
		Address:      address,
		TokenAddress: tokenAddress,
		AddressType:  addressType,
		Balance:      big.NewInt(0),
		LockBalance:  big.NewInt(0),
		Timestamp:    uint64(time.Now().Unix()),
	}

	tableName := utils.GetTableName("balances", requestId, chainName)
	if err := db.gorm.Table(tableName).Create(balance).Error; err != nil {
		log.Error("Failed to create initial balance",
			"tableName", tableName,
			"address", address,
			"tokenAddress", tokenAddress,
			"error", err,
		)
		return nil, fmt.Errorf("create initial balance failed: %w", err)
	}

	log.Debug("Created initial balance",
		"tableName", tableName,
		"address", address,
		"tokenAddress", tokenAddress,
	)

	return balance, nil
}

func (db *balancesDB) UpdateOrCreate(requestId string, chainName string, balanceList []*TokenBalance) error {
	if len(balanceList) == 0 {
		return nil
	}

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, balance := range balanceList {
			log.Info("Processing balance update",
				"txType", balance.TxType,
				"from", balance.FromAddress,
				"to", balance.ToAddress,
				"token", balance.TokenAddress,
				"amount", balance.Balance)

			if err := db.handleBalanceUpdate(tx, requestId, chainName, balance); err != nil {
				return fmt.Errorf("failed to handle balance update: %w", err)
			}
		}
		return nil
	})
}

func (db *balancesDB) handleBalanceUpdate(tx *gorm.DB, requestId string, chainName string, balance *TokenBalance) error {
	switch balance.TxType {
	case TxTypeDeposit:
		return db.handleDeposit(tx, requestId, chainName, balance)
	case TxTypeWithdraw:
		return db.handleWithdraw(tx, requestId, chainName, balance)
	case TxTypeCollection:
		return db.handleCollection(tx, requestId, chainName, balance)
	case TxTypeHot2Cold:
		return db.handleHotToCold(tx, requestId, chainName, balance)
	case TxTypeCold2Hot:
		return db.handleColdToHot(tx, requestId, chainName, balance)
	default:
		return fmt.Errorf("unsupported transaction type: %s", balance.TxType)
	}
}

func (db *balancesDB) handleDeposit(tx *gorm.DB, requestId string, chainName string, balance *TokenBalance) error {
	userAddress, err := db.QueryWalletBalanceByTokenAndAddress(requestId, chainName, AddressTypeEOA, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query user address failed", "err", err)
		return err
	}
	log.Info("Processing handleDeposit",
		"txType", balance.TxType,
		"from", balance.FromAddress,
		"to", balance.ToAddress,
		"token", balance.TokenAddress,
		"amount", balance.Balance,
		"userAddress.Balance,", userAddress.Balance)
	userAddress.Balance = new(big.Int).Add(userAddress.Balance, balance.Balance)
	log.Info("userAddress.Balance after", new(big.Int).Add(userAddress.Balance, balance.Balance))
	tableName := utils.GetTableName("balances", requestId, chainName)
	return db.UpdateAndSaveBalance(tx, tableName, userAddress)
}

func (db *balancesDB) handleWithdraw(tx *gorm.DB, requestId string, chainName string, balance *TokenBalance) error {
	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, chainName, AddressTypeHot, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet failed", "err", err)
		return err
	}

	hotWallet.Balance = new(big.Int).Sub(hotWallet.Balance, balance.Balance)
	tableName := utils.GetTableName("balances", requestId, chainName)
	return db.UpdateAndSaveBalance(tx, tableName, hotWallet)
}

func (db *balancesDB) handleCollection(tx *gorm.DB, requestId string, chainName string, balance *TokenBalance) error {
	userWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, chainName, AddressTypeEOA, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query user wallet failed", "err", err)
		return err
	}
	userWallet.Balance = new(big.Int).Sub(userWallet.Balance, balance.Balance)
	tableName := utils.GetTableName("balances", requestId, chainName)
	if err := db.UpdateAndSaveBalance(tx, tableName, userWallet); err != nil {
		return err
	}

	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, chainName, AddressTypeHot, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet failed", "err", err)
		return err
	}
	hotWallet.Balance = new(big.Int).Add(hotWallet.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, tableName, hotWallet)
}

func (db *balancesDB) handleHotToCold(tx *gorm.DB, requestId string, chainName string, balance *TokenBalance) error {
	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, chainName, AddressTypeHot, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet failed", "err", err)
		return err
	}
	hotWallet.Balance = new(big.Int).Sub(hotWallet.Balance, balance.Balance)
	tableName := utils.GetTableName("balances", requestId, chainName)
	if err := db.UpdateAndSaveBalance(tx, tableName, hotWallet); err != nil {
		return err
	}

	coldWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, chainName, AddressTypeCold, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query cold wallet failed", "err", err)
		return err
	}
	coldWallet.Balance = new(big.Int).Add(coldWallet.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, tableName, coldWallet)
}

func (db *balancesDB) handleColdToHot(tx *gorm.DB, requestId string, chainName string, balance *TokenBalance) error {
	coldWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, chainName, AddressTypeCold, balance.FromAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query cold wallet failed", "err", err)
		return err
	}
	coldWallet.Balance = new(big.Int).Sub(coldWallet.Balance, balance.Balance)
	tableName := utils.GetTableName("balances", requestId, chainName)
	if err := db.UpdateAndSaveBalance(tx, tableName, coldWallet); err != nil {
		return err
	}

	hotWallet, err := db.QueryWalletBalanceByTokenAndAddress(requestId, chainName, AddressTypeHot, balance.ToAddress, balance.TokenAddress)
	if err != nil {
		log.Error("Query hot wallet failed", "err", err)
		return err
	}
	hotWallet.Balance = new(big.Int).Add(hotWallet.Balance, balance.Balance)
	return db.UpdateAndSaveBalance(tx, tableName, hotWallet)
}
