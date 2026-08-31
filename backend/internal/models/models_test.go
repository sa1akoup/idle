// 模型 JSON 回归测试：验证敌人模板的属性边界字段不会因错误 tag 丢失。
package models

import (
	"encoding/json"
	"testing"
)

func TestEnemyTemplateDefJSONRoundTripHPBounds(t *testing.T) {
	original := EnemyTemplateDef{ID: "template_test", HPFloor: 40, HPCap: 90}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("序列化敌人模板失败: %v", err)
	}

	var decoded EnemyTemplateDef
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("反序列化敌人模板失败: %v", err)
	}
	if decoded.HPFloor != original.HPFloor || decoded.HPCap != original.HPCap {
		t.Fatalf("HP 边界 round-trip 异常: got floor=%d cap=%d, want floor=%d cap=%d", decoded.HPFloor, decoded.HPCap, original.HPFloor, original.HPCap)
	}
}
