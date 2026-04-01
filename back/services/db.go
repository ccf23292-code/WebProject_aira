package services

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitPostgres 初始化 PostgreSQL 连接，优先读取 DATABASE_URL。
func InitPostgres() (*gorm.DB, error) {
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		return nil, fmt.Errorf("missing DATABASE_URL or POSTGRES_DSN")
	}
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
