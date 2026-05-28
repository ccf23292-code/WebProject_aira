package services

import (
	"fmt"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitPostgres 初始化数据库连接。
// 优先使用 DATABASE_URL（PostgreSQL）；未设置时自动使用本地 SQLite 文件。
func InitPostgres() (*gorm.DB, error) {
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		// 本地开发：使用 SQLite，数据文件存在 back/aira.db
		dbPath := os.Getenv("SQLITE_PATH")
		if dbPath == "" {
			dbPath = "aira.db"
		}
		fmt.Println("[db] 使用 SQLite:", dbPath)
		return gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	}

	// 生产环境：使用 PostgreSQL
	fmt.Println("[db] 使用 PostgreSQL")
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
