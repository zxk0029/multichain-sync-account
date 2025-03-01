package database

import (
	"errors"
	"fmt"
	"math/big"

	"gorm.io/gorm"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"

	"github.com/dapplink-labs/multichain-sync-account/database/utils"
)

type Internals struct {
	// 基础信息
	GUID      uuid.UUID `gorm:"primaryKey" json:"guid"`
	Timestamp uint64    `json:"timestamp"`
	Status    TxStatus  `json:"status" gorm:"column:status"`

	// 区块信息
	BlockHash   common.Hash     `gorm:"column:block_hash;serializer:bytes" json:"block_hash"`
	BlockNumber *big.Int        `gorm:"serializer:u256;column:block_number" json:"block_number"`
	TxHash      common.Hash     `gorm:"column:hash;serializer:bytes" json:"hash"`
	TxType      TransactionType `json:"tx_type" gorm:"column:tx_type"`

	// 交易基础信息
	FromAddress string   `json:"from_address" gorm:"column:from_address"`
	ToAddress   string   `json:"to_address" gorm:"column:to_address"`
	Amount      *big.Int `gorm:"serializer:u256;column:amount" json:"amount"`

	// Gas 费用
	GasLimit             uint64 `json:"gas_limit"`
	MaxFeePerGas         string `json:"max_fee_per_gas"`
	MaxPriorityFeePerGas string `json:"max_priority_fee_per_gas"`

	// Token 相关信息
	TokenType    TokenType `json:"token_type" gorm:"column:token_type"` // ETH, ERC20, ERC721, ERC1155
	TokenAddress string    `json:"token_address" gorm:"column:token_address"`
	TokenId      string    `json:"token_id" gorm:"column:token_id"`     // ERC721/ERC1155 的 token ID
	TokenMeta    string    `json:"token_meta" gorm:"column:token_meta"` // Token 元数据

	// 交易签名
	TxSignHex string `json:"tx_sign_hex" gorm:"column:tx_sign_hex"`
}

type InternalsView interface {
	QueryNotifyInternal(requestId string, chainName string) ([]*Internals, error)
	QueryInternalsByTxHash(requestId string, chainName string, txHash common.Hash) (*Internals, error)
	QueryInternalsById(requestId string, chainName string, guid string) (*Internals, error)
	UnSendInternalsList(requestId string, chainName string) ([]*Internals, error)
}

type InternalsDB interface {
	InternalsView

	StoreInternal(requestId string, chainName string, internals *Internals) error
	UpdateInternalByTxHash(requestId string, chainName string, txHash common.Hash, signedTx string, status TxStatus) error
	UpdateInternalById(requestId string, chainName string, id string, signedTx string, status TxStatus) error
	UpdateInternalStatusByTxHash(requestId string, chainName string, status TxStatus, internalsList []*Internals) error
	UpdateInternalListByHash(requestId string, chainName string, internalsList []*Internals) error
	UpdateInternalListById(requestId string, chainName string, internalsList []*Internals) error
}

type internalsDB struct {
	gorm *gorm.DB
}

func NewInternalsDB(db *gorm.DB) InternalsDB {
	return &internalsDB{gorm: db}
}

func (db *internalsDB) QueryNotifyInternal(requestId string, chainName string) ([]*Internals, error) {
	tableName := utils.GetTableName("internals", requestId, chainName)
	var notifyInternals []*Internals
	result := db.gorm.Table(tableName).
		Where("status = ? or status = ?", TxStatusWalletDone, TxStatusNotified).
		Find(&notifyInternals)
	if result.Error != nil {
		return nil, result.Error
	}
	return notifyInternals, nil
}

func (db *internalsDB) StoreInternal(requestId string, chainName string, internals *Internals) error {
	tableName := utils.GetTableName("internals", requestId, chainName)
	return db.gorm.Table(tableName).Create(internals).Error
}

func (db *internalsDB) QueryInternalsByTxHash(requestId string, chainName string, txHash common.Hash) (*Internals, error) {
	tableName := utils.GetTableName("internals", requestId, chainName)
	var internalsEntity Internals
	result := db.gorm.Table(tableName).
		Where("hash = ?", txHash.String()).
		Take(&internalsEntity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &internalsEntity, nil
}

func (db *internalsDB) QueryInternalsById(requestId string, chainName string, guid string) (*Internals, error) {
	tableName := utils.GetTableName("internals", requestId, chainName)
	var internalsEntity Internals
	result := db.gorm.Table(tableName).
		Where("guid = ?", guid).
		Take(&internalsEntity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &internalsEntity, nil
}

func (db *internalsDB) UnSendInternalsList(requestId string, chainName string) ([]*Internals, error) {
	tableName := utils.GetTableName("internals", requestId, chainName)
	var internalsList []*Internals
	err := db.gorm.Table(tableName).
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

func (db *internalsDB) UpdateInternalByTxHash(requestId string, chainName string, txHash common.Hash, signedTx string, status TxStatus) error {
	tableName := utils.GetTableName("internals", requestId, chainName)
	updates := map[string]interface{}{
		"status": status,
	}

	if signedTx != "" {
		updates["tx_sign_hex"] = signedTx
	}

	result := db.gorm.Table(tableName).
		Where("hash = ?", txHash.String()).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (db *internalsDB) UpdateInternalById(requestId string, chainName string, id string, signedTx string, status TxStatus) error {
	tableName := utils.GetTableName("internals", requestId, chainName)
	updates := map[string]interface{}{
		"status": status,
	}

	if signedTx != "" {
		updates["tx_sign_hex"] = signedTx
	}

	result := db.gorm.Table(tableName).
		Where("guid = ?", id).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (db *internalsDB) UpdateInternalStatusByTxHash(requestId string, chainName string, status TxStatus, internalsList []*Internals) error {
	if len(internalsList) == 0 {
		return nil
	}
	tableName := utils.GetTableName("internals", requestId, chainName)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var txHashList []string
		for _, internal := range internalsList {
			txHashList = append(txHashList, internal.TxHash.String())
		}

		result := tx.Table(tableName).
			Where("hash IN ?", txHashList).
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

func (db *internalsDB) UpdateInternalListById(requestId string, chainName string, internalsList []*Internals) error {
	if len(internalsList) == 0 {
		return nil
	}
	tableName := utils.GetTableName("internals", requestId, chainName)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, internal := range internalsList {
			// Update each record individually based on TxHash
			result := tx.Table(tableName).
				Where("guid = ?", internal.GUID.String()).
				Updates(map[string]interface{}{
					"status": internal.Status,
					"amount": internal.Amount,
					"hash":   internal.TxHash.String(),
				})

			// Check for errors in the update operation
			if result.Error != nil {
				return fmt.Errorf("update failed for TxHash %s: %w", internal.TxHash, result.Error)
			}

			// Log a warning if no rows were updated
			if result.RowsAffected == 0 {
				log.Info("No internals updated for TxHash:", "txHash", internal.TxHash)
			} else {
				// Log success message with the number of rows affected
				log.Info("Updated internals for ", "TxHash", internal.TxHash, "status", internal.Status, "amount", internal.Amount)
			}
		}

		return nil
	})
}

func (db *internalsDB) UpdateInternalListByHash(requestId string, chainName string, internalsList []*Internals) error {
	if len(internalsList) == 0 {
		return nil
	}
	tableName := utils.GetTableName("internals", requestId, chainName)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		for _, internal := range internalsList {
			// Update each record individually based on TxHash
			result := tx.Table(tableName).
				Where("hash = ?", internal.TxHash.String()).
				Updates(map[string]interface{}{
					"status": internal.Status,
					"amount": internal.Amount,
				})

			// Check for errors in the update operation
			if result.Error != nil {
				return fmt.Errorf("update failed for TxHash %s: %w", internal.TxHash, result.Error)
			}

			// Log a warning if no rows were updated
			if result.RowsAffected == 0 {
				log.Info("No internals updated for TxHash:", "txHash", internal.TxHash)
			} else {
				// Log success message with the number of rows affected
				log.Info("Updated internals for ", "TxHash", internal.TxHash, "status", internal.Status, "amount", internal.Amount)
			}
		}

		return nil
	})
}
