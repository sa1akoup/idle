package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"idle/internal/models"
	"idle/internal/repository/database"
)

// TestRecoveryRatesMonotonicAcrossLevels 校验三设施恢复值逐级递增，避免升级收益倒退。
func TestRecoveryRatesMonotonicAcrossLevels(t *testing.T) {
	dsn := filepath.Join(os.TempDir(), fmt.Sprintf("idle-rec-%d.db", time.Now().UnixNano()))
	db, err := database.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db, "sqlite"); err != nil {
		t.Fatal(err)
	}
	if err := Seed(db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
		_ = os.Remove(dsn)
	})

	check := func(facility string, field string, get func(models.FacilityLevelDef) float64) {
		var levels []models.FacilityLevelDef
		if err := db.Where("facility_id = ?", facility).Order("level asc").Find(&levels).Error; err != nil {
			t.Fatal(err)
		}
		prev := -1.0
		for _, level := range levels {
			value := get(level)
			if value < prev {
				t.Fatalf("%s L%d %s = %.1f < L%d 的 %.1f，升级收益不应倒退", facility, level.Level, field, value, level.Level-1, prev)
			}
			if level.Level > 0 && value == prev {
				t.Fatalf("%s L%d %s 与上一级持平（%.1f），升级应带来提升", facility, level.Level, field, value)
			}
			prev = value
		}
	}
	check("heating", "EnergyRecoveryPerHour", func(l models.FacilityLevelDef) float64 { return l.EnergyRecoveryPerHour })
	check("water_collector", "HydrationRecoveryPerHour", func(l models.FacilityLevelDef) float64 { return l.HydrationRecoveryPerHour })
	check("rest_area", "StressRecoveryPerHour", func(l models.FacilityLevelDef) float64 { return l.StressRecoveryPerHour })
	check("medstation", "HPRecoveryPerHour", func(l models.FacilityLevelDef) float64 { return l.HPRecoveryPerHour })
}