package database

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

type Internals struct {
	// 基础信息
	GUID      uuid.UUID `gorm:"primaryKey" json:"guid"`
	Timestamp uint64    `json:"timestamp"`
	Status    TxStatus  `json:"status" gorm:"column:status"`

	// 区块信息
	BlockHash   common.Hash `gorm:"column:block_hash;serializer:bytes" json:"block_hash"`
	BlockNumber *big.Int    `gorm:"serializer:u256;column:block_number" json:"block_number"`
	TxHash      common.Hash `gorm:"column:hash;serializer:bytes" json:"hash"`

	// 交易基础信息
	FromAddress common.Address `json:"from_address" gorm:"serializer:bytes;column:from_address"`
	ToAddress   common.Address `json:"to_address" gorm:"serializer:bytes;column:to_address"`
	Amount      *big.Int       `gorm:"serializer:u256;column:amount" json:"amount"`

	// Gas 费用
	GasLimit             uint64 `json:"gas_limit"`
	MaxFeePerGas         string `json:"max_fee_per_gas"`
	MaxPriorityFeePerGas string `json:"max_priority_fee_per_gas"`

	// Token 相关信息
	TokenType    TokenType      `json:"token_type" gorm:"column:token_type"` // ETH, ERC20, ERC721, ERC1155
	TokenAddress common.Address `json:"token_address" gorm:"serializer:bytes;column:token_address"`
	TokenId      string         `json:"token_id" gorm:"column:token_id"`     // ERC721/ERC1155 的 token ID
	TokenMeta    string         `json:"token_meta" gorm:"column:token_meta"` // Token 元数据

	// 交易签名
	TxSignHex string `json:"tx_sign_hex" gorm:"column:tx_sign_hex"`
}

type InternalsView interface {
	QueryNotifyInternal(requestId string) ([]Internals, error)
	QueryInternalsByHash(requestId string, txId string) (*Internals, error)
	UnSendInternalsList(requestId string) ([]Internals, error)
}

type InternalsDB interface {
	InternalsView

	StoreInternal(string, *Internals) error
	UpdateInternalTx(requestId string, transactionId string, signedTx string, status TxStatus) error
	UpdateInternalStatus(requestId string, status TxStatus, internalsList []Internals) error
}

type internalsDB struct {
	gorm *gorm.DB
}

func NewInternalsDB(db *gorm.DB) InternalsDB {
	return &internalsDB{gorm: db}
}

func (db *internalsDB) QueryNotifyInternal(requestId string) ([]Internals, error) {
	var notifyInternals []Internals
	result := db.gorm.Table("internals_"+requestId).
		Where("status = ?", TxStatusWalletDone).
		Find(&notifyInternals)
	if result.Error != nil {
		return nil, result.Error
	}
	return notifyInternals, nil
}

func (db *internalsDB) StoreInternal(requestId string, internals *Internals) error {
	return db.gorm.Table("internals_" + requestId).Create(internals).Error
}

func (db *internalsDB) QueryInternalsByHash(requestId string, txId string) (*Internals, error) {
	var internalsEntity Internals
	result := db.gorm.Table("internals_"+requestId).
		Where("guid = ?", txId).
		Take(&internalsEntity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &internalsEntity, nil
}

func (db *internalsDB) UnSendInternalsList(requestId string) ([]Internals, error) {
	var internalsList []Internals
	err := db.gorm.Table("internals_"+requestId).
		Where("status = ?", TxStatusSigned).
		Find(&internalsList).Error
	if err != nil {
		return nil, err
	}
	return internalsList, nil
}

type GasInfo struct {
	GasLimit             uint64
	MaxFeePerGas         string
	MaxPriorityFeePerGas string
}

func (db *internalsDB) UpdateInternalTx(requestId string, transactionId string, signedTx string, status TxStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if signedTx != "" {
		updates["tx_sign_hex"] = signedTx
	}

	result := db.gorm.Table("internals_"+requestId).
		Where("guid = ?", transactionId).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (db *internalsDB) UpdateInternalStatus(requestId string, status TxStatus, internalsList []Internals) error {
	if len(internalsList) == 0 {
		return nil
	}
	tableName := fmt.Sprintf("internals_%s", requestId)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var guids []uuid.UUID
		for _, internal := range internalsList {
			guids = append(guids, internal.GUID)
		}

		result := tx.Table(tableName).
			Where("guid IN ?", guids).
			Where("status = ?", TxStatusWalletDone).
			Update("status", status)

		if result.Error != nil {
			return fmt.Errorf("batch update status failed: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			log.Warn("No internals updated",
				"requestId", requestId,
				"expectedCount", len(internalsList),
			)
		}

		log.Info("Batch update internals status success",
			"requestId", requestId,
			"count", result.RowsAffected,
			"status", status,
		)

		return nil
	})
}
