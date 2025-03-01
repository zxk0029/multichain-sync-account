package database

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"

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
	QueryHotWalletInfo(string) (*Addresses, error)
	QueryColdWalletInfo(string) (*Addresses, error)
	GetAllAddresses(string) ([]*Addresses, error)
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

// StoreAddresses store address
func (db *addressesDB) StoreAddresses(requestId string, chainName string, addressList []*Addresses) error {
	tableName := utils.GetTableName("addresses", requestId, chainName)
	return db.gorm.Table(tableName).CreateInBatches(&addressList, len(addressList)).Error
}

func (db *addressesDB) QueryHotWalletInfo(requestId string) (*Addresses, error) {
	var addressEntry Addresses
	err := db.gorm.Table("addresses_"+requestId).
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

func (db *addressesDB) QueryColdWalletInfo(requestId string) (*Addresses, error) {
	var addressEntry Addresses
	err := db.gorm.Table("addresses_"+requestId).
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

func (db *addressesDB) GetAllAddresses(requestId string) ([]*Addresses, error) {
	var addresses []*Addresses
	err := db.gorm.Table("addresses_" + requestId).Find(&addresses).Error
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
