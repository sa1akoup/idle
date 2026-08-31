// 快照集成测试：migrate+seed 后构建快照，敌人变体应生成且节点/遭遇池引用被替换为变体 ID。
package service

import (
	"path/filepath"
	"testing"

	"idle/internal/config"
	"idle/internal/models"
	"idle/internal/repository/database"
)

func TestBuildSnapshotGeneratesEnemyVariants(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "idle-enemygen.db")
	db, err := database.Open("sqlite", dsn)
	if err != nil { t.Fatal(err) }
	if err := database.Migrate(db, "sqlite"); err != nil { t.Fatal(err) }
	if err := config.Seed(db); err != nil { t.Fatal(err) }

	snapshot, _, _, err := buildScenarioSnapshot(db, models.DefaultUserID, "city_ruins")
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { sqlDB, _ := db.DB(); _ = sqlDB.Close() })
	if len(snapshot.Enemies) == 0 { t.Fatal("快照没有生成敌人") }
	// 节点引用应指向变体 ID
	for _, node := range snapshot.Nodes {
		if node.EnemyID == "" { continue }
		if _, ok := snapshot.Enemies[node.EnemyID]; !ok {
			t.Fatalf("节点 %s 引用不存在的敌人变体 %s", node.ID, node.EnemyID)
		}
	}
	// 遭遇池引用应指向变体 ID
	for _, entries := range snapshot.Events.EncounterPools {
		for _, entry := range entries {
			if entry.EnemyID == "" { continue }
			if _, ok := snapshot.Enemies[entry.EnemyID]; !ok {
				t.Fatalf("遭遇池 %s 引用不存在的敌人变体 %s", entry.ID, entry.EnemyID)
			}
		}
	}
	// 每个变体装备合法：武器/护甲/弹药存在且口径匹配
	for id, enemy := range snapshot.Enemies {
		if _, ok := snapshot.Weapons[enemy.WeaponID]; !ok {
			t.Fatalf("敌人 %s 武器 %s 不存在", id, enemy.WeaponID)
		}
		if _, ok := snapshot.Armors[enemy.ArmorID]; !ok {
			t.Fatalf("敌人 %s 护甲 %s 不存在", id, enemy.ArmorID)
		}
		w := snapshot.Weapons[enemy.WeaponID]
		if w.AmmoPerRound > 0 {
			ammo, ok := snapshot.Ammos[enemy.AmmoID]
			if !ok { t.Fatalf("敌人 %s 弹药 %s 不存在", id, enemy.AmmoID) }
			if ammo.CaliberID != w.CaliberID { t.Fatalf("敌人 %s 弹药口径不匹配", id) }
		}
		if enemy.BackpackContainerID != "" {
			if _, ok := snapshot.Containers[enemy.BackpackContainerID]; !ok {
				t.Fatalf("敌人 %s 背包 %s 不存在", id, enemy.BackpackContainerID)
			}
		}
	}
}
