// 迁移文件守护测试：阻断会击穿旧库的危险语法、方言漂移和历史迁移篡改。
package database

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

// TestSQLiteMigrationsForbidNonLiteralAddColumnDefaults 静态扫描 sqlite 迁移：
// ALTER TABLE ADD COLUMN 不允许 DEFAULT CURRENT_* 或表达式默认值，
// 这类语法会在有存量行/历史形状的库上直接失败（参见 docs/database-upgrade.md）。
func TestSQLiteMigrationsForbidNonLiteralAddColumnDefaults(t *testing.T) {
	files, err := listMigrationFiles("sqlite")
	if err != nil {
		t.Fatal(err)
	}
	addColumn := regexp.MustCompile(`(?i)\bADD\s+COLUMN\b`)
	forbiddenDefault := regexp.MustCompile(`(?i)\bDEFAULT\s+(CURRENT_TIMESTAMP|CURRENT_DATE|CURRENT_TIME|[a-zA-Z_]+\s*\()`)
	for _, file := range files {
		raw, err := fs.ReadFile(migrationFiles, file.path)
		if err != nil {
			t.Fatal(err)
		}
		contents := string(raw)
		for _, statement := range strings.Split(contents, ";") {
			code := stripCommentLines(statement)
			if !addColumn.MatchString(code) {
				continue
			}
			if match := forbiddenDefault.FindString(code); match != "" {
				t.Errorf("%s 含非法的 ADD COLUMN 默认值（%q）。请改为字面量默认值并用 DML 回填存量行", file.path, strings.TrimSpace(match))
			}
		}
	}
}

// TestMigrationVersionsMatchAcrossDialects 保证两套方言文件的版本集合一致，防止一侧漏写迁移。
func TestMigrationVersionsMatchAcrossDialects(t *testing.T) {
	sqliteVersions, err := migrationVersionSet("sqlite")
	if err != nil {
		t.Fatal(err)
	}
	pgVersions, err := migrationVersionSet("postgres")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sqliteVersions, pgVersions) {
		t.Fatalf("两套方言迁移版本集合不一致：\nsqlite=%v\npostgres=%v", sqliteVersions, pgVersions)
	}
}

// TestMigrationChecksumDetectsTampering 校验和能发现已发布迁移被改动，且基线补记、恢复原内容后放行。
func TestMigrationChecksumDetectsTampering(t *testing.T) {
	db, _ := newGuardTestDB(t)
	if err := Migrate(db, "sqlite"); err != nil {
		t.Fatalf("首次全量迁移失败: %v", err)
	}
	var record migrationRecord
	if err := db.Where("version = ?", 1).First(&record).Error; err != nil {
		t.Fatalf("读取迁移记录: %v", err)
	}
	if record.Checksum == nil || *record.Checksum == "" {
		t.Fatalf("新应用的迁移应记录校验和")
	}

	// 历史行无校验和（旧库基线）应静默补记而非报错。
	if err := db.Model(&migrationRecord{}).Where("version = ?", 1).Update("checksum", nil).Error; err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db, "sqlite"); err != nil {
		t.Fatalf("基线补记不应失败: %v", err)
	}
	var restored migrationRecord
	if err := db.Where("version = ?", 1).First(&restored).Error; err != nil {
		t.Fatal(err)
	}
	if restored.Checksum == nil || *restored.Checksum == "" {
		t.Fatalf("基线补记后校验和仍为空")
	}

	// 模拟篡改：校验和不匹配时再次启动必须报错并指出版本号。
	if err := db.Model(&migrationRecord{}).Where("version = ?", 1).Update("checksum", "tampered").Error; err != nil {
		t.Fatal(err)
	}
	err := Migrate(db, "sqlite")
	if err == nil || !strings.Contains(err.Error(), "迁移版本 1") || !strings.Contains(err.Error(), "不一致") {
		t.Fatalf("篡改校验和后的报错不符合预期: %v", err)
	}
}

// TestMigrationsCreateRequiredSchema 确认全量迁移后 VerifySchema 通过，完整性清单与实际结构同步。
func TestMigrationsCreateRequiredSchema(t *testing.T) {
	db, _ := newGuardTestDB(t)
	if err := Migrate(db, "sqlite"); err != nil {
		t.Fatalf("全量迁移失败: %v", err)
	}
	if err := VerifySchema(db, "sqlite"); err != nil {
		t.Fatalf("完整迁移后关键结构校验失败（请核对 requiredSchema 清单）: %v", err)
	}
}

func newGuardTestDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	dsn := filepath.Join(os.TempDir(), fmt.Sprintf("idle-guard-%d.db", time.Now().UnixNano()))
	db, err := Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
		_ = os.Remove(dsn)
		_ = os.Remove(dsn + "-wal")
		_ = os.Remove(dsn + "-shm")
		_ = os.Remove(dsn + ".pre-upgrade.bak")
	})
	return db, dsn
}

func migrationVersionSet(driver string) ([]int, error) {
	files, err := listMigrationFiles(driver)
	if err != nil {
		return nil, err
	}
	versions := make([]int, 0, len(files))
	for _, file := range files {
		versions = append(versions, file.version)
	}
	return versions, nil
}

// stripCommentLines 移除语句中的独立注释行，避免解释性文字触发误报。
func stripCommentLines(statement string) string {
	kept := make([]string, 0, 8)
	for _, line := range strings.Split(statement, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// TestUpgradeBackupNotOverwrittenWithoutPending 验证：升级完成后（无待应用迁移）的普通启动
// 既不重新创建、也不截断已有的 .pre-upgrade.bak，真正的升级前快照得以保留。
func TestUpgradeBackupNotOverwrittenWithoutPending(t *testing.T) {
	db, dsn := newGuardTestDB(t)
	// 先建立有实际内容的旧版本库（该路径不触发备份），再做真正会升级的启动。
	if err := MigrateToVersion(db, "sqlite", 12); err != nil {
		t.Fatalf("构造历史版本库失败: %v", err)
	}
	if err := Migrate(db, "sqlite"); err != nil {
		t.Fatalf("升级迁移失败: %v", err)
	}
	before, err := os.ReadFile(dsn + ".pre-upgrade.bak")
	if err != nil {
		t.Fatalf("有待应用迁移时必须生成升级前备份: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := Migrate(db, "sqlite"); err != nil {
			t.Fatalf("无待迁移的第%d次启动不应失败: %v", i+1, err)
		}
		after, err := os.ReadFile(dsn + ".pre-upgrade.bak")
		if err != nil {
			t.Fatalf("普通启动不得删除升级前备份: %v", err)
		}
		if string(before) != string(after) {
			t.Fatalf("普通启动覆盖了升级前备份（第%d次）", i+1)
		}
	}
}

// TestUpgradeBackupPreservesWALData 验证一致性备份包含尚未合并到主库文件的 WAL 提交。
func TestUpgradeBackupPreservesWALData(t *testing.T) {
	db, dsn := newGuardTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("读取测试连接: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := MigrateToVersion(db, "sqlite", 12); err != nil {
		t.Fatalf("构造历史版本库失败: %v", err)
	}

	var journalMode struct {
		Mode string `gorm:"column:journal_mode"`
	}
	if err := db.Raw("PRAGMA journal_mode = WAL").Scan(&journalMode).Error; err != nil {
		t.Fatalf("启用 WAL 失败: %v", err)
	}
	if journalMode.Mode != "wal" {
		t.Fatalf("SQLite journal_mode = %s，期望 wal", journalMode.Mode)
	}
	if err := db.Exec("PRAGMA wal_autocheckpoint = 0").Error; err != nil {
		t.Fatalf("关闭 WAL 自动 checkpoint 失败: %v", err)
	}
	const username = "wal-snapshot-user"
	if err := db.Exec("INSERT INTO users (username, status, created_at, updated_at) VALUES (?, ?, ?, ?)", username, "active", time.Now(), time.Now()).Error; err != nil {
		t.Fatalf("写入 WAL 测试数据失败: %v", err)
	}
	if _, err := os.Stat(dsn + "-wal"); err != nil {
		t.Fatalf("WAL 文件未生成，测试未覆盖活动 WAL 场景: %v", err)
	}

	if err := Migrate(db, "sqlite"); err != nil {
		t.Fatalf("升级迁移失败: %v", err)
	}
	backup, err := Open("sqlite", dsn+".pre-upgrade.bak")
	if err != nil {
		t.Fatalf("打开升级前备份失败: %v", err)
	}
	backupSQL, err := backup.DB()
	if err != nil {
		t.Fatalf("读取备份连接失败: %v", err)
	}
	t.Cleanup(func() { _ = backupSQL.Close() })

	var count int64
	if err := backup.Table("users").Where("username = ?", username).Count(&count).Error; err != nil {
		t.Fatalf("读取备份中的 WAL 数据失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("备份中的 WAL 数据数量 = %d，期望 1", count)
	}
	var integrity struct {
		Result string `gorm:"column:integrity_check"`
	}
	if err := backup.Raw("PRAGMA integrity_check").Scan(&integrity).Error; err != nil {
		t.Fatalf("读取备份完整性检查失败: %v", err)
	}
	if integrity.Result != "ok" {
		t.Fatalf("备份完整性检查结果 = %s，期望 ok", integrity.Result)
	}
}

// TestBackupFailureAbortsPendingMigration 验证：存在待应用迁移但无法生成备份时，
// 迁移整体中止、一个版本都不落库；解除阻碍后可正常完成并补上文件备份。
func TestBackupFailureAbortsPendingMigration(t *testing.T) {
	db, dsn := newGuardTestDB(t)
	if err := MigrateToVersion(db, "sqlite", 10); err != nil {
		t.Fatalf("构造历史版本库失败: %v", err)
	}
	// 用同名目录占用备份路径，迫使备份创建必然失败。
	backupPath := dsn + ".pre-upgrade.bak"
	if err := os.Mkdir(backupPath, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(backupPath) })

	err := Migrate(db, "sqlite")
	if err == nil || !strings.Contains(err.Error(), "备份") {
		t.Fatalf("备份失败时应中止迁移并报错，实际: %v", err)
	}
	var applied int64
	db.Table("schema_migrations").Count(&applied)
	if applied != 10 {
		t.Fatalf("备份失败的迁移不得应用任何版本，实际已记录 %d 个版本", applied)
	}

	if err := os.Remove(backupPath); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db, "sqlite"); err != nil {
		t.Fatalf("解除备份阻碍后迁移应成功: %v", err)
	}
	info, statErr := os.Stat(backupPath)
	if statErr != nil || info.IsDir() {
		t.Fatalf("恢复后未生成文件形式的升级前备份")
	}
}
