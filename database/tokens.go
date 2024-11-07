package database

import (
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"

	common2 "github.com/dapplink-labs/multichain-sync-account/database/utils"
	"github.com/ethereum/go-ethereum/common"
)

type Tokens struct {
	GUID            uuid.UUID      `gorm:"primaryKey" json:"guid"`
	TokenAddress    common.Address `json:"token_address" gorm:"serializer:bytes"`
	Decimals        uint8          `json:"uint"`
	TokenName       string         `json:"tokens_name"`
	CollectionLimit uint64         `json:"collection_limit"`
	Timestamp       uint64         `json:"timestamp"`
}

type TokensView interface {
	TokensInfoByAddress(string) (*Tokens, error)
}

type TokensDB interface {
	TokensView

	StoreTokens([]Tokens, uint64) error
}

type tokensDB struct {
	gorm *gorm.DB
}

func NewTokensDB(db *gorm.DB) TokensDB {
	return &tokensDB{gorm: db}
}

func (db *tokensDB) StoreTokens(headers []Tokens, blockLength uint64) error {
	result := db.gorm.CreateInBatches(&headers, common2.BatchInsertSize)
	return result.Error
}

func (db *tokensDB) TokensInfoByAddress(address string) (*Tokens, error) {
	var tokensEntry Tokens
	err := db.gorm.Table("tokens").Where("token_address", address).Take(&tokensEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tokensEntry, nil
}
