package dynamic

import (
	"fmt"

	"github.com/dapplink-labs/multichain-sync-account/database"
	"github.com/dapplink-labs/multichain-sync-account/database/utils"
)

func CreateTableFromTemplate(requestId string, chainName string, db *database.DB) error {
	if err := createAddresses(requestId, chainName, db); err != nil {
		return fmt.Errorf("failed to create addresses table: %w", err)
	}
	if err := createTokens(requestId, chainName, db); err != nil {
		return fmt.Errorf("failed to create tokens table: %w", err)
	}
	if err := createBalances(requestId, chainName, db); err != nil {
		return fmt.Errorf("failed to create balances table: %w", err)
	}
	if err := createDeposits(requestId, chainName, db); err != nil {
		return fmt.Errorf("failed to create deposits table: %w", err)
	}
	if err := createTransactions(requestId, chainName, db); err != nil {
		return fmt.Errorf("failed to create transactions table: %w", err)
	}
	if err := createWithdraws(requestId, chainName, db); err != nil {
		return fmt.Errorf("failed to create withdraws table: %w", err)
	}
	if err := createInternals(requestId, chainName, db); err != nil {
		return fmt.Errorf("failed to create internals table: %w", err)
	}
	return nil
}

func createAddresses(requestId string, chainName string, db *database.DB) error {
	tableName := "addresses"
	tableNameByChain := utils.GetTableName(tableName, requestId, chainName)
	return db.CreateTable.CreateTable(tableNameByChain, tableName)
}

func createTokens(requestId string, chainName string, db *database.DB) error {
	tableName := "tokens"
	tableNameByChain := utils.GetTableName(tableName, requestId, chainName)
	return db.CreateTable.CreateTable(tableNameByChain, tableName)
}

func createBalances(requestId string, chainName string, db *database.DB) error {
	tableName := "balances"
	tableNameByChain := utils.GetTableName(tableName, requestId, chainName)
	return db.CreateTable.CreateTable(tableNameByChain, tableName)
}

func createDeposits(requestId string, chainName string, db *database.DB) error {
	tableName := "deposits"
	tableNameByChain := utils.GetTableName(tableName, requestId, chainName)
	return db.CreateTable.CreateTable(tableNameByChain, tableName)
}

func createTransactions(requestId string, chainName string, db *database.DB) error {
	tableName := "transactions"
	tableNameByChain := utils.GetTableName(tableName, requestId, chainName)
	return db.CreateTable.CreateTable(tableNameByChain, tableName)
}

func createWithdraws(requestId string, chainName string, db *database.DB) error {
	tableName := "withdraws"
	tableNameByChain := utils.GetTableName(tableName, requestId, chainName)
	return db.CreateTable.CreateTable(tableNameByChain, tableName)
}

func createInternals(requestId string, chainName string, db *database.DB) error {
	tableName := "internals"
	tableNameByChain := utils.GetTableName(tableName, requestId, chainName)
	return db.CreateTable.CreateTable(tableNameByChain, tableName)
}
