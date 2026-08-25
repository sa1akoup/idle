// 弹药快照测试：确保每种弹药固定商人价格与可购状态，并禁止 N5-N6 商店补给。
package engine

import "testing"

func TestValidateSnapshotRequiresImmutableAmmoSupply(t *testing.T) {
	snapshot := replayTestSnapshot()
	ammo := Ammo{
		ID: "ammo_556x45_n5", Name: "5.56×45mm N5级弹", CaliberID: "556x45", Level: 5,
		FleshDamageMultiplier: 0.95, ArmorDamageMultiplier: 1.2, Price: 16, RoundsPerSlot: 999,
	}
	snapshot.Ammos = map[string]Ammo{ammo.ID: ammo}
	if err := ValidateSnapshot(snapshot); err == nil {
		t.Fatal("缺少弹药补给快照时应校验失败")
	}
	snapshot.AmmoSupplies = map[string]AmmoSupply{
		ammo.ID: {AmmoID: ammo.ID, CaliberID: ammo.CaliberID, Level: ammo.Level, UnitPrice: 16, Available: true},
	}
	if err := ValidateSnapshot(snapshot); err == nil {
		t.Fatal("N5 弹药标记为商店可购时应校验失败")
	}
	supply := snapshot.AmmoSupplies[ammo.ID]
	supply.Available = false
	snapshot.AmmoSupplies[ammo.ID] = supply
	if err := ValidateSnapshot(snapshot); err != nil {
		t.Fatalf("不可购买的 N5 弹药快照应有效: %v", err)
	}
}
