package service

import (
	"strings"
	"testing"

	"idle/internal/models"
)

func TestInventoryPurposesIncludeActiveQuest(t *testing.T) {
	db := newCraftingTestDB(t)
	if err := AcceptQuestForUser(db, models.DefaultUserID, "medical_shortage"); err != nil {
		t.Fatalf("接取合同: %v", err)
	}
	instances, err := ListItemInstancesForUser(db, models.DefaultUserID)
	if err != nil {
		t.Fatalf("读取物品实例: %v", err)
	}
	foundShop := false
	for _, item := range instances {
		if item.ItemID != "salewa" {
			continue
		}
		foundShop = true
		if item.RaidExtract {
			continue
		}
		joined := strings.Join(item.Purposes, ",")
		if strings.Contains(joined, "合同：") || strings.Contains(joined, "医疗站") {
			t.Fatalf("商店/开局 Salewa 不应标注合同或升级用途: %v", item.Purposes)
		}
	}
	if !foundShop {
		t.Fatal("种子库存应有 Salewa 耐久实例")
	}
	if err := db.Model(&models.ItemInstance{}).
		Where("user_id = ? AND item_id = ?", models.DefaultUserID, "salewa").
		Update("raid_extract", true).Error; err != nil {
		t.Fatal(err)
	}
	instances, err = ListItemInstancesForUser(db, models.DefaultUserID)
	if err != nil {
		t.Fatalf("读取局内带出实例: %v", err)
	}
	foundFIR := false
	for _, item := range instances {
		if item.ItemID != "salewa" || !item.RaidExtract {
			continue
		}
		foundFIR = true
		joined := strings.Join(item.Purposes, ",")
		if !strings.Contains(joined, "合同：") {
			t.Fatalf("局内带出 Salewa 用途未包含合同: %v", item.Purposes)
		}
		if !strings.Contains(joined, "医疗站") {
			t.Fatalf("局内带出 Salewa 用途未包含藏身处升级: %v", item.Purposes)
		}
		if !strings.Contains(joined, "局内带出") {
			t.Fatalf("局内带出的 Salewa 应标注来源: %v", item.Purposes)
		}
	}
	if !foundFIR {
		t.Fatal("应有局内带出的 Salewa 实例")
	}
}

func TestInventoryPurposesIncludeStackedFIRLoot(t *testing.T) {
	db := newCraftingTestDB(t)
	if err := AcceptQuestForUser(db, models.DefaultUserID, "mechanical_screws"); err != nil {
		t.Fatalf("接取合同: %v", err)
	}
	seedTestMaterials(t, db, map[string]int{"pack_of_screws": 2})
	items, err := ListInventoryForUser(db, models.DefaultUserID)
	if err != nil {
		t.Fatalf("读取仓库: %v", err)
	}
	found := false
	for _, item := range items {
		if item.ItemID != "pack_of_screws" {
			continue
		}
		found = true
		joined := strings.Join(item.Purposes, ",")
		if !strings.Contains(joined, "合同：") {
			t.Fatalf("螺丝用途未包含合同: %v", item.Purposes)
		}
		if !item.RaidExtract || !strings.Contains(joined, "局内带出") {
			t.Fatalf("局内带出的螺丝应标注来源: raidExtract=%v purposes=%v", item.RaidExtract, item.Purposes)
		}
	}
	if !found {
		t.Fatal("应有局内带出的螺丝库存")
	}
}
