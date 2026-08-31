// 升级演练测试：在“历史版本旧库 + 存量数据”上执行全量迁移与数据适配，
// 验证新版本上线时老玩家的存档可以自动、无损坏地完成升级。
package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"idle/internal/config"
	"idle/internal/models"
	"idle/internal/repository/database"
)

// TestUpgradeFromLegacyV10Database 复刻 survival 改版之前的旧库形状（migration v10）：
// 残留 waiting_injury 会话、无实例行的聚合医疗品、已下架物品的悬空引用，
// 全量迁移并执行 RunUserDataUpgrades 后应恢复到可正常游玩的状态。
func TestUpgradeFromLegacyV10Database(t *testing.T) {
	dsn := filepath.Join(os.TempDir(), fmt.Sprintf("idle-drill-%d.db", time.Now().UnixNano()))
	db, err := database.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("打开演练数据库: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
		_ = os.Remove(dsn)
		_ = os.Remove(dsn + ".pre-upgrade.bak")
	})

	if err := database.MigrateToVersion(db, "sqlite", 10); err != nil {
		t.Fatalf("构造历史版本库失败: %v", err)
	}

	const drillUserID uint = 2
	// 历史形状数据只能按当时的列结构用原始 SQL 写入（新模型的字段在 v10 还不存在）。
	startTime := time.Now().Add(-2 * time.Hour)
	if err := db.Exec(`INSERT INTO characters (user_id, name, created_at) VALUES (?, ?, ?)`,
		drillUserID, "老档幸存者", startTime).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO sessions (user_id, status, start_time, created_at) VALUES (?, ?, ?, ?)`,
		drillUserID, "waiting_injury", startTime, startTime).Error; err != nil {
		t.Fatal(err)
	}
	for _, item := range []models.Inventory{
		{UserID: drillUserID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 5213, Price: 1},
		{UserID: drillUserID, ItemID: "medkit", Name: "医疗包", Kind: "consumable", Quantity: 3, Price: 100},
		{UserID: drillUserID, ItemID: "bandage", Name: "绷带", Kind: "consumable", Quantity: 5, Price: 40},
		{UserID: drillUserID, ItemID: "old_rope", Name: "已下架绳索", Kind: "material", Quantity: 2, Price: 30},
	} {
		if err := db.Create(&item).Error; err != nil {
			t.Fatal(err)
		}
	}

	// ---- 全量迁移：模拟新版本上线自动升级（应创建 .pre-upgrade.bak 并清退遗留会话）----
	if err := database.Migrate(db, "sqlite"); err != nil {
		t.Fatalf("全量迁移失败: %v", err)
	}
	if _, statErr := os.Stat(dsn + ".pre-upgrade.bak"); statErr != nil {
		t.Fatalf("迁移未生成升级前备份: %v", statErr)
	}
	var legacyActive int64
	db.Table("sessions").Where("user_id = ? AND status NOT IN ?", drillUserID,
		[]string{"running", "success", "incapacitated", "failed"}).Count(&legacyActive)
	if legacyActive != 0 {
		t.Fatalf("迁移后仍存在非终态历史会话 %d 条", legacyActive)
	}
	var failedSession models.Session
	if err := db.Where("user_id = ?", drillUserID).First(&failedSession).Error; err != nil {
		t.Fatal(err)
	}
	if failedSession.Status != "failed" || failedSession.TerminalReason != "legacy_status" || failedSession.EndTime == nil {
		t.Fatalf("遗留会话未正确收尾: %+v", failedSession)
	}

	// 迁移后注入“新版本下架物品”场景：悬空实例行与悬空装备引用。
	if err := db.Create(&models.ItemInstance{
		UserID: drillUserID, ItemID: "gone_kit", CurrentDurability: 50, MaxDurability: 100,
		Status: "normal", LocationType: "inventory",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PlayerLoadout{
		UserID: drillUserID, CharacterID: 100,
		WeaponID: "ghost_weapon", ArmorID: "light_01",
		Consumables:       []string{"bandage", "ghost_food"},
		PresetConsumables: []string{"medkit", "gone_snack"},
		ConsumableRefs: []models.LoadoutItemRef{
			{ItemID: "bandage", Quantity: 1},
			{ItemID: "ghost_food", Quantity: 1},
		},
	}).Error; err != nil {
		t.Fatal(err)
	}

	// 目录种子与真实启动顺序一致：先 Seed 再跑存量数据适配。
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入目录种子: %v", err)
	}
	processed, err := RunUserDataUpgrades(db)
	if err != nil {
		t.Fatalf("存量数据适配失败: %v", err)
	}
	if processed < 2 {
		t.Fatalf("处理的用户数 = %d，期望至少覆盖新旧两代用户", processed)
	}

	// ---- 断言 1：聚合医疗品已换算为耐久实例且聚合行被移除 ----
	for _, tc := range []struct {
		itemID string
		want   int64
	}{
		{itemID: "medkit", want: 3},
		{itemID: "bandage", want: 5},
	} {
		var instances int64
		db.Model(&models.ItemInstance{}).
			Where("user_id = ? AND item_id = ? AND location_type = ?", drillUserID, tc.itemID, "inventory").
			Count(&instances)
		if instances != tc.want {
			t.Fatalf("用户%d 实例物品 %s 数量 = %d，期望 %d", drillUserID, tc.itemID, instances, tc.want)
		}
		var aggregates int64
		db.Model(&models.Inventory{}).
			Where("user_id = ? AND item_id = ?", drillUserID, tc.itemID).Count(&aggregates)
		if aggregates != 0 {
			t.Fatalf("用户%d 实例物品 %s 的聚合库存行未清理", drillUserID, tc.itemID)
		}
	}

	// ---- 断言 2：悬空引用与悬空行全部摘除，有效引用保留 ----
	var loadout models.PlayerLoadout
	if err := db.Where("user_id = ?", drillUserID).First(&loadout).Error; err != nil {
		t.Fatal(err)
	}
	if loadout.WeaponID != "" {
		t.Fatalf("悬空武器引用未被摘除: %q", loadout.WeaponID)
	}
	if loadout.ArmorID != "light_01" {
		t.Fatalf("有效护甲引用不应被动: %q", loadout.ArmorID)
	}
	assertSliceEquals(t, "consumables", loadout.Consumables, []string{"bandage"})
	assertSliceEquals(t, "preset_consumables", loadout.PresetConsumables, []string{"medkit"})
	assertRefsOnlyContain(t, loadout.ConsumableRefs, map[string]bool{"bandage": true})

	var danglingInventory int64
	db.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ?", drillUserID, "old_rope").Count(&danglingInventory)
	if danglingInventory != 0 {
		t.Fatalf("悬空库存行 old_rope 未清理")
	}
	// 现金是货币行、不在商品目录内，任何情况下都不得被当作悬空物品清理。
	assertInventoryQuantityForUser(t, db, drillUserID, "cash", 5213)
	var danglingInstance int64
	db.Model(&models.ItemInstance{}).Where("user_id = ? AND item_id = ?", drillUserID, "gone_kit").Count(&danglingInstance)
	if danglingInstance != 0 {
		t.Fatalf("悬空实例行 gone_kit 未清理")
	}

	// ---- 断言 3：完整性校验通过、重复执行幂等 ----
	if err := database.VerifySchema(db, "sqlite"); err != nil {
		t.Fatalf("升级后结构校验失败: %v", err)
	}
	var migration userDataUpgradeRecord
	if err := db.Where("version = ?", currentUserDataUpgradeVersion).First(&migration).Error; err != nil {
		t.Fatalf("未记录用户数据适配版本: %v", err)
	}
	if migration.ProcessedUsers != processed || migration.CreatedInstances != 8 {
		t.Fatalf("用户数据适配统计不正确: %+v，处理用户=%d", migration, processed)
	}
	if err := db.Create(&models.Inventory{
		UserID: drillUserID, ItemID: "appeared_after_upgrade", Name: "后续出现的悬空物品", Kind: "material", Quantity: 1,
	}).Error; err != nil {
		t.Fatalf("写入二次执行测试数据失败: %v", err)
	}
	secondPass, err := RunUserDataUpgrades(db)
	if err != nil {
		t.Fatalf("二次执行数据适配失败（应幂等）: %v", err)
	}
	if secondPass != processed {
		t.Fatalf("幂等执行用户数变化: %d -> %d", processed, secondPass)
	}
	var retainedAfterSecondPass int64
	db.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ?", drillUserID, "appeared_after_upgrade").Count(&retainedAfterSecondPass)
	if retainedAfterSecondPass != 1 {
		t.Fatalf("已完成版本的二次执行不应再次清理新库存行，实际 %d", retainedAfterSecondPass)
	}
	var medkitInstancesAfterSecondPass int64
	db.Model(&models.ItemInstance{}).
		Where("user_id = ? AND item_id = ? AND location_type = ?", drillUserID, "medkit", "inventory").
		Count(&medkitInstancesAfterSecondPass)
	if medkitInstancesAfterSecondPass != 3 {
		t.Fatalf("幂等执行后实例数应为 3，实际 %d（不应重复补发）", medkitInstancesAfterSecondPass)
	}
}

func assertSliceEquals(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v，期望 %v", label, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %v，期望 %v", label, got, want)
		}
	}
}

func assertRefsOnlyContain(t *testing.T, refs []models.LoadoutItemRef, allowed map[string]bool) {
	t.Helper()
	for _, ref := range refs {
		if !allowed[ref.ItemID] {
			t.Fatalf("补给引用残留悬空物品 %s", ref.ItemID)
		}
	}
}

// TestUserDataUpgradeRefusesWithoutCatalogSeed 复刻生产误用场景：
// 只跑了 migrate、目录种子从未执行（SEED_ON_START=false 且未手工 seed）时启动服务——
// 数据适配必须拒绝运行，且玩家的任何资产都不得被触碰。
func TestUserDataUpgradeRefusesWithoutCatalogSeed(t *testing.T) {
	dsn := filepath.Join(os.TempDir(), fmt.Sprintf("idle-unseeded-%d.db", time.Now().UnixNano()))
	db, err := database.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("打开演练数据库: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
		_ = os.Remove(dsn)
		_ = os.Remove(dsn + ".pre-upgrade.bak")
	})
	if err := database.MigrateToVersion(db, "sqlite", 18); err != nil {
		t.Fatalf("构造无种子库失败: %v", err)
	}

	const userID uint = 2
	startTime := time.Now().Add(-time.Hour)
	if err := db.Exec(`INSERT INTO characters (user_id, name, created_at) VALUES (?, ?, ?)`,
		userID, "未播种幸存者", startTime).Error; err != nil {
		t.Fatal(err)
	}
	for _, item := range []models.Inventory{
		{UserID: userID, ItemID: "cash", Name: "现金", Kind: "currency", Quantity: 777, Price: 1},
		{UserID: userID, ItemID: "medkit", Name: "医疗包", Kind: "consumable", Quantity: 2, Price: 100},
	} {
		if err := db.Create(&item).Error; err != nil {
			t.Fatal(err)
		}
	}

	_, err = RunUserDataUpgrades(db)
	if err == nil || !strings.Contains(err.Error(), "目录种子") {
		t.Fatalf("未种子化时应拒绝数据适配并提示，实际错误: %v", err)
	}

	// 拒绝路径不得对玩家资产产生任何写操作。
	var medkitQty, cashQty int64
	db.Model(&models.Inventory{}).
		Select("COALESCE(SUM(CASE WHEN item_id = 'medkit' THEN quantity ELSE 0 END), 0)").Scan(&medkitQty)
	db.Model(&models.Inventory{}).
		Select("COALESCE(SUM(CASE WHEN item_id = 'cash' THEN quantity ELSE 0 END), 0)").Scan(&cashQty)
	if medkitQty != 2 || cashQty != 777 {
		t.Fatalf("拒绝执行时资产被改动：medkit=%d cash=%d", medkitQty, cashQty)
	}
}
