// 工作台制造测试：覆盖配方列表门槛、正常制造（聚合与耐久产物）、材料不足、
// 单作业互斥（含并发）、到期产出与入队容量预检。
package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"idle/internal/config"
	"idle/internal/models"
	"idle/internal/repository/database"

	"gorm.io/gorm"
)

func newCraftingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(os.TempDir(), fmt.Sprintf("idle-crafting-%d.db", time.Now().UnixNano()))
	db, err := database.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := database.Migrate(db, "sqlite"); err != nil {
		t.Fatalf("迁移测试数据库: %v", err)
	}
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入测试种子: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("读取测试连接: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
		_ = os.Remove(dsn)
	})
	return db
}

// clearSeededInventory 清空种子默认用户除现金外的全部库存与耐久实例，
// 使容量与材料断言只受测试自己写入的数据影响。
func clearSeededInventory(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Where("user_id = ? AND item_id <> ?", models.DefaultUserID, "cash").
		Delete(&models.Inventory{}).Error; err != nil {
		t.Fatalf("清理种子库存: %v", err)
	}
	if err := db.Where("user_id = ?", models.DefaultUserID).
		Delete(&models.ItemInstance{}).Error; err != nil {
		t.Fatalf("清理种子耐久实例: %v", err)
	}
}

func seedTestMaterials(t *testing.T, db *gorm.DB, items map[string]int) {
	t.Helper()
	for itemID, quantity := range items {
		if err := db.Create(&models.Inventory{
			UserID: models.DefaultUserID, ItemID: itemID, Name: itemID, Kind: "loot",
			Quantity: quantity, Price: 1, MerchantCategory: "mechanical",
		}).Error; err != nil {
			t.Fatalf("写入制造材料 %s: %v", itemID, err)
		}
	}
}

func setWorkbenchLevel(t *testing.T, db *gorm.DB, level int) {
	t.Helper()
	if err := db.Model(&models.HideoutFacility{}).
		Where("user_id = ? AND facility_id = ?", models.DefaultUserID, "workbench").
		Update("level", level).Error; err != nil {
		t.Fatalf("调整工作台等级: %v", err)
	}
}

func assertMaterialQuantity(t *testing.T, db *gorm.DB, itemID string, want int) {
	t.Helper()
	var quantity int
	if err := db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ?", models.DefaultUserID, itemID).
		Select("COALESCE(SUM(quantity), 0)").Scan(&quantity).Error; err != nil {
		t.Fatalf("读取库存 %s: %v", itemID, err)
	}
	if quantity != want {
		t.Fatalf("库存 %s 数量 = %d，期望 %d", itemID, quantity, want)
	}
}

func dueCraftJobs(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Model(&models.FacilityJob{}).
		Where("user_id = ? AND status = ?", models.DefaultUserID, facilityJobRunning).
		Update("complete_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("推进制造作业到期: %v", err)
	}
}

func countRunningCraftJobs(t *testing.T, db *gorm.DB) int {
	t.Helper()
	var count int64
	if err := db.Model(&models.FacilityJob{}).
		Where("user_id = ? AND job_type = ? AND status = ?", models.DefaultUserID, facilityJobTypeCraft, facilityJobRunning).
		Count(&count).Error; err != nil {
		t.Fatalf("统计制造作业: %v", err)
	}
	return int(count)
}

func TestListCraftingRecipesGatesByWorkbenchLevel(t *testing.T) {
	db := newCraftingTestDB(t)
	recipes, err := ListCraftingRecipesForUser(db, models.DefaultUserID)
	if err != nil {
		t.Fatalf("读取配方列表: %v", err)
	}
	if len(recipes) != 8 {
		t.Fatalf("配方数量 = %d，期望 8", len(recipes))
	}
	if recipes[0].ID != "craft_tool_set" || recipes[0].RequiredLevel != 1 {
		t.Fatalf("首条配方异常: %+v", recipes[0])
	}
	if recipes[0].WorkbenchLevel != 1 {
		t.Fatalf("工作台等级 = %d，期望 1", recipes[0].WorkbenchLevel)
	}
	// L1 材料不足时应锁定并给出原因，L2/L3 应因等级不足锁定。
	if recipes[0].CanStart {
		t.Fatalf("无材料时 L1 配方不应可制造")
	}
	if !strings.Contains(recipes[0].Reason, "材料不足") {
		t.Fatalf("L1 配方锁定原因异常: %s", recipes[0].Reason)
	}
	var l2Found, l3Found bool
	for _, recipe := range recipes {
		if recipe.RequiredLevel == 2 && recipe.Reason != "需要工作台 LV.2" {
			t.Fatalf("L2 配方 %s 锁定原因异常: %s", recipe.ID, recipe.Reason)
		}
		if recipe.RequiredLevel == 2 {
			l2Found = true
		}
		if recipe.RequiredLevel == 3 && recipe.Reason != "需要工作台 LV.3" {
			t.Fatalf("L3 配方 %s 锁定原因异常: %s", recipe.ID, recipe.Reason)
		}
		if recipe.RequiredLevel == 3 {
			l3Found = true
		}
	}
	if !l2Found || !l3Found {
		t.Fatalf("配方列表缺少 L2/L3 配方: %v", recipes)
	}
}

func TestCraftProducesAggregatedOutput(t *testing.T) {
	db := newCraftingTestDB(t)
	clearSeededInventory(t, db)
	seedTestMaterials(t, db, map[string]int{"metal_spare_parts": 2, "pack_of_screws": 2, "bundle_of_wires": 1})

	if err := StartCraftForUser(db, models.DefaultUserID, "craft_tool_set"); err != nil {
		t.Fatalf("开始制造工具组: %v", err)
	}
	for itemID, want := range map[string]int{"metal_spare_parts": 0, "pack_of_screws": 0, "bundle_of_wires": 0} {
		assertMaterialQuantity(t, db, itemID, want)
	}
	if countRunningCraftJobs(t, db) != 1 {
		t.Fatalf("制造作业未创建")
	}

	dueCraftJobs(t, db)
	if err := settleDueHideoutJobsForUser(db, models.DefaultUserID); err != nil {
		t.Fatalf("结算到期制造: %v", err)
	}
	assertMaterialQuantity(t, db, "set_of_tools", 1)
	if countRunningCraftJobs(t, db) != 0 {
		t.Fatalf("制造作业未完成")
	}
	var completed int64
	if err := db.Model(&models.FacilityJob{}).
		Where("user_id = ? AND job_type = ? AND status = ?", models.DefaultUserID, facilityJobTypeCraft, facilityJobCompleted).
		Count(&completed).Error; err != nil {
		t.Fatal(err)
	}
	if completed != 1 {
		t.Fatalf("已完成制造作业数 = %d，期望 1", completed)
	}
}

func TestCraftProducesDurabilityInstance(t *testing.T) {
	db := newCraftingTestDB(t)
	clearSeededInventory(t, db)
	setWorkbenchLevel(t, db, 3)
	seedTestMaterials(t, db, map[string]int{"set_of_tools": 1, "metal_spare_parts": 2})

	if err := StartCraftForUser(db, models.DefaultUserID, "craft_weapon_repair_kit"); err != nil {
		t.Fatalf("开始制造武器维修包: %v", err)
	}
	dueCraftJobs(t, db)
	if err := settleDueHideoutJobsForUser(db, models.DefaultUserID); err != nil {
		t.Fatalf("结算到期制造: %v", err)
	}
	var instance models.ItemInstance
	if err := db.Where("user_id = ? AND item_id = ? AND location_type = ? AND status = ?",
		models.DefaultUserID, "weapon_repair_kit_used", "inventory", "normal").First(&instance).Error; err != nil {
		t.Fatalf("读取制造产物实例: %v", err)
	}
	if instance.MaxDurability <= 0 || instance.CurrentDurability != instance.MaxDurability {
		t.Fatalf("耐久产物实例不完整: %+v", instance)
	}
}

func TestCraftRejectsInsufficientMaterials(t *testing.T) {
	db := newCraftingTestDB(t)
	clearSeededInventory(t, db)
	seedTestMaterials(t, db, map[string]int{"metal_spare_parts": 1, "pack_of_screws": 2, "bundle_of_wires": 1})

	err := StartCraftForUser(db, models.DefaultUserID, "craft_tool_set")
	if err == nil || !strings.Contains(err.Error(), "数量不足") {
		t.Fatalf("材料不足时错误 = %v，期望数量不足", err)
	}
	if countRunningCraftJobs(t, db) != 0 {
		t.Fatalf("材料不足时不应创建制造作业")
	}
	// 事务回滚后材料应原样保留。
	assertMaterialQuantity(t, db, "metal_spare_parts", 1)
}

func TestCraftRejectsUnmetWorkbenchLevel(t *testing.T) {
	db := newCraftingTestDB(t)
	clearSeededInventory(t, db)
	seedTestMaterials(t, db, map[string]int{"set_of_tools": 1, "metal_spare_parts": 2})

	err := StartCraftForUser(db, models.DefaultUserID, "craft_weapon_repair_kit")
	if err == nil || !strings.Contains(err.Error(), "LV.3") {
		t.Fatalf("等级不足时错误 = %v，期望 LV.3", err)
	}
	if countRunningCraftJobs(t, db) != 0 {
		t.Fatalf("等级不足时不应创建制造作业")
	}
	assertMaterialQuantity(t, db, "metal_spare_parts", 2)
}

func TestCraftEnforcesSingleActiveJob(t *testing.T) {
	db := newCraftingTestDB(t)
	clearSeededInventory(t, db)
	seedTestMaterials(t, db, map[string]int{"metal_spare_parts": 2, "pack_of_screws": 2, "bundle_of_wires": 1})

	if err := StartCraftForUser(db, models.DefaultUserID, "craft_tool_set"); err != nil {
		t.Fatalf("首次制造: %v", err)
	}
	err := StartCraftForUser(db, models.DefaultUserID, "craft_electric_drill")
	if err == nil || !strings.Contains(err.Error(), "已有作业") {
		t.Fatalf("第二次制造错误 = %v，期望已有作业", err)
	}
	if countRunningCraftJobs(t, db) != 1 {
		t.Fatalf("互斥失败：制造作业数 = %d，期望 1", countRunningCraftJobs(t, db))
	}
}

func TestCraftConcurrentStartOnlyOneSucceeds(t *testing.T) {
	db := newCraftingTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)
	clearSeededInventory(t, db)
	seedTestMaterials(t, db, map[string]int{"metal_spare_parts": 2, "pack_of_screws": 2, "bundle_of_wires": 1})

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- StartCraftForUser(db, models.DefaultUserID, "craft_tool_set")
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	var craftErrors []error
	for craftErr := range results {
		if craftErr == nil {
			successes++
		} else {
			craftErrors = append(craftErrors, craftErr)
		}
	}
	if successes != 1 {
		t.Fatalf("并发制造成功数 = %d，期望 1，错误=%v", successes, craftErrors)
	}
	for _, craftErr := range craftErrors {
		if craftErr == nil || !strings.Contains(craftErr.Error(), "已有作业") {
			t.Fatalf("并发制造失败原因异常: %v", craftErr)
		}
	}
	if countRunningCraftJobs(t, db) != 1 {
		t.Fatalf("并发制造后作业数 = %d，期望 1", countRunningCraftJobs(t, db))
	}
}

func TestCraftCapacityPrecheckRollsBackMaterials(t *testing.T) {
	db := newCraftingTestDB(t)
	clearSeededInventory(t, db)
	setWorkbenchLevel(t, db, 2)
	// 使用 L2 配方 craft_screw_split：金属件×1 + 建筑胶带×1 → 螺丝×3。
	// 清空种子后仓库容量 480（储物间 L1 无加成）。填满至 479：消耗材料后 477，产物 3 格 → 480 刚好放得下；
	// 填满至 480：消耗后 478，产物 3 格 → 481 超容，必须拒绝。
	seedTestMaterials(t, db, map[string]int{"metal_spare_parts": 1, "construction_tape": 1})
	if err := db.Create(&models.Inventory{
		UserID: models.DefaultUserID, ItemID: "filler_item", Name: "占位材料", Kind: "loot",
		Quantity: 478, Price: 1, MerchantCategory: "mechanical",
	}).Error; err != nil {
		t.Fatal(err)
	}

	err := StartCraftForUser(db, models.DefaultUserID, "craft_screw_split")
	if err == nil || !strings.Contains(err.Error(), "仓库空间不足") {
		t.Fatalf("容量不足时错误 = %v，期望仓库空间不足", err)
	}
	// 事务回滚：材料与占位材料都应原样保留。
	assertMaterialQuantity(t, db, "metal_spare_parts", 1)
	assertMaterialQuantity(t, db, "construction_tape", 1)
	assertMaterialQuantity(t, db, "filler_item", 478)
	if countRunningCraftJobs(t, db) != 0 {
		t.Fatalf("容量不足时不应创建制造作业")
	}

	// 释放 1 格后同配方可正常制造。
	if err := db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ?", models.DefaultUserID, "filler_item").
		Update("quantity", 477).Error; err != nil {
		t.Fatal(err)
	}
	if err := StartCraftForUser(db, models.DefaultUserID, "craft_screw_split"); err != nil {
		t.Fatalf("释放空间后制造失败: %v", err)
	}
	dueCraftJobs(t, db)
	if err := settleDueHideoutJobsForUser(db, models.DefaultUserID); err != nil {
		t.Fatalf("结算到期制造: %v", err)
	}
	assertMaterialQuantity(t, db, "pack_of_screws", 3)
}

func TestCraftCompletionDefersDeliveryWhenFull(t *testing.T) {
	db := newCraftingTestDB(t)
	clearSeededInventory(t, db)
	setWorkbenchLevel(t, db, 2)
	// 入队时仓库容量刚好容纳产物：材料 2 格 → 消耗后 477，产物 3 格 → 480 放得下；
	// 队列期间玩家填满仓库，到期时产物无处可放 → 转入待交付而非硬塞。
	seedTestMaterials(t, db, map[string]int{"metal_spare_parts": 1, "construction_tape": 1})
	if err := db.Create(&models.Inventory{
		UserID: models.DefaultUserID, ItemID: "filler_item", Name: "占位材料", Kind: "loot",
		Quantity: 477, Price: 1, MerchantCategory: "mechanical",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := StartCraftForUser(db, models.DefaultUserID, "craft_screw_split"); err != nil {
		t.Fatalf("入队制造: %v", err)
	}
	// 队列期间把仓库填满。
	if err := db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ?", models.DefaultUserID, "filler_item").
		Update("quantity", 480).Error; err != nil {
		t.Fatal(err)
	}
	dueCraftJobs(t, db)
	// 到期结算：产物无法放入满仓，作业转入待交付而非硬塞。
	if err := settleDueHideoutJobsForUser(db, models.DefaultUserID); err != nil {
		t.Fatalf("满仓结算不应失败: %v", err)
	}
	var unclaimed int64
	if err := db.Model(&models.FacilityJob{}).
		Where("user_id = ? AND job_type = ? AND status = ?", models.DefaultUserID, facilityJobTypeCraft, facilityJobCompletedUnclaimed).
		Count(&unclaimed).Error; err != nil {
		t.Fatal(err)
	}
	if unclaimed != 1 {
		t.Fatalf("满仓到期待交付作业数 = %d，期望 1", unclaimed)
	}
	var quantity int64
	if err := db.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ?", models.DefaultUserID, "pack_of_screws").
		Count(&quantity).Error; err != nil {
		t.Fatal(err)
	}
	if quantity != 0 {
		t.Fatalf("满仓时不应已交付产物")
	}

	// 释放空间后再次结算完成交付。
	if err := db.Where("user_id = ? AND item_id = ?", models.DefaultUserID, "filler_item").Delete(&models.Inventory{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := settleDueHideoutJobsForUser(db, models.DefaultUserID); err != nil {
		t.Fatalf("释放空间后的补交结算: %v", err)
	}
	assertMaterialQuantity(t, db, "pack_of_screws", 3)
	var remaining int64
	if err := db.Model(&models.FacilityJob{}).
		Where("user_id = ? AND job_type = ? AND status = ?", models.DefaultUserID, facilityJobTypeCraft, facilityJobCompletedUnclaimed).
		Count(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("交付后仍有待交付作业")
	}
}

func TestCraftRejectsUnknownRecipe(t *testing.T) {
	db := newCraftingTestDB(t)
	err := StartCraftForUser(db, models.DefaultUserID, "craft_missing")
	if err == nil || !strings.Contains(err.Error(), "配方不存在") {
		t.Fatalf("未知配方错误 = %v，期望配方不存在", err)
	}
}