package common

import (
	"dodevops-api/pkg/db"
	"log"

	"gorm.io/gorm"
)

// GetDB 获取数据库连接（使用pkg/db中的连接）
func GetDB() *gorm.DB {
	if db.Db == nil {
		panic("Database connection is not initialized")
	}

	sqlDB, err := db.Db.DB()
	if err != nil {
		log.Printf("[WARN] Failed to get underlying sql.DB: %v", err)
		return db.Db
	}

	if err := sqlDB.Ping(); err != nil {
		log.Printf("[WARN] Database ping failed (GORM will auto-reconnect): %v", err)
	}

	return db.Db
}

