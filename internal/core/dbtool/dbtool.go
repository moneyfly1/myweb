// Package dbtool 提供 cmd/ 与 scripts/ 下运维工具共用的「开库 / 复制 / 只读打开」实现。
//
// 为什么单独成包：这些工具此前各自抄了一份完全相同的样板
// （config.LoadConfig → database.InitDatabase → GetDB），7 份逐行相同；
// 抄出来的第二份迟早与主流程漂移（例如 db_preflight 手写的路径解析就漏了
// filepath.Join(".", ...) 分支，会读到另一个库还看不出来）。
package dbtool

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"

	"gorm.io/driver/sqlite"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open 加载配置并连接数据库（运维工具通用入口）。调用方负责在结束时 Close。
func Open() (*gorm.DB, error) {
	if _, err := config.LoadConfig(); err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}
	return openInitialized()
}

// OpenWithMigrate 同 Open，但额外执行一次 AutoMigrate（需要写库结构的工具用）。
func OpenWithMigrate() (*gorm.DB, error) {
	db, err := Open()
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	return db, nil
}

func openInitialized() (*gorm.DB, error) {
	if err := database.InitDatabase(); err != nil {
		return nil, fmt.Errorf("数据库初始化失败: %w", err)
	}
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	return db, nil
}

// OpenReadOnly 以只读方式打开一个 SQLite 文件（物理上杜绝误写）。
// 路径会做一次 Clean 并拒绝包含 ".." 的相对跳转。
func OpenReadOnly(path string) (*gorm.DB, error) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" || clean == "." {
		return nil, fmt.Errorf("无效的数据库路径")
	}
	dsn := fmt.Sprintf("file:%s?mode=ro", clean)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("以只读方式打开数据库失败(%s): %w", dsn, err)
	}
	return db, nil
}

// CopyFile 复制文件并 fsync（备份/恢复场景不能依赖页缓存及时落盘）。
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("写入目标文件失败: %w", err)
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return fmt.Errorf("刷盘失败: %w", err)
	}
	return out.Close()
}

// ResolveSQLitePath 把 DATABASE_URL 解析成本地 SQLite 路径（委托数据库层，避免再抄一份规则）。
func ResolveSQLitePath(databaseURL string) string {
	return database.ResolveSQLitePath(databaseURL)
}
