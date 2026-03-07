package utils

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RollbackTransaction 安全地回滚事务并记录错误。
func RollbackTransaction(tx *gorm.DB) {
	if err := tx.Rollback().Error; err != nil && err != gorm.ErrInvalidTransaction {
		zap.L().Error("事务回滚失败", zap.Error(err))
	}
}
