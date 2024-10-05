package database

import (
	"errors"
	"gorm.io/gorm"
	"strings"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/common"
)

type Addresses struct {
	GUID        uuid.UUID      `gorm:"primaryKey" json:"guid"`
	BusinessUid string         `json:"business_uid"`
	Address     common.Address `json:"address" gorm:"serializer:bytes"`
	AddressType uint8          `json:"address_type"` //0:用户地址；1:热钱包地址(归集地址)；2:冷钱包地址
	PublicKey   string         `json:"public_key"`
	Timestamp   uint64
}

type AddressesView interface {
	QueryAddressesByToAddress(string, *common.Address) (*Addresses, error)
	QueryHotWalletInfo(string) (*Addresses, error)
	QueryColdWalletInfo(string) (*Addresses, error)
}

type AddressesDB interface {
	AddressesView

	StoreAddresses(string, []Addresses, uint64) error
}

type addressesDB struct {
	gorm *gorm.DB
}

func (db *addressesDB) QueryAddressesByToAddress(requestId string, address *common.Address) (*Addresses, error) {
	var addressEntry Addresses
	err := db.gorm.Table("addresses_"+requestId).Where("address", strings.ToLower(address.String())).Take(&addressEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &addressEntry, nil
}

func NewAddressesDB(db *gorm.DB) AddressesDB {
	return &addressesDB{gorm: db}
}

// StoreAddresses 存储地址
func (db *addressesDB) StoreAddresses(requestId string, addressList []Addresses, addressLength uint64) error {
	result := db.gorm.Table("addresses_"+requestId).CreateInBatches(&addressList, int(addressLength))
	return result.Error
}

func (db *addressesDB) QueryHotWalletInfo(requestId string) (*Addresses, error) {
	var addressEntry Addresses
	err := db.gorm.Table("addresses_"+requestId).Where("address_type", 1).Take(&addressEntry).Error
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
	err := db.gorm.Table("addresses_"+requestId).Where("address_type", 2).Take(&addressEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &addressEntry, nil
}
