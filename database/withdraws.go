package database

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Withdraws struct {
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

type WithdrawsView interface {
	QueryNotifyWithdraws(requestId string) ([]Withdraws, error)
	QueryWithdrawsByHash(requestId string, txId string) (*Withdraws, error)
	UnSendWithdrawsList(requestId string) ([]Withdraws, error)

	SubmitWithdrawFromBusiness(requestId string, withdraw *Withdraws) error
}

type WithdrawsDB interface {
	WithdrawsView

	StoreWithdraw(string, *Withdraws) error
	UpdateWithdrawTx(requestId string, transactionId string, signedTx string, status TxStatus) error
	UpdateWithdrawStatus(requestId string, status TxStatus, withdrawsList []Withdraws) error
}

type withdrawsDB struct {
	gorm *gorm.DB
}

func (db *withdrawsDB) QueryNotifyWithdraws(requestId string) ([]Withdraws, error) {
	var notifyWithdraws []Withdraws
	result := db.gorm.Table("withdraws_"+requestId).
		Where("status = ?", TxStatusWalletDone).
		Find(&notifyWithdraws)

	if result.Error != nil {
		return nil, fmt.Errorf("query notify withdraws failed: %w", result.Error)
	}

	return notifyWithdraws, nil
}

func (db *withdrawsDB) UnSendWithdrawsList(requestId string) ([]Withdraws, error) {
	var withdrawsList []Withdraws
	err := db.gorm.Table("withdraws_"+requestId).
		Where("status = ?", TxStatusSigned).
		Find(&withdrawsList).Error

	if err != nil {
		return nil, fmt.Errorf("query unsend withdraws failed: %w", err)
	}

	return withdrawsList, nil
}

func (db *withdrawsDB) QueryWithdrawsByHash(requestId string, txId string) (*Withdraws, error) {
	var withdrawsEntity Withdraws
	result := db.gorm.Table("withdraws_"+requestId).Where("guid", txId).Take(&withdrawsEntity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &withdrawsEntity, nil
}

func (db *withdrawsDB) SubmitWithdrawFromBusiness(requestId string, withdraw *Withdraws) error {
	// 1. 设置基础字段
	withdraw.GUID = uuid.New()
	withdraw.Timestamp = uint64(time.Now().Unix())
	withdraw.Status = TxStatusUnsigned

	// 2. 初始化区块信息
	withdraw.BlockHash = common.Hash{}
	withdraw.BlockNumber = nil
	withdraw.TxHash = common.Hash{}

	// 3. 初始化签名字段
	withdraw.TxSignHex = ""

	// 4. 表名处理
	tableName := fmt.Sprintf("withdraws_%s", requestId)

	// 5. 保存记录
	if err := db.gorm.Table(tableName).Create(withdraw).Error; err != nil {
		return fmt.Errorf("create withdraw failed: %w", err)
	}

	log.Info("Submit withdraw success",
		"guid", withdraw.GUID,
		"from", withdraw.FromAddress.Hex(),
		"to", withdraw.ToAddress.Hex(),
		"token", withdraw.TokenAddress.Hex(),
		"amount", withdraw.Amount.String(),
		"gasLimit", withdraw.GasLimit,
		"maxFeePerGas", withdraw.MaxFeePerGas,
		"maxPriorityFeePerGas", withdraw.MaxPriorityFeePerGas,
	)

	return nil
}

func (db *withdrawsDB) UpdateWithdrawTx(requestId string, transactionId string, signedTx string, status TxStatus) error {
	tableName := fmt.Sprintf("withdraws_%s", requestId)

	if err := db.CheckWithdrawExists(tableName, transactionId); err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status": status,
	}
	if signedTx != "" {
		updates["tx_sign_hex"] = signedTx
	}

	// 3. 执行更新
	if err := db.gorm.Table(tableName).
		Where("guid = ?", transactionId).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("update withdraw failed: %w", err)
	}

	// 4. 记录日志
	log.Info("Update withdraw success",
		"requestId", requestId,
		"transactionId", transactionId,
		"status", status,
		"updates", updates,
	)
	return nil
}

func NewWithdrawsDB(db *gorm.DB) WithdrawsDB {
	return &withdrawsDB{gorm: db}
}

func (db *withdrawsDB) StoreWithdraw(requestId string, withdrawsList *Withdraws) error {
	result := db.gorm.Table("withdraws_" + requestId).Create(&withdrawsList)
	return result.Error
}

func (db *withdrawsDB) UpdateWithdrawStatus(requestId string, status TxStatus, withdrawsList []Withdraws) error {
	if len(withdrawsList) == 0 {
		return nil
	}
	tableName := fmt.Sprintf("withdraws_%s", requestId)

	return db.gorm.Transaction(func(tx *gorm.DB) error {
		var guids []uuid.UUID
		for _, withdraw := range withdrawsList {
			guids = append(guids, withdraw.GUID)
		}

		result := tx.Table(tableName).
			Where("guid IN ?", guids).
			Where("status = ?", TxStatusWalletDone).
			Update("status", status)

		if result.Error != nil {
			return fmt.Errorf("batch update status failed: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			log.Warn("No withdraws updated",
				"requestId", requestId,
				"expectedCount", len(withdrawsList),
			)
		}

		log.Info("Batch update withdraws status success",
			"requestId", requestId,
			"count", result.RowsAffected,
			"status", status,
		)

		return nil
	})
}

func (db *withdrawsDB) CheckWithdrawExists(tableName string, guid string) error {
	var exist bool
	err := db.gorm.Table(tableName).
		Where("guid = ?", guid).
		Select("1").
		Find(&exist).Error

	if err != nil {
		return fmt.Errorf("check withdraw exist failed: %w", err)
	}

	if !exist {
		return fmt.Errorf("withdraw not found: %s", guid)
	}

	return nil
}
