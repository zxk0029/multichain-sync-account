package database

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/dapplink-labs/multichain-sync-account/database/utils"
)

type Addresses struct {
	GUID        uuid.UUID   `gorm:"primary_key" json:"guid"`
	Address     string      `gorm:"type:varchar;unique;not null" json:"address"`
	AddressType AddressType `gorm:"type:varchar(10);not null;default:'eoa'" json:"address_type"`
	PublicKey   string      `gorm:"type:varchar;not null" json:"public_key"`
	Timestamp   uint64      `gorm:"type:bigint;not null;check:timestamp > 0" json:"timestamp"`
}

type AddressesView interface {
	AddressExist(requestId string, chainName string, address string) (bool, AddressType)
	QueryAddressesByToAddress(requestId string, chainName string, address string) (*Addresses, error)
	QueryHotWalletInfo(requestId string, chainName string) (*Addresses, error)
	QueryColdWalletInfo(requestId string, chainName string) (*Addresses, error)
	GetAllAddresses(requestId string, chainName string) ([]*Addresses, error)
}

type AddressesDB interface {
	AddressesView

	StoreAddresses(requestId string, chainName string, addresses []*Addresses) error
}

func NewAddressesDB(db *gorm.DB) AddressesDB {
	return &addressesDB{gorm: db}
}

type addressesDB struct {
	gorm *gorm.DB
}

func (db *addressesDB) AddressExist(requestId string, chainName string, address string) (bool, AddressType) {
	var addressEntry Addresses
	tableName := utils.GetTableName("addresses", requestId, chainName)
	err := db.gorm.Table(tableName).
		Where("address = ?", strings.ToLower(address)).
		First(&addressEntry).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, AddressTypeEOA
		}
		return false, AddressTypeEOA
	}
	return true, addressEntry.AddressType
}

func (db *addressesDB) QueryAddressesByToAddress(requestId string, chainName string, address string) (*Addresses, error) {
	var addressEntry Addresses
	tableName := utils.GetTableName("addresses", requestId, chainName)
	err := db.gorm.Table(tableName).
		Where("address = ?", strings.ToLower(address)).
		Take(&addressEntry).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &addressEntry, nil
}

// StoreAddresses store address, if address already exists, do nothing
func (db *addressesDB) StoreAddresses(requestId string, chainName string, addressList []*Addresses) error {
	tableName := utils.GetTableName("addresses", requestId, chainName)

	// 使用 OnConflict.DoNothing() 在冲突时不更新
	return db.gorm.Table(tableName).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "address"}},
			DoNothing: true, // 当记录存在时不做任何操作
		}).
		CreateInBatches(addressList, len(addressList)).Error
}

func (db *addressesDB) QueryHotWalletInfo(requestId string, chainName string) (*Addresses, error) {
	var addressEntry Addresses
	tableName := utils.GetTableName("addresses", requestId, chainName)
	err := db.gorm.Table(tableName).
		Where("address_type = ?", AddressTypeHot).
		Take(&addressEntry).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &addressEntry, nil
}

func (db *addressesDB) QueryColdWalletInfo(requestId string, chainName string) (*Addresses, error) {
	var addressEntry Addresses
	tableName := utils.GetTableName("addresses", requestId, chainName)
	err := db.gorm.Table(tableName).
		Where("address_type = ?", AddressTypeCold).
		Take(&addressEntry).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &addressEntry, nil
}

func (db *addressesDB) GetAllAddresses(requestId string, chainName string) ([]*Addresses, error) {
	var addresses []*Addresses
	tableName := utils.GetTableName("addresses", requestId, chainName)
	err := db.gorm.Table(tableName).Find(&addresses).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return addresses, nil
}

func (a *Addresses) Validate() error {
	if a.Address == (common.Address{}.String()) {
		return errors.New("invalid address")
	}
	if a.PublicKey == "" {
		return errors.New("invalid public key")
	}
	if a.Timestamp == 0 {
		return errors.New("invalid timestamp")
	}
	switch a.AddressType {
	case AddressTypeEOA, AddressTypeHot, AddressTypeCold:
		return nil
	default:
		return errors.New("invalid address type")
	}
}
