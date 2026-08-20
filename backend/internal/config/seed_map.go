package config

import (
	"idle/internal/models"

	"gorm.io/gorm"
)

// seedMap 初始化地图和带标签的节点拓扑。
func seedMap(db *gorm.DB) error {
	if err := db.FirstOrCreate(&models.MapDef{ID: "city_ruins"}, models.MapDef{ID: "city_ruins"}).Error; err != nil {
		return err
	}
	if err := db.Model(&models.MapDef{}).Where("id=?", "city_ruins").Updates(models.MapDef{Name: "废弃城区", Desc: "雨夜城区，巡逻密集，1个撤离点", StartNodeID: "node_apt", ExtractionNodeID: "node_pier", Tags: []string{"urban", "industrial"}}).Error; err != nil {
		return err
	}

	nodes := []models.NodeDef{
		{ID: "node_apt", MapID: "city_ruins", Name: "居民楼", RouteOrder: 1, ExploreTime: 3, Distance: "close", EnemyID: "enemy_patrol", EncounterRole: "patrol", ContainerSlots: 3, ValueTier: 1, Connections: "node_warehouse", Tags: []string{"residential", "indoor", "food"}},
		{ID: "node_warehouse", MapID: "city_ruins", Name: "仓库区", RouteOrder: 2, ExploreTime: 4, Distance: "mid", EnemyID: "enemy_guard", EncounterRole: "guard", ContainerSlots: 4, ValueTier: 2, Connections: "node_customs", Tags: []string{"industrial", "indoor", "warehouse", "material"}},
		{ID: "node_customs", MapID: "city_ruins", Name: "海关办公楼", RouteOrder: 3, ExploreTime: 5, Distance: "close", EnemyID: "enemy_elite", EncounterRole: "elite", ContainerSlots: 4, ValueTier: 5, Connections: "node_tunnel", Tags: []string{"indoor", "intel", "secured", "high_value"}},
		{ID: "node_tunnel", MapID: "city_ruins", Name: "地下通道", RouteOrder: 4, ExploreTime: 4, Distance: "close", EnemyID: "enemy_patrol", EncounterRole: "patrol", ContainerSlots: 2, ValueTier: 1, Connections: "node_container", Tags: []string{"underground", "enclosed", "medical"}},
		{ID: "node_container", MapID: "city_ruins", Name: "集装箱场", RouteOrder: 5, ExploreTime: 4, Distance: "mid", EnemyID: "enemy_guard", EncounterRole: "guard", ContainerSlots: 5, ValueTier: 3, Connections: "node_pier", Tags: []string{"industrial", "outdoor", "cargo", "material"}},
		{ID: "node_pier", MapID: "city_ruins", Name: "码头撤离点", RouteOrder: 6, ExploreTime: 2, Distance: "far", EnemyID: "enemy_patrol", EncounterRole: "extraction", ContainerSlots: 2, ValueTier: 1, Connections: "", Tags: []string{"extraction", "outdoor", "waterfront"}},
	}
	for _, n := range nodes {
		if err := db.FirstOrCreate(&n, models.NodeDef{ID: n.ID}).Error; err != nil {
			return err
		}
		if err := db.Model(&models.NodeDef{}).Where("id=?", n.ID).Select("*").Updates(n).Error; err != nil {
			return err
		}
	}

	return nil
}
