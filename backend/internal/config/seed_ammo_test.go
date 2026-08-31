// 弹药种子测试：确认 12 种口径生成 54 个唯一等级，并保留三种初始携弹库存。
package config

import (
	"fmt"
	"testing"
	"time"

	"idle/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeedAmmoCatalog(t *testing.T) {
	dsn := fmt.Sprintf("file:seed-ammo-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&models.AmmoDef{}, &models.Inventory{}, &models.EnemyTemplateDef{}); err != nil {
		t.Fatalf("迁移测试数据库: %v", err)
	}
	if err := seedAmmo(db); err != nil {
		t.Fatalf("写入弹药种子: %v", err)
	}

	var ammoCount int64
	if err := db.Model(&models.AmmoDef{}).Count(&ammoCount).Error; err != nil {
		t.Fatal(err)
	}
	if ammoCount != 54 {
		t.Fatalf("弹药定义数量 = %d，期望 54", ammoCount)
	}
	var caliberCount int64
	if err := db.Model(&models.AmmoDef{}).Distinct("caliber_id").Count(&caliberCount).Error; err != nil {
		t.Fatal(err)
	}
	if caliberCount != 12 {
		t.Fatalf("口径数量 = %d，期望 12", caliberCount)
	}
	for level, requirement := range map[int]int{1: 0, 2: 0, 3: 15, 4: 30} {
		var ammo models.AmmoDef
		if err := db.Where("caliber_id = ? AND level = ?", "556x45", level).First(&ammo).Error; err != nil {
			t.Fatalf("读取 N%d 弹药: %v", level, err)
		}
		if ammo.RoundsPerSlot != 999 || ammo.RepRequirement != requirement {
			t.Fatalf("N%d 弹药配置异常: %+v", level, ammo)
		}
	}
	for _, item := range initialAmmoInventory() {
		var inventory models.Inventory
		if err := db.Where("user_id = ? AND item_id = ?", models.DefaultUserID, item.ItemID).First(&inventory).Error; err != nil {
			t.Fatalf("读取初始弹药 %s: %v", item.ItemID, err)
		}
		if inventory.Quantity != item.Quantity || inventory.Kind != "ammo" {
			t.Fatalf("初始弹药 %s 异常: %+v", item.ItemID, inventory)
		}
	}
	if err := seedEnemyTemplates(db); err != nil {
		t.Fatalf("写入敌人模板: %v", err)
	}
	for templateID, level := range map[string]int{"template_elite": 4, "template_sniper": 4} {
		var template models.EnemyTemplateDef
		if err := db.First(&template, "id = ?", templateID).Error; err != nil {
			t.Fatalf("读取敌人模板 %s: %v", templateID, err)
		}
		if template.AmmoLevelMin > level || template.AmmoLevelMax < level {
			t.Fatalf("模板 %s 弹药等级区间 %d-%d 未覆盖 N%d", templateID, template.AmmoLevelMin, template.AmmoLevelMax, level)
		}
	}
}
