// Session 引擎辅助函数：集中处理伤势等待、报告与掉落序列化。
package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"idle/internal/engine"
)

func injuryWaitSeconds(injury string) int64 {
	switch injury {
	case "light":
		return lightInjuryWaitSec
	case "heavy":
		return heavyInjuryWaitSec
	case "lethal":
		return lethalInjuryWaitSec
	default:
		return 0
	}
}

func appendEngineReport(report []string, line string) []string {
	return append(append([]string(nil), report...), line)
}

func classifyResourceUnavailable(err error, fallback string) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "现金"):
		return "cash_insufficient"
	case strings.Contains(message, "弹药") || strings.Contains(message, "口径"):
		return "ammo_unavailable"
	default:
		return fallback
	}
}

func encodeEngineReport(report []string) (string, error) {
	encoded, err := json.Marshal(report)
	if err != nil {
		return "", fmt.Errorf("序列化单局报告: %w", err)
	}
	return string(encoded), nil
}

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
		item, ok := snapshot.LootItems[drop.ItemID]
		if !ok {
			return "", fmt.Errorf("掉落物品 %s 不在场景快照中", drop.ItemID)
		}
		summaries = append(summaries, lootSummary{ID: drop.ID, ItemID: drop.ItemID, Name: item.Name, Category: item.Category, Quantity: drop.Quantity, ContainerID: drop.ContainerID, Source: drop.Source})
	}
	encoded, err := json.Marshal(summaries)
	if err != nil {
		return "", fmt.Errorf("序列化单局掉落: %w", err)
	}
	return string(encoded), nil
}
