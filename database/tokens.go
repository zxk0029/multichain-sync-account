package database

import (
	"errors"
	"math/big"

	"gorm.io/gorm"

	"github.com/google/uuid"

	"github.com/dapplink-labs/multichain-sync-account/database/utils"
)

type Tokens struct {
	GUID          uuid.UUID `gorm:"primaryKey" json:"guid"`
	TokenAddress  string    `gorm:"type:varchar" json:"token_address"`
	Decimals      uint8     `json:"uint"`
	TokenName     string    `json:"tokens_name"`
	CollectAmount *big.Int  `gorm:"serializer:u256" json:"collect_amount"`
	ColdAmount    *big.Int  `gorm:"serializer:u256" json:"cold_amount"`
	Timestamp     uint64    `json:"timestamp"`
}

type TokensView interface {
	TokensInfoByAddress(requestId string, chainName string, address string) (*Tokens, error)
}

type TokensDB interface {
	TokensView

	StoreTokens(requestId string, chainName string, tokens []Tokens) error
}

type tokensDB struct {
	gorm *gorm.DB
}

func NewTokensDB(db *gorm.DB) TokensDB {
	return &tokensDB{gorm: db}
}

func (db *tokensDB) StoreTokens(requestId string, chainName string, tokenList []Tokens) error {
	tableName := utils.GetTableName("tokens", requestId, chainName)
	return db.gorm.Table(tableName).CreateInBatches(&tokenList, len(tokenList)).Error
}

func (db *tokensDB) TokensInfoByAddress(requestId string, chainName string, address string) (*Tokens, error) {
	var tokensEntry Tokens
	tableName := utils.GetTableName("tokens", requestId, chainName)
	err := db.gorm.Table(tableName).Where("token_address", address).Take(&tokensEntry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tokensEntry, nil
}
