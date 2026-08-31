// 数据库基础设施：统一连接驱动、迁移版本记录、内容校验和与迁移文件执行入口。
package database

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
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
	Version   int     `gorm:"primaryKey"`
	AppliedAt string  `gorm:"column:applied_at"`
	Checksum  *string `gorm:"column:checksum"`
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

// Migrate 执行当前驱动尚未应用的版本化 SQL migration；执行前若存在待应用版本会自动备份 SQLite 文件。
func Migrate(db *gorm.DB, driver string) error {
	return runMigrations(db, driver, 0, true)
}

// MigrateToVersion 只执行到指定版本为止的迁移（不含该版本之后的内容），
// 供升级演练测试构造“历史形状”的旧库使用。不触发自动备份。
func MigrateToVersion(db *gorm.DB, driver string, maxVersion int) error {
	return runMigrations(db, driver, maxVersion, false)
}

func runMigrations(db *gorm.DB, driver string, maxVersion int, backup bool) error {
	if driver != "sqlite" && driver != "postgres" {
		return fmt.Errorf("不支持的数据库驱动: %s", driver)
	}

	files, err := listMigrationFiles(driver)
	if err != nil {
		return err
	}
	if maxVersion > 0 {
		filtered := make([]migrationFile, 0, len(files))
		for _, file := range files {
			if file.version <= maxVersion {
				filtered = append(filtered, file)
			}
		}
		files = filtered
	}

	pending, err := pendingMigrationFiles(db, files)
	if err != nil {
		return err
	}

	// 备份仅在有真正待应用迁移时创建：升级完成后的普通启动绝不触碰已有备份，
	// 避免"升级前快照"被后续启动覆盖。备份无法生成时宁可中止迁移等待人工处理。
	if backup && driver == "sqlite" && len(pending) > 0 {
		if err := backupSQLiteBeforeUpgrade(db); err != nil {
			if errors.Is(err, errBackupSkipped) {
				log.Printf("[数据升级] %v", err)
			} else {
				return fmt.Errorf("创建升级前备份失败，已中止迁移（可手动备份数据库后重试）: %w", err)
			}
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := ensureMigrationBaseTx(tx); err != nil {
			return err
		}
		for _, file := range files {
			var record migrationRecord
			findResult := tx.Where("version = ?", file.version).Limit(1).Find(&record)
			switch {
			case findResult.Error != nil:
				return fmt.Errorf("读取迁移版本 %d 失败: %w", file.version, findResult.Error)
			case findResult.RowsAffected == 1:
				if err := reconcileMigrationChecksum(tx, record, file); err != nil {
					return err
				}
				continue
			}

			contents, checksum, err := readMigrationFile(file)
			if err != nil {
				return err
			}
			if err := tx.Exec(contents).Error; err != nil {
				return fmt.Errorf("执行迁移版本 %d 失败: %w", file.version, err)
			}
			newRecord := migrationRecord{Version: file.version, Checksum: &checksum}
			if err := tx.Create(&newRecord).Error; err != nil {
				return fmt.Errorf("记录迁移版本 %d 失败: %w", file.version, err)
			}
		}
		return nil
	})
}

// ensureMigrationBaseTx 创建迁移记录表并补齐校验和列（历史库平滑升级）。
func ensureMigrationBaseTx(tx *gorm.DB) error {
	if !tx.Migrator().HasTable("schema_migrations") {
		if err := tx.Exec(`CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			checksum TEXT
		)`).Error; err != nil {
			return fmt.Errorf("创建迁移记录表失败: %w", err)
		}
		return nil
	}
	if tx.Migrator().HasColumn("schema_migrations", "checksum") {
		return nil
	}
	if err := tx.Exec(`ALTER TABLE schema_migrations ADD COLUMN checksum TEXT`).Error; err != nil {
		return fmt.Errorf("补齐迁移校验和列失败: %w", err)
	}
	return nil
}

// reconcileMigrationChecksum 校验已应用迁移文件是否被改动：
// 历史行没有校验和时以当前文件内容为基线补记；此后任何改动都会启动失败并被明确指出。
func reconcileMigrationChecksum(tx *gorm.DB, record migrationRecord, file migrationFile) error {
	_, checksum, err := readMigrationFile(file)
	if err != nil {
		return err
	}
	if record.Checksum == nil || *record.Checksum == "" {
		if err := tx.Model(&migrationRecord{}).Where("version = ?", file.version).
			Update("checksum", checksum).Error; err != nil {
			return fmt.Errorf("补记迁移版本 %d 校验和失败: %w", file.version, err)
		}
		return nil
	}
	if *record.Checksum != checksum {
		return fmt.Errorf(
			"迁移版本 %d 的文件内容与数据库记录不一致：数据库=%s 当前=%s。"+
				"已发布的迁移不允许修改，请恢复原内容或按 docs/database-upgrade.md 的流程新增迁移版本",
			file.version, shortHash(*record.Checksum), shortHash(checksum))
	}
	return nil
}

func readMigrationFile(file migrationFile) (string, string, error) {
	contents, err := fs.ReadFile(migrationFiles, file.path)
	if err != nil {
		return "", "", fmt.Errorf("读取迁移文件 %s 失败: %w", file.path, err)
	}
	sum := sha256.Sum256([]byte(contents))
	return string(contents), hex.EncodeToString(sum[:]), nil
}

func shortHash(hash string) string {
	const visible = 12
	if len(hash) <= visible {
		return hash
	}
	return hash[:visible] + "…"
}

// backupSQLiteBeforeUpgrade 在存在待应用迁移前生成 SQLite 一致性快照为 <库名>.pre-upgrade.bak。
// errBackupSkipped 表示当前场景无需备份（内存库等），调用方应静默继续。
var errBackupSkipped = errors.New("跳过升级前备份")

func backupSQLiteBeforeUpgrade(db *gorm.DB) error {
	type databaseListRow struct {
		Seq  int    `gorm:"column:seq"`
		Name string `gorm:"column:name"`
		File string `gorm:"column:file"`
	}
	var rows []databaseListRow
	if err := db.Raw("PRAGMA database_list").Scan(&rows).Error; err != nil {
		return fmt.Errorf("读取数据库文件路径: %w", err)
	}
	source := ""
	for _, row := range rows {
		if row.Name == "main" {
			source = strings.TrimSpace(row.File)
		}
	}
	if source == "" || source == ":memory:" || strings.Contains(source, "mode=memory") {
		return fmt.Errorf("%w：内存或未命名数据库", errBackupSkipped)
	}
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("确认数据库文件: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("%w：空数据库", errBackupSkipped)
	}
	dest := source + ".pre-upgrade.bak"
	if destInfo, statErr := os.Stat(dest); statErr == nil {
		if destInfo.IsDir() {
			return fmt.Errorf("创建备份文件: 目标路径是目录")
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("确认备份文件: %w", statErr)
	}
	temp, err := os.CreateTemp(filepath.Dir(dest), filepath.Base(dest)+".tmp-*")
	if err != nil {
		return fmt.Errorf("创建临时备份文件: %w", err)
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("关闭临时备份文件: %w", err)
	}
	defer os.Remove(tempPath)

	// VACUUM INTO 从 SQLite 一致性快照导出，能够包含仍在 WAL 中的最新提交。
	if err := db.Exec("VACUUM INTO ?", tempPath).Error; err != nil {
		return fmt.Errorf("生成 SQLite 一致性备份: %w", err)
	}
	if err := verifySQLiteBackup(tempPath); err != nil {
		return err
	}
	if err := os.Remove(dest); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("替换旧备份文件: %w", err)
	}
	if err := os.Rename(tempPath, dest); err != nil {
		return fmt.Errorf("提交备份文件: %w", err)
	}
	log.Printf("[数据升级] 已创建升级前备份: %s", dest)
	return nil
}

// verifySQLiteBackup 打开临时快照并执行完整性检查，避免把损坏文件提交为升级底线备份。
func verifySQLiteBackup(path string) error {
	backup, err := Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("打开临时 SQLite 备份: %w", err)
	}
	sqlDB, err := backup.DB()
	if err != nil {
		return fmt.Errorf("读取临时 SQLite 备份连接: %w", err)
	}
	defer sqlDB.Close()
	var result struct {
		Integrity string `gorm:"column:integrity_check"`
	}
	if err := backup.Raw("PRAGMA integrity_check").Scan(&result).Error; err != nil {
		return fmt.Errorf("检查 SQLite 备份完整性: %w", err)
	}
	if result.Integrity != "ok" {
		return fmt.Errorf("SQLite 备份完整性检查失败: %s", result.Integrity)
	}
	return nil
}

// pendingMigrationFiles 计算当前数据库尚未应用的迁移文件，用于判断是否存在真正的升级动作。
func pendingMigrationFiles(db *gorm.DB, files []migrationFile) ([]migrationFile, error) {
	if !db.Migrator().HasTable("schema_migrations") {
		return files, nil
	}
	type row struct {
		Version int `gorm:"column:version"`
	}
	var rows []row
	if err := db.Table("schema_migrations").Select("version").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("读取已应用迁移版本: %w", err)
	}
	applied := make(map[int]struct{}, len(rows))
	for _, r := range rows {
		applied[r.Version] = struct{}{}
	}
	pending := make([]migrationFile, 0, len(files))
	for _, file := range files {
		if _, ok := applied[file.version]; !ok {
			pending = append(pending, file)
		}
	}
	return pending, nil
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
