package config

import (
	"fmt"
	"testing"
	"time"

	"idle/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeedLootAssignsRarityToEveryItem(t *testing.T) {
	dsn := fmt.Sprintf("file:seed-loot-rarity-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&models.LootItemDef{}); err != nil {
		t.Fatalf("迁移战利品表: %v", err)
	}
	if err := seedLoot(db); err != nil {
		t.Fatalf("写入战利品种子: %v", err)
	}
	var items []models.LootItemDef
	if err := db.Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) < 80 {
		t.Fatalf("战利品数量 = %d，目录过短", len(items))
	}
	legendary := map[string]bool{}
	for _, item := range items {
		if item.Rarity == "" || item.DropWeight <= 0 {
			t.Fatalf("物品 %s 未设置稀有度/权重", item.ID)
		}
		if item.Rarity == lootRarityLegendary {
			legendary[item.ID] = true
		}
	}
	for _, id := range []string{"ledx", "physical_bitcoin", "virtex"} {
		if !legendary[id] {
			t.Fatalf("%s 应为 legendary", id)
		}
	}
	if len(legendary) != 3 {
		t.Fatalf("legendary 数量 = %d，期望仅 LEDX/比特币/Virtex", len(legendary))
	}
}

func TestLootRarityMatchesTarkovTiers(t *testing.T) {
	items := []models.LootItemDef{
		{ID: "bolts", Category: "tool", Price: 35},
		{ID: "pack_of_screws", Category: "material", Price: 70},
		{ID: "salewa", Category: "medical", Price: 400},
		{ID: "printed_circuit_board", Category: "electronics", Price: 240},
		{ID: "graphics_card", Category: "electronics", Price: 2200},
		{ID: "ledx", Category: "medical", Price: 8500},
		{ID: "physical_bitcoin", Category: "valuable", Price: 10000},
		{ID: "key_customs_office", Category: "key", Price: 1800},
	}
	applyLootRarity(items)
	byID := map[string]models.LootItemDef{}
	for _, item := range items {
		byID[item.ID] = item
	}
	if byID["bolts"].Rarity != lootRarityCommon || byID["bolts"].DropWeight != 40 {
		t.Fatalf("螺栓应为 common/40，实际 %s/%d", byID["bolts"].Rarity, byID["bolts"].DropWeight)
	}
	if byID["salewa"].Rarity != lootRarityUncommon {
		t.Fatalf("Salewa 应为 uncommon，实际 %s", byID["salewa"].Rarity)
	}
	if byID["graphics_card"].Rarity != lootRaritySuperrare {
		t.Fatalf("显卡应为 superrare，实际 %s", byID["graphics_card"].Rarity)
	}
	if byID["ledx"].Rarity != lootRarityLegendary || byID["physical_bitcoin"].Rarity != lootRarityLegendary {
		t.Fatalf("LEDX/比特币应为 legendary，实际 %s/%s", byID["ledx"].Rarity, byID["physical_bitcoin"].Rarity)
	}
	if byID["ledx"].DropWeight >= byID["graphics_card"].DropWeight || byID["graphics_card"].DropWeight >= byID["salewa"].DropWeight {
		t.Fatalf("权重应为 螺栓>Salewa>显卡>LEDX，实际 bolts=%d salewa=%d gpu=%d ledx=%d",
			byID["bolts"].DropWeight, byID["salewa"].DropWeight, byID["graphics_card"].DropWeight, byID["ledx"].DropWeight)
	}
	if byID["key_customs_office"].Rarity != lootRarityUncommon {
		t.Fatalf("钥匙应能从夹克抽出（uncommon），实际 %s", byID["key_customs_office"].Rarity)
	}
}

func TestNodeSearchPoolsPlaceTarkovContainers(t *testing.T) {
	search := map[string]map[string]int{}
	for _, assignment := range nodeContainerAssignments() {
		if assignment.Pool != "" && assignment.Pool != "search" {
			continue
		}
		if search[assignment.NodeID] == nil {
			search[assignment.NodeID] = map[string]int{}
		}
		search[assignment.NodeID][assignment.ContainerID] = assignment.Weight
	}
	checks := []struct {
		node      string
		container string
	}{
		{"city_ruins_node_8", "safe"},
		{"city_ruins_node_7", "medcase"},
		{"city_ruins_node_5", "suitcase"},
		{"city_ruins_node_1", "ground_cache"},
		{"city_ruins_node_5", "jacket"},
		{"city_ruins_node_3", "cash_register"},
		{"city_ruins_node_9", "fuel_stash"},
		{"city_ruins_node_8", "computer_case"},
	}
	for _, check := range checks {
		if search[check.node][check.container] <= 0 {
			t.Fatalf("节点 %s 搜索池缺少容器 %s", check.node, check.container)
		}
	}
}
