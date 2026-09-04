package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"idle/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type queryCounter struct {
	logger.Interface
	traces int
}

func (l *queryCounter) LogMode(level logger.LogLevel) logger.Interface {
	l.Interface = l.Interface.LogMode(level)
	return l
}

func (l *queryCounter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	l.traces++
	l.Interface.Trace(ctx, begin, fc, err)
}

func TestFindByIDsBatchesAcrossCatalogTablesAndCachesResults(t *testing.T) {
	counter := &queryCounter{Interface: logger.Default.LogMode(logger.Silent)}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: counter})
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := db.AutoMigrate(
		&models.WeaponDef{}, &models.ArmorDef{}, &models.AmmoDef{}, &models.ConsumableDef{},
		&models.ChestRigDef{}, &models.BackpackDef{}, &models.HelmetDef{}, &models.HeadsetDef{},
		&models.LootItemDef{}, &models.ItemUseDef{},
	); err != nil {
		t.Fatalf("迁移测试数据库: %v", err)
	}
	if err := db.Create(&models.WeaponDef{ID: "weapon-1", Name: "步枪", Price: 10, Slots: 2}).Error; err != nil {
		t.Fatalf("写入武器: %v", err)
	}
	if err := db.Create(&models.AmmoDef{ID: "ammo-1", Name: "弹药", CaliberID: "9x19", Level: 2, RoundsPerSlot: 30}).Error; err != nil {
		t.Fatalf("写入弹药: %v", err)
	}
	counter.traces = 0

	repo := New(db)
	items, err := repo.FindByIDs([]string{"weapon-1", "ammo-1", "weapon-1"})
	if err != nil {
		t.Fatalf("批量读取目录: %v", err)
	}
	if len(items) != 2 || items["weapon-1"].Kind != "weapon" || items["ammo-1"].Kind != "ammo" {
		t.Fatalf("批量目录结果异常: %+v", items)
	}
	if counter.traces != 11 {
		t.Fatalf("批量读取应访问 11 张目录表，实际 %d 次", counter.traces)
	}

	before := counter.traces
	if _, err := repo.FindByID("weapon-1"); err != nil {
		t.Fatalf("缓存读取目录: %v", err)
	}
	if counter.traces != before {
		t.Fatalf("缓存命中不应产生新 SQL，之前 %d，之后 %d", before, counter.traces)
	}
}

func TestFindByIDsReportsMissingItemsWithoutDiscardingFoundItems(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	if err := db.AutoMigrate(
		&models.WeaponDef{}, &models.ArmorDef{}, &models.AmmoDef{}, &models.ConsumableDef{},
		&models.ChestRigDef{}, &models.BackpackDef{}, &models.HelmetDef{}, &models.HeadsetDef{},
		&models.LootItemDef{},
	); err != nil {
		t.Fatalf("迁移测试数据库: %v", err)
	}
	if err := db.Create(&models.WeaponDef{ID: "weapon-1", Name: "步枪"}).Error; err != nil {
		t.Fatalf("写入武器: %v", err)
	}

	items, err := New(db).FindByIDs([]string{"weapon-1", "missing"})
	if !errors.Is(err, ErrItemNotFound) {
		t.Fatalf("缺失目录错误 = %v，期望 ErrItemNotFound", err)
	}
	if _, ok := items["weapon-1"]; !ok {
		t.Fatalf("缺失目录不应丢弃已找到的商品: %+v", items)
	}
}
