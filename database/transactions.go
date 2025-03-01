package database

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dapplink-labs/multichain-sync-account/database/utils"
	"github.com/dapplink-labs/multichain-sync-account/rpcclient/chain-account/account"
)

type Transactions struct {
	GUID         uuid.UUID        `gorm:"primaryKey" json:"guid"`
	BlockHash    common.Hash      `gorm:"column:block_hash;serializer:bytes"  db:"block_hash" json:"block_hash"`
	BlockNumber  *big.Int         `gorm:"serializer:u256;column:block_number" db:"block_number" json:"BlockNumber" form:"block_number"`
	Hash         common.Hash      `gorm:"column:hash;serializer:bytes"  db:"hash" json:"hash"`
	FromAddress  string           `json:"from_address" gorm:"type:varchar;not null"`
	ToAddress    string           `json:"to_address" gorm:"type:varchar;not null"`
	TokenAddress string           `json:"token_address" gorm:"type:varchar"`
	TokenId      string           `json:"token_id" gorm:"column:token_id"`
	TokenMeta    string           `json:"token_meta" gorm:"column:token_meta"`
	Fee          *big.Int         `gorm:"serializer:u256;column:fee" db:"fee" json:"Fee" form:"fee"`
	Amount       *big.Int         `gorm:"serializer:u256;column:amount" db:"amount" json:"Amount" form:"amount"`
	Status       account.TxStatus `json:"status"`
	TxType       TransactionType  `json:"tx_type" gorm:"column:tx_type"`
	Timestamp    uint64
}

type TransactionsView interface {
	QueryTransactionByHash(requestId string, chainName string, hash common.Hash) (*Transactions, error)
}

type TransactionsDB interface {
	TransactionsView

	StoreTransactions(requestId string, chainName string, transactionsList []*Transactions, transactionsLength uint64) error
	UpdateTransactionsStatus(requestId string, chainName string, blockNumber *big.Int) error
	UpdateTransactionStatus(requestId string, txList []*Transactions) error
}

type transactionsDB struct {
	gorm *gorm.DB
}

func (db *transactionsDB) QueryTransactionByHash(requestId string, chainName string, hash common.Hash) (*Transactions, error) {
	tableName := utils.GetTableName("transactions", requestId, chainName)
	var transactionEntry Transactions
	result := db.gorm.Table(tableName).Where("hash", hash.String()).Take(&transactionEntry)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &transactionEntry, nil
}

func (db *transactionsDB) UpdateTransactionsStatus(requestId string, chainName string, blockNumber *big.Int) error {
	tableName := utils.GetTableName("transactions", requestId, chainName)
	result := db.gorm.Table(tableName).Where("status = ? and block_number = ?", 0, blockNumber).Updates(map[string]interface{}{"status": gorm.Expr("GREATEST(1)")})
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		return result.Error
	}
	return nil
}

func NewTransactionsDB(db *gorm.DB) TransactionsDB {
	return &transactionsDB{gorm: db}
}

func (db *transactionsDB) StoreTransactions(requestId string, chainName string, transactionsList []*Transactions, transactionsLength uint64) error {
	tableName := utils.GetTableName("transactions", requestId, chainName)
	result := db.gorm.Table(tableName).CreateInBatches(transactionsList, int(transactionsLength))
	return result.Error
}

func (db *transactionsDB) UpdateTransactionStatus(requestId string, txList []*Transactions) error {
	for i := 0; i < len(txList); i++ {
		var transactionSingle = Transactions{}

		result := db.gorm.Table("transactions_" + requestId).Where(&Transactions{Hash: txList[i].Hash}).Take(&transactionSingle)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil
			}
			return result.Error
		}
		transactionSingle.Status = txList[i].Status
		transactionSingle.Fee = txList[i].Fee
		err := db.gorm.Table("transactions_" + requestId).Save(&transactionSingle).Error
		if err != nil {
			return err
		}
	}
	return nil
}
