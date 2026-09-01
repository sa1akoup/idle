// Session 引擎辅助函数：集中处理伤势等待、报告与掉落序列化。
package service

import (
	"encoding/json"
	"fmt"

	"idle/internal/engine"
)

// encodeEngineReport 将单局报告文本列表序列化为 JSON 字符串。
func encodeEngineReport(report []string) (string, error) {
	encoded, err := json.Marshal(report)
	if err != nil {
		return "", fmt.Errorf("序列化单局报告: %w", err)
	}
	return string(encoded), nil
}

// resolveLootSummary 从场景快照统一解析普通掉落和弹药掉落的名称与稳定类目。
func resolveLootSummary(snapshot engine.ScenarioSnapshot, drop engine.LootDrop) (string, string, error) {
	if item, ok := snapshot.LootItems[drop.ItemID]; ok {
		return item.Name, item.Category, nil
	}
	if ammo, ok := snapshot.Ammos[drop.ItemID]; ok {
		return ammo.Name, "ammo", nil
	}
	return "", "", fmt.Errorf("掉落物品 %s 不在场景快照目录中", drop.ItemID)
}

// encodeEngineLoot 将单局战利品序列化为含物品名与类目的摘要 JSON，物品定义必须存在于场景快照。
func encodeEngineLoot(loot []engine.LootDrop, snapshot engine.ScenarioSnapshot) (string, error) {
	type lootSummary struct {
		ID          string `json:"id"`
		ItemID      string `json:"itemId"`
		Name        string `json:"name"`
		Category    string `json:"category"`
		Quantity    int    `json:"quantity"`
		ContainerID string `json:"containerId"`
		Source      string `json:"source"`
	}
	summaries := make([]lootSummary, 0, len(loot))
	for _, drop := range loot {
		name, category, err := resolveLootSummary(snapshot, drop)
		if err != nil {
			return "", err
		}
		summaries = append(summaries, lootSummary{ID: drop.ID, ItemID: drop.ItemID, Name: name, Category: category, Quantity: drop.Quantity, ContainerID: drop.ContainerID, Source: drop.Source})
	}
	encoded, err := json.Marshal(summaries)
	if err != nil {
		return "", fmt.Errorf("序列化单局掉落: %w", err)
	}
	return string(encoded), nil
}
