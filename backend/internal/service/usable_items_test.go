package service

// 可用物资目录回归测试：事件用补给（烟雾弹/工具包）必须有会话可用效果定义，
// 否则会被 /api/consumables 过滤，导致角色页/部署页无法显示中文名称、预设挑选器无法添加。

import (
	"testing"

	"idle/internal/config"
)

func TestListUsableItemsIncludesEventConsumables(t *testing.T) {
	db := newSessionEventsTestDB(t, "usable-items")
	if err := config.Seed(db); err != nil {
		t.Fatalf("写入测试种子: %v", err)
	}
	list, err := ListUsableItems(db)
	if err != nil {
		t.Fatalf("读取可用物资目录: %v", err)
	}
	present := make(map[string]bool, len(list))
	for _, item := range list {
		present[item.ID] = true
	}
	for _, want := range []string{"smoke", "toolkit"} {
		if !present[want] {
			t.Fatalf("事件补给 %s 未出现在可用物资目录中", want)
		}
	}
}