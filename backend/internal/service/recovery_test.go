// 恢复计划回归测试：覆盖即时恢复失败、藏身处持续恢复、无可用方式终态和旧数据修复。
package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"idle/internal/config"
	"idle/internal/engine"
	"idle/internal/models"
	"idle/internal/repository/database"

	"gorm.io/gorm"
)

func newRecoveryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(os.TempDir(), fmt.Sprintf("idle-recovery-%d.db", time.Now().UnixNano()))
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
		for _, suffix := range []string{"", "-wal", "-shm", ".pre-upgrade.bak"} {
			_ = os.Remove(dsn + suffix)
		}
	})
	return db
}

func recoveryTestPolicyJSON(t *testing.T) string {
	t.Helper()
	policy := RecoveryPolicy{
		HP:             RecoveryChoice{TargetPercent: 100, PrimaryMethod: recoveryMethodInventory, FallbackMethod: recoveryMethodHideout},
		Energy:         RecoveryChoice{TargetPercent: 80, PrimaryMethod: recoveryMethodInventory, FallbackMethod: recoveryMethodHideout},
		Hydration:      RecoveryChoice{TargetPercent: 80, PrimaryMethod: recoveryMethodInventory, FallbackMethod: recoveryMethodHideout},
		MerchantEnable: true,
	}
	encoded, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("序列化测试恢复策略: %v", err)
	}
	return string(encoded)
}

func TestDecodeRecoveryPolicyRejectsCorruptedJSON(t *testing.T) {
	if _, err := decodeRecoveryPolicy(`{"hp":`); err == nil {
		t.Fatal("损坏的恢复策略 JSON 应返回错误")
	}
}

func removeRecoveryItemsAndCash(t *testing.T, db *gorm.DB) {
	t.Helper()
	userID := models.DefaultUserID
	itemIDs := []string{"bandage", "medkit", "ai2_medkit", "ifak", "salewa"}
	if err := db.Where("user_id = ? AND item_id IN ?", userID, itemIDs).Delete(&models.Inventory{}).Error; err != nil {
		t.Fatalf("清理恢复品库存: %v", err)
	}
	if err := db.Where("user_id = ? AND item_id IN ?", userID, itemIDs).Delete(&models.ItemInstance{}).Error; err != nil {
		t.Fatalf("清理恢复品实例: %v", err)
	}
	if err := db.Model(&models.Inventory{}).Where("user_id = ? AND item_id = ?", userID, "cash").Update("quantity", 0).Error; err != nil {
		t.Fatalf("清空测试现金: %v", err)
	}
}

func findRecoveryTask(t *testing.T, db *gorm.DB, planID uint, resource string) models.RecoveryTask {
	t.Helper()
	var task models.RecoveryTask
	if err := db.Where("recovery_plan_id = ? AND resource_type = ?", planID, resource).First(&task).Error; err != nil {
		t.Fatalf("读取 %s 恢复任务: %v", resource, err)
	}
	return task
}

func createRecoveryTestPlan(t *testing.T, db *gorm.DB, state engine.CharacterState) models.RecoveryPlan {
	t.Helper()
	const sessionID uint = 9001
	if err := db.Transaction(func(tx *gorm.DB) error {
		return createRecoveryPlanTx(tx, models.DefaultUserID, sessionID, state, recoveryTestPolicyJSON(t))
	}); err != nil {
		t.Fatalf("创建测试恢复计划: %v", err)
	}
	var plan models.RecoveryPlan
	if err := db.Where("user_id = ? AND source_session_id = ?", models.DefaultUserID, sessionID).First(&plan).Error; err != nil {
		t.Fatalf("读取测试恢复计划: %v", err)
	}
	return plan
}

func TestCreateRecoveryPlanKeepsHideoutAfterFailedMerchantRecovery(t *testing.T) {
	db := newRecoveryTestDB(t)
	removeRecoveryItemsAndCash(t, db)

	plan := createRecoveryTestPlan(t, db, engine.CharacterState{
		Strength:  55,
		HP:        50,
		Energy:    100,
		Hydration: 100,
	})
	if plan.Status != "running" {
		t.Fatalf("恢复计划状态 = %s，期望 running", plan.Status)
	}

	task := findRecoveryTask(t, db, plan.ID, "hp")
	if task.ActualMethod != recoveryMethodHideout {
		t.Fatalf("HP 实际恢复方式 = %s，期望 hideout", task.ActualMethod)
	}
	if task.Status != "running" || task.RatePerHour <= 0 || task.CompleteAt == nil {
		t.Fatalf("藏身处恢复任务异常: %+v", task)
	}
}

func TestRecoveryPlanFailsWhenNoRecoveryMethodAvailable(t *testing.T) {
	db := newRecoveryTestDB(t)
	removeRecoveryItemsAndCash(t, db)
	if err := db.Model(&models.HideoutFacility{}).
		Where("user_id = ? AND facility_id = ?", models.DefaultUserID, "medstation").
		Update("state", "upgrading").Error; err != nil {
		t.Fatalf("禁用医疗站: %v", err)
	}

	plan := createRecoveryTestPlan(t, db, engine.CharacterState{
		Strength:  55,
		HP:        50,
		Energy:    100,
		Hydration: 100,
	})
	if plan.Status != "failed" {
		t.Fatalf("恢复计划状态 = %s，期望 failed", plan.Status)
	}
	task := findRecoveryTask(t, db, plan.ID, "hp")
	if task.Status != "failed" || task.ActualMethod != recoveryMethodNone {
		t.Fatalf("无可用恢复方式时任务异常: %+v", task)
	}
	if err := ensureRecoveryReadyForStartTx(db, models.DefaultUserID); err != nil {
		t.Fatalf("失败恢复计划不应阻止新行动: %v", err)
	}
}

func TestSettleRecoveryRepairsLegacyImmediateMethodTask(t *testing.T) {
	db := newRecoveryTestDB(t)
	removeRecoveryItemsAndCash(t, db)
	startedAt := time.Now().Add(-time.Hour)
	plan := models.RecoveryPlan{
		UserID: models.DefaultUserID, SourceSessionID: 9002, Status: "running", StartedAt: startedAt,
	}
	if err := db.Create(&plan).Error; err != nil {
		t.Fatalf("创建旧恢复计划: %v", err)
	}
	if err := db.Create(&models.RecoveryTask{
		RecoveryPlanID: plan.ID, UserID: models.DefaultUserID, ResourceType: "hp",
		StartValue: 50, CurrentValue: 50, TargetValue: 101, PrimaryMethod: recoveryMethodInventory,
		ActualMethod: recoveryMethodMerchant, StartedAt: startedAt, Status: "running",
	}).Error; err != nil {
		t.Fatalf("创建旧恢复任务: %v", err)
	}
	if err := db.Model(&models.Character{}).Where("user_id = ?", models.DefaultUserID).Updates(map[string]interface{}{
		"hp": 50, "energy": 100, "hydration": 100,
	}).Error; err != nil {
		t.Fatalf("设置测试角色资源: %v", err)
	}

	if err := SettleRecoveryForUser(db, models.DefaultUserID); err != nil {
		t.Fatalf("结算旧恢复计划: %v", err)
	}
	var task models.RecoveryTask
	if err := db.Where("id = ?", plan.ID).First(&plan).Error; err != nil {
		t.Fatalf("读取旧恢复计划结果: %v", err)
	}
	if err := db.Where("recovery_plan_id = ?", plan.ID).First(&task).Error; err != nil {
		t.Fatalf("读取旧恢复任务结果: %v", err)
	}
	if plan.Status != "running" || task.ActualMethod != recoveryMethodHideout || task.RatePerHour <= 0 {
		t.Fatalf("旧卡死任务未切换为藏身处恢复: plan=%+v task=%+v", plan, task)
	}
}
