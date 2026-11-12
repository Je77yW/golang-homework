package util

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const DefaultDBPath = "db/blog.db"

// OpenDB 创建数据库目录并打开 SQLite 文件；调用者负责关闭底层连接。
func OpenDB(path string) (*gorm.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("database path must not be empty")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	// URI 转义避免文件名中的 ?、# 等字符被解释为连接参数。
	dsn := url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}
	query := dsn.Query()
	query.Set("_foreign_keys", "on")
	query.Set("_busy_timeout", "5000")
	dsn.RawQuery = query.Encode()
	db, err := gorm.Open(sqlite.Open(dsn.String()), &gorm.Config{
		TranslateError: true,
		Logger: logger.New(log.New(os.Stderr, "gorm ", log.LstdFlags), logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sqlite connection: %w", err)
	}
	// SQLite 同时只有一个写者，本作业使用单连接避免并发写锁争用。
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}
	return db, nil
}
