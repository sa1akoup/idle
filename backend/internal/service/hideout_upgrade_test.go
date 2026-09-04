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

	"gorm.io/gorm"
)

func newHideoutUpgradeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(os.TempDir(), fmt.Sprintf("idle-hideout-upgrade-%d.db", time.Now().UnixNano()))
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

func TestHideoutItemRequirementsExistInLootCatalog(t *testing.T) {
	db := newHideoutUpgradeTestDB(t)
	if err := ValidateEventConfig(db); err != nil {
		t.Fatalf("事件配置校验失败: %v", err)
	}
	var requirements []models.FacilityRequirement
	if err := db.Where("requirement_type = ?", "item").Find(&requirements).Error; err != nil {
		t.Fatalf("读取升级材料: %v", err)
	}
	var levels []models.FacilityLevelDef
	if err := db.Find(&levels).Error; err != nil {
		t.Fatalf("读取设施等级: %v", err)
	}
	itemIDs := make(map[string]struct{})
	for _, requirement := range requirements {
		if requirement.ReferenceID != "" {
			itemIDs[requirement.ReferenceID] = struct{}{}
		}
	}
	for _, level := range levels {
		if level.MaterialID != "" {
			itemIDs[level.MaterialID] = struct{}{}
		}
	}
	for itemID := range itemIDs {
		var loot models.LootItemDef
		if err := db.Where("id = ?", itemID).First(&loot).Error; err != nil {
			t.Fatalf("升级材料 %s 不在战利品目录中: %v", itemID, err)
		}
	}
}

func TestHideoutUpgradeRequiresRaidExtractMaterials(t *testing.T) {
	db := newHideoutUpgradeTestDB(t)
	shopItems := map[string]int{
		"construction_tape": 2,
		"bolts":             2,
		"screw_nuts":        2,
		"metal_spare_parts": 1,
	}
	for itemID, quantity := range shopItems {
		if err := db.Create(&models.Inventory{
			UserID: models.DefaultUserID, ItemID: itemID, Name: itemID, Kind: "loot",
			Quantity: quantity, Price: 1, RaidExtract: false, MerchantCategory: "mechanical",
		}).Error; err != nil && !strings.Contains(err.Error(), "UNIQUE") {
			t.Fatalf("写入商店材料 %s: %v", itemID, err)
		}
	}
	err := StartFacilityUpgradeForUser(db, models.DefaultUserID, "security")
	if err == nil {
		t.Fatal("商店材料不应能启动安保升级")
	}
	if !strings.Contains(err.Error(), "升级条件未满足") {
		t.Fatalf("商店材料错误 = %v，期望升级条件未满足", err)
	}

	for itemID, quantity := range shopItems {
		if err := db.Create(&models.Inventory{
			UserID: models.DefaultUserID, ItemID: itemID, Name: itemID, Kind: "loot",
			Quantity: quantity, Price: 1, RaidExtract: true, MerchantCategory: "mechanical",
		}).Error; err != nil {
			t.Fatalf("写入局内带出材料 %s: %v", itemID, err)
		}
	}
	if err := StartFacilityUpgradeForUser(db, models.DefaultUserID, "security"); err != nil {
		t.Fatalf("局内带出材料应能启动安保升级: %v", err)
	}
	var state models.HideoutFacility
	if err := db.Where("user_id = ? AND facility_id = ?", models.DefaultUserID, "security").First(&state).Error; err != nil {
		t.Fatalf("读取安保状态: %v", err)
	}
	if state.State != "upgrading" {
		t.Fatalf("安保状态 = %s，期望 upgrading", state.State)
	}
}
