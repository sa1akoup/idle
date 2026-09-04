// 地图 Seed：初始化可复用的 3x3 探索图、节点移动边和独立撤离点。
package config

import (
	"idle/internal/models"

	"gorm.io/gorm"
)

// seedMap 按地图全量替换 city_ruins 拓扑，防止旧节点和旧挂载残留。
func seedMap(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var oldNodeIDs []string
		if err := tx.Model(&models.NodeDef{}).Where("map_id = ?", "city_ruins").Pluck("id", &oldNodeIDs).Error; err != nil {
			return err
		}
		if len(oldNodeIDs) > 0 {
			if err := tx.Where("node_id IN ?", oldNodeIDs).Delete(&models.NodeContainerDef{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("map_id = ?", "city_ruins").Delete(&models.MapEdgeDef{}).Error; err != nil {
			return err
		}
		if err := tx.Where("map_id = ?", "city_ruins").Delete(&models.ExtractionPointDef{}).Error; err != nil {
			return err
		}
		if err := tx.Where("map_id = ?", "city_ruins").Delete(&models.NodeDef{}).Error; err != nil {
			return err
		}

		gameMap := models.MapDef{
			ID: "city_ruins", Name: "废弃城区", Desc: "雨夜城区，3×3 九宫格探索区，码头与加油站提供常规撤离。",
			StartNodeID: "city_ruins_node_5", LayoutColumns: 3, LayoutRows: 3,
			Tags: []string{"urban", "industrial", "grid"},
		}
		storedMap := models.MapDef{ID: gameMap.ID}
		if err := tx.FirstOrCreate(&storedMap, models.MapDef{ID: gameMap.ID}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.MapDef{}).Where("id = ?", gameMap.ID).Select("*").Updates(gameMap).Error; err != nil {
			return err
		}

		nodes := []models.NodeDef{
			{ID: "city_ruins_node_1", MapID: gameMap.ID, Name: "码头北区", PositionX: 0, PositionY: 0, ExploreTime: 3, Distance: "far", EnemyID: "template_patrol", EncounterRole: "patrol", ContainerSlots: 3, ValueTier: 2, EncounterChance: 50, Tags: []string{"outdoor", "waterfront", "extraction_anchor"}},
			{ID: "city_ruins_node_2", MapID: gameMap.ID, Name: "废弃仓库", PositionX: 1, PositionY: 0, ExploreTime: 4, Distance: "mid", EnemyID: "template_guard", EncounterRole: "guard", ContainerSlots: 4, ValueTier: 3, EncounterChance: 60, Tags: []string{"industrial", "indoor", "warehouse", "material"}},
			{ID: "city_ruins_node_3", MapID: gameMap.ID, Name: "旧市场", PositionX: 2, PositionY: 0, ExploreTime: 3, Distance: "mid", EnemyID: "template_patrol", EncounterRole: "patrol", ContainerSlots: 3, ValueTier: 2, EncounterChance: 50, Tags: []string{"outdoor", "market", "supply"}},
			{ID: "city_ruins_node_4", MapID: gameMap.ID, Name: "地下通道", PositionX: 0, PositionY: 1, ExploreTime: 4, Distance: "close", EnemyID: "template_patrol", EncounterRole: "patrol", ContainerSlots: 3, ValueTier: 2, EncounterChance: 55, Tags: []string{"underground", "enclosed", "medical"}},
			{ID: "city_ruins_node_5", MapID: gameMap.ID, Name: "居民楼", PositionX: 1, PositionY: 1, ExploreTime: 3, Distance: "close", EnemyID: "template_patrol", EncounterRole: "patrol", ContainerSlots: 3, ValueTier: 1, EncounterChance: 40, Tags: []string{"residential", "indoor", "food", "spawn"}},
			{ID: "city_ruins_node_6", MapID: gameMap.ID, Name: "集装箱场", PositionX: 2, PositionY: 1, ExploreTime: 4, Distance: "mid", EnemyID: "template_guard", EncounterRole: "guard", ContainerSlots: 5, ValueTier: 3, EncounterChance: 60, Tags: []string{"industrial", "outdoor", "cargo", "material"}},
			{ID: "city_ruins_node_7", MapID: gameMap.ID, Name: "社区诊所", PositionX: 0, PositionY: 2, ExploreTime: 3, Distance: "close", EnemyID: "template_patrol", EncounterRole: "patrol", ContainerSlots: 3, ValueTier: 3, EncounterChance: 55, Tags: []string{"indoor", "medical", "safehouse"}},
			{ID: "city_ruins_node_8", MapID: gameMap.ID, Name: "海关办公楼", PositionX: 1, PositionY: 2, ExploreTime: 5, Distance: "close", EnemyID: "template_elite", EncounterRole: "elite", ContainerSlots: 4, ValueTier: 5, EncounterChance: 80, Tags: []string{"indoor", "intel", "secured", "high_value"}},
			{ID: "city_ruins_node_9", MapID: gameMap.ID, Name: "加油站", PositionX: 2, PositionY: 2, ExploreTime: 3, Distance: "far", EnemyID: "template_guard", EncounterRole: "guard", ContainerSlots: 3, ValueTier: 4, EncounterChance: 70, Tags: []string{"outdoor", "fuel", "extraction_anchor"}},
		}
		for _, node := range nodes {
			if err := tx.Create(&node).Error; err != nil {
				return err
			}
		}

		edges := []models.MapEdgeDef{
			{MapID: gameMap.ID, FromNodeID: "city_ruins_node_1", ToNodeID: "city_ruins_node_2", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "city_ruins_node_1", ToNodeID: "city_ruins_node_4", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "city_ruins_node_2", ToNodeID: "city_ruins_node_3", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "city_ruins_node_2", ToNodeID: "city_ruins_node_5", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "city_ruins_node_3", ToNodeID: "city_ruins_node_6", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "city_ruins_node_4", ToNodeID: "city_ruins_node_5", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "city_ruins_node_4", ToNodeID: "city_ruins_node_7", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "city_ruins_node_5", ToNodeID: "city_ruins_node_6", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "city_ruins_node_5", ToNodeID: "city_ruins_node_8", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "city_ruins_node_6", ToNodeID: "city_ruins_node_9", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "city_ruins_node_7", ToNodeID: "city_ruins_node_8", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "city_ruins_node_8", ToNodeID: "city_ruins_node_9", MoveTime: 2, Bidirectional: true},
		}
		for _, edge := range edges {
			if err := tx.Create(&edge).Error; err != nil {
				return err
			}
		}

		extractionPoints := []models.ExtractionPointDef{
			{ID: "extract_pier", MapID: gameMap.ID, Name: "码头撤离点", Kind: "normal", AnchorNodeID: "city_ruins_node_1", TravelTime: 3, Enabled: true, IconKey: "pier", Tags: []string{"normal", "waterfront"}},
			{ID: "extract_gas_station", MapID: gameMap.ID, Name: "加油站撤离点", Kind: "normal", AnchorNodeID: "city_ruins_node_9", TravelTime: 2, Enabled: true, IconKey: "gas_station", Tags: []string{"normal", "fuel"}},
		}
		for _, point := range extractionPoints {
			if err := tx.Create(&point).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
