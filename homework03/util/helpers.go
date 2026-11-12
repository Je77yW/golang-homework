package util

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// OpenSQLite 打开工作目录下的 db/homework03.db，并在需要时创建目录。
func OpenSQLite() (*gorm.DB, error) {
	const directory = "db"
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, fmt.Errorf("创建数据库目录: %w", err)
	}

	path := filepath.Join(directory, "homework03.db")
	db, err := gorm.Open(sqlite.Open(path+"?_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 数据库 %s: %w", path, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	return db, nil
}
