package database

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/ethereum/go-ethereum/log"
)

type CreateTableDB interface {
	CreateTable(tableName, realTableName string) error
}

type createTableDB struct {
	gorm *gorm.DB
}

func NewCreateTableDB(db *gorm.DB) CreateTableDB {
	return &createTableDB{gorm: db}
}

func (dao *createTableDB) CreateTable(tableName, realTableName string) error {
	err := dao.gorm.Exec("CREATE TABLE IF NOT EXISTS " + tableName + "(like " + realTableName + " including all)").Error
	if err != nil {
		log.Error("create table from base table fail", "err", err)
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}
	return nil
}
