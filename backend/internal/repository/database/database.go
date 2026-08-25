// 数据库基础设施：统一连接驱动、迁移版本记录和迁移文件执行入口。
package database

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// migrationFiles 将同一版本的 SQL 按数据库驱动隔离，避免自增主键语法互相污染。
//
//go:embed migrations/sqlite/*.sql migrations/postgres/*.sql
var migrationFiles embed.FS

type migrationRecord struct {
	Version   int `gorm:"primaryKey"`
	AppliedAt string
}

func (migrationRecord) TableName() string {
	return "schema_migrations"
}

// Open 根据配置创建 GORM 数据库连接。
func Open(driver, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case "sqlite":
		// SQLite 的写锁需要等待，避免并发用户资源事务直接返回 database is locked。
		if !strings.Contains(dsn, "_busy_timeout") {
			separator := "?"
			if strings.Contains(dsn, "?") {
				separator = "&"
			}
			dsn += separator + "_busy_timeout=5000"
		}
		dialector = sqlite.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败: %w", err)
	}
	return db, nil
}

// Migrate 执行当前驱动尚未应用的版本化 SQL migration。
func Migrate(db *gorm.DB, driver string) error {
	if driver != "sqlite" && driver != "postgres" {
		return fmt.Errorf("不支持的数据库驱动: %s", driver)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if !tx.Migrator().HasTable("schema_migrations") {
			if err := tx.Exec(`CREATE TABLE schema_migrations (
				version INTEGER PRIMARY KEY,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`).Error; err != nil {
				return fmt.Errorf("创建迁移记录表失败: %w", err)
			}
		}

		files, err := listMigrationFiles(driver)
		if err != nil {
			return err
		}
		for _, file := range files {
			var record migrationRecord
			findResult := tx.Where("version = ?", file.version).Limit(1).Find(&record)
			switch {
			case findResult.Error != nil:
				return fmt.Errorf("读取迁移版本 %d 失败: %w", file.version, findResult.Error)
			case findResult.RowsAffected == 1:
				continue
			}

			contents, err := fs.ReadFile(migrationFiles, file.path)
			if err != nil {
				return fmt.Errorf("读取迁移文件 %s 失败: %w", file.path, err)
			}
			if err := tx.Exec(string(contents)).Error; err != nil {
				return fmt.Errorf("执行迁移版本 %d 失败: %w", file.version, err)
			}
			if err := tx.Create(&migrationRecord{Version: file.version}).Error; err != nil {
				return fmt.Errorf("记录迁移版本 %d 失败: %w", file.version, err)
			}
		}
		return nil
	})
}

// EnsureMigrated 用于关闭启动迁移时确认当前数据库已完成该 driver 的全部 migration。
func EnsureMigrated(db *gorm.DB, driver string) error {
	if driver != "sqlite" && driver != "postgres" {
		return fmt.Errorf("不支持的数据库驱动: %s", driver)
	}
	if !db.Migrator().HasTable("schema_migrations") {
		return fmt.Errorf("数据库尚未迁移，请先执行 migrate 命令")
	}
	files, err := listMigrationFiles(driver)
	if err != nil {
		return err
	}
	for _, file := range files {
		var count int64
		if err := db.Model(&migrationRecord{}).Where("version = ?", file.version).Count(&count).Error; err != nil {
			return fmt.Errorf("读取迁移版本 %d 状态失败: %w", file.version, err)
		}
		if count != 1 {
			return fmt.Errorf("数据库缺少 migration %d，请先执行 migrate 命令", file.version)
		}
	}
	return nil
}

type migrationFile struct {
	version int
	path    string
}

func listMigrationFiles(driver string) ([]migrationFile, error) {
	directory := path.Join("migrations", driver)
	entries, err := fs.ReadDir(migrationFiles, directory)
	if err != nil {
		return nil, fmt.Errorf("读取 migration 目录失败: %w", err)
	}

	result := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix := strings.SplitN(entry.Name(), "_", 2)[0]
		version, err := strconv.Atoi(prefix)
		if err != nil {
			return nil, fmt.Errorf("migration 文件名缺少数字版本: %s", entry.Name())
		}
		result = append(result, migrationFile{version: version, path: path.Join(directory, entry.Name())})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].version < result[j].version })
	return result, nil
}
