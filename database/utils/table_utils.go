package utils

import (
	"fmt"
	"strings"
)

// GetTableName 生成包含链名的表名
func GetTableName(baseTable string, requestId string, chainName string) string {
	normalizedChain := strings.ToLower(strings.ReplaceAll(chainName, "-", "_"))
	return fmt.Sprintf("%s_%s_%s", baseTable, requestId, normalizedChain)
}
