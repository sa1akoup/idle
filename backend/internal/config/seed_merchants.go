package config

import (
	"idle/internal/models"

	"gorm.io/gorm"
)

// seedMerchants 商人：武器/医疗/服装开放交易，黑市/联合/机械暂为占位。
func seedMerchants(db *gorm.DB) error {
	merchants := []models.MerchantDef{
		{ID: "weapon", Name: "武器商人", Category: "weapon", Reputation: 0, Desc: "买卖枪械、弹药与近战武器", Open: true, SortOrder: 1},
		{ID: "medical", Name: "医疗商人", Category: "medical", Reputation: 0, Desc: "买卖医疗道具、补给、食物与针剂", Open: true, SortOrder: 2},
		{ID: "clothing", Name: "服装商人", Category: "clothing", Reputation: 0, Desc: "买卖护甲、背包、头盔、耳机与胸挂", Open: true, SortOrder: 3},
		{ID: "mechanical", Name: "机械商人", Category: "mechanical", Reputation: 0, Desc: "买卖基础材料与工具", Open: true, SortOrder: 4},
		{ID: "black", Name: "黑市商人", Category: "black", Reputation: 0, Desc: "通用交易，无视类别（待开放）", Open: false, SortOrder: 5},
		{ID: "union", Name: "联合商人", Category: "union", Reputation: 0, Desc: "联合商会特供（待开放）", Open: false, SortOrder: 6},
	}
	for _, m := range merchants {
		if err := upsertSeedDef(db, &m, m.ID); err != nil {
			return err
		}
	}
	return nil
}
