package database

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"streetlight/internal/config"
)

// Connect 根据配置建立数据库连接, 支持 sqlite(默认, 零依赖) 与 postgres。
func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	options := &gorm.Config{
		Logger:         logger.Default.LogMode(logLevel(cfg.LogLevel)),
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		NowFunc:        func() time.Time { return time.Now().Local() },
	}

	switch cfg.Driver {
	case "sqlite":
		if err := ensureDir(cfg.DSN); err != nil {
			return nil, err
		}
		return gorm.Open(sqlite.Open(sqliteDSN(cfg.DSN)), options)
	case "postgres":
		return gorm.Open(postgres.Open(cfg.DSN), options)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.Driver)
	}
}

// AutoMigrate 按模块注册的模型执行自动迁移。
func AutoMigrate(db *gorm.DB, models []any) error {
	if len(models) == 0 {
		return nil
	}
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("自动迁移数据表失败: %w", err)
	}
	slog.Info("数据表迁移完成", "模型数量", len(models))
	return nil
}

// Close 关闭底层数据库连接。
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// sqliteDSN 补齐 sqlite 连接参数(WAL / 外键 / 忙等待), 提升并发读写稳定性。
func sqliteDSN(dsn string) string {
	if strings.Contains(dsn, "?") {
		return dsn
	}
	if !strings.HasPrefix(dsn, "file:") {
		dsn = "file:" + dsn
	}
	return dsn + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
}

// ensureDir 保证 sqlite 数据文件所在目录存在。
func ensureDir(dsn string) error {
	path := strings.TrimPrefix(dsn, "file:")
	if index := strings.Index(path, "?"); index >= 0 {
		path = path[:index]
	}
	if path == "" || path == ":memory:" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建数据目录 %s 失败: %w", dir, err)
	}
	return nil
}

func logLevel(level string) logger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "info":
		return logger.Info
	default:
		return logger.Warn
	}
}
