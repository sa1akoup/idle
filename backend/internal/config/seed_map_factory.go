package config

import (
	"idle/internal/models"

	"gorm.io/gorm"
)

const factoryWoodsMapID = "factory_woods"

func seedMapFactory(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var oldNodeIDs []string
		if err := tx.Model(&models.NodeDef{}).Where("map_id = ?", factoryWoodsMapID).Pluck("id", &oldNodeIDs).Error; err != nil {
			return err
		}
		if len(oldNodeIDs) > 0 {
			if err := tx.Where("node_id IN ?", oldNodeIDs).Delete(&models.NodeContainerDef{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("map_id = ?", factoryWoodsMapID).Delete(&models.MapEdgeDef{}).Error; err != nil {
			return err
		}
		if err := tx.Where("map_id = ?", factoryWoodsMapID).Delete(&models.ExtractionPointDef{}).Error; err != nil {
			return err
		}
		if err := tx.Where("map_id = ?", factoryWoodsMapID).Delete(&models.NodeDef{}).Error; err != nil {
			return err
		}

		gameMap := models.MapDef{
			ID: factoryWoodsMapID, Name: "林缘厂房", Desc: "城区外的伐木场与厂房，室内有基拉出没，林道上可能撞上施图尔曼。",
			StartNodeID: "factory_woods_shop", LayoutColumns: 3, LayoutRows: 3,
			Tags: []string{"industrial", "woods", "factory"},
		}
		storedMap := models.MapDef{ID: gameMap.ID}
		if err := tx.FirstOrCreate(&storedMap, models.MapDef{ID: gameMap.ID}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.MapDef{}).Where("id = ?", gameMap.ID).Select("*").Updates(gameMap).Error; err != nil {
			return err
		}

		nodes := []models.NodeDef{
			{ID: "factory_woods_trail", MapID: gameMap.ID, Name: "林道", PositionX: 0, PositionY: 0, ExploreTime: 4, Distance: "far", EnemyID: "template_boss_shturman", EncounterRole: "boss", ContainerSlots: 3, ValueTier: 3, EncounterChance: 20, Tags: []string{"outdoor", "far", "woods"}},
			{ID: "factory_woods_lumber", MapID: gameMap.ID, Name: "伐木场", PositionX: 1, PositionY: 0, ExploreTime: 4, Distance: "mid", EnemyID: "template_guard", EncounterRole: "guard", ContainerSlots: 4, ValueTier: 3, EncounterChance: 60, Tags: []string{"outdoor", "industrial", "material"}},
			{ID: "factory_woods_fuel", MapID: gameMap.ID, Name: "油料棚", PositionX: 2, PositionY: 0, ExploreTime: 3, Distance: "mid", EnemyID: "template_guard", EncounterRole: "guard", ContainerSlots: 3, ValueTier: 4, EncounterChance: 65, Tags: []string{"outdoor", "fuel", "extraction_anchor"}},
			{ID: "factory_woods_dock", MapID: gameMap.ID, Name: "装卸坡", PositionX: 0, PositionY: 1, ExploreTime: 3, Distance: "mid", EnemyID: "template_patrol", EncounterRole: "patrol", ContainerSlots: 4, ValueTier: 2, EncounterChance: 50, Tags: []string{"outdoor", "cargo"}},
			{ID: "factory_woods_shop", MapID: gameMap.ID, Name: "厂房车间", PositionX: 1, PositionY: 1, ExploreTime: 4, Distance: "close", EnemyID: "template_guard", EncounterRole: "guard", ContainerSlots: 4, ValueTier: 3, EncounterChance: 50, Tags: []string{"indoor", "industrial", "spawn"}},
			{ID: "factory_woods_office", MapID: gameMap.ID, Name: "车间办公室", PositionX: 2, PositionY: 1, ExploreTime: 5, Distance: "close", EnemyID: "template_boss_killa", EncounterRole: "boss", ContainerSlots: 4, ValueTier: 5, EncounterChance: 25, Tags: []string{"indoor", "high_value", "secured"}},
			{ID: "factory_woods_extract", MapID: gameMap.ID, Name: "林间撤离点", PositionX: 0, PositionY: 2, ExploreTime: 3, Distance: "far", EnemyID: "template_guard", EncounterRole: "extraction", ContainerSlots: 2, ValueTier: 2, EncounterChance: 45, Tags: []string{"outdoor", "woods", "extraction_anchor"}},
		}
		for _, node := range nodes {
			if err := tx.Create(&node).Error; err != nil {
				return err
			}
		}

		edges := []models.MapEdgeDef{
			{MapID: gameMap.ID, FromNodeID: "factory_woods_trail", ToNodeID: "factory_woods_lumber", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "factory_woods_lumber", ToNodeID: "factory_woods_fuel", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "factory_woods_trail", ToNodeID: "factory_woods_dock", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "factory_woods_lumber", ToNodeID: "factory_woods_shop", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "factory_woods_fuel", ToNodeID: "factory_woods_office", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "factory_woods_dock", ToNodeID: "factory_woods_shop", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "factory_woods_dock", ToNodeID: "factory_woods_extract", MoveTime: 2, Bidirectional: true},
			{MapID: gameMap.ID, FromNodeID: "factory_woods_shop", ToNodeID: "factory_woods_office", MoveTime: 2, Bidirectional: true},
		}
		for _, edge := range edges {
			if err := tx.Create(&edge).Error; err != nil {
				return err
			}
		}

		extractionPoints := []models.ExtractionPointDef{
			{ID: "extract_factory_woods", MapID: gameMap.ID, Name: "林间撤离点", Kind: "normal", AnchorNodeID: "factory_woods_extract", TravelTime: 3, Enabled: true, IconKey: "woods", Tags: []string{"normal", "woods"}},
			{ID: "extract_factory_fuel", MapID: gameMap.ID, Name: "油料棚撤离点", Kind: "normal", AnchorNodeID: "factory_woods_fuel", TravelTime: 2, Enabled: true, IconKey: "fuel", Tags: []string{"normal", "fuel"}},
		}
		for _, point := range extractionPoints {
			if err := tx.Create(&point).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
