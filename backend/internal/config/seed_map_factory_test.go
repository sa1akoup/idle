package config

import (
	"fmt"
	"testing"
	"time"

	"idle/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeedMapFactoryKeepsCityRuins(t *testing.T) {
	dsn := fmt.Sprintf("file:factory-map-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.MapDef{}, &models.NodeDef{}, &models.MapEdgeDef{}, &models.ExtractionPointDef{}, &models.NodeContainerDef{}); err != nil {
		t.Fatal(err)
	}
	if err := seedMap(db); err != nil {
		t.Fatalf("seed city: %v", err)
	}
	if err := seedMapFactory(db); err != nil {
		t.Fatalf("seed factory: %v", err)
	}
	if err := seedMapFactory(db); err != nil {
		t.Fatalf("reseed factory: %v", err)
	}
	var maps []models.MapDef
	if err := db.Order("id").Find(&maps).Error; err != nil {
		t.Fatal(err)
	}
	if len(maps) != 2 {
		t.Fatalf("maps = %d, want 2", len(maps))
	}
	var cityNodes, factoryNodes int64
	db.Model(&models.NodeDef{}).Where("map_id = ?", "city_ruins").Count(&cityNodes)
	db.Model(&models.NodeDef{}).Where("map_id = ?", factoryWoodsMapID).Count(&factoryNodes)
	if cityNodes != 9 {
		t.Fatalf("city nodes = %d, want 9", cityNodes)
	}
	if factoryNodes != 7 {
		t.Fatalf("factory nodes = %d, want 7", factoryNodes)
	}
	var office models.NodeDef
	if err := db.First(&office, "id = ?", "factory_woods_office").Error; err != nil {
		t.Fatal(err)
	}
	if office.EncounterRole != "boss" || office.EnemyID != "template_boss_killa" || office.EncounterChance >= 100 {
		t.Fatalf("办公室 Boss 配置错误: %+v", office)
	}
}
