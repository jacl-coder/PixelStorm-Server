package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jacl-coder/PixelStorm-Server/config"
	_ "github.com/lib/pq"
)

var (
	// DB 全局数据库连接实例
	DB *sql.DB
)

// InitPostgres 初始化PostgreSQL连接
func InitPostgres() error {
	dsn := config.GlobalConfig.Database.GetDSN()
	var err error

	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 测试连接
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("数据库Ping失败: %w", err)
	}

	log.Println("成功连接到PostgreSQL数据库")
	return nil
}

// Close 关闭数据库连接
func Close() {
	if DB != nil {
		DB.Close()
		log.Println("数据库连接已关闭")
	}
}
