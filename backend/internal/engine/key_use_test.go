package engine

import "testing"

func TestRemainingItemUsesCountsDurability(t *testing.T) {
	defs := map[string]ItemUseDefinition{
		"key_customs_office": {ItemID: "key_customs_office", UseDurability: 1, MaxDurability: 5},
	}
	item := CarriedItem{ItemID: "key_customs_office", CurrentDurability: 5, MaxDurability: 5, Secure: true, InstanceID: 1}
	if got := remainingItemUses(item, defs); got != 5 {
		t.Fatalf("uses = %d, want 5", got)
	}
	item.CurrentDurability = 1
	if got := remainingItemUses(item, defs); got != 1 {
		t.Fatalf("uses = %d, want 1", got)
	}
	item.CurrentDurability = 0
	if got := remainingItemUses(item, defs); got != 0 {
		t.Fatalf("uses = %d, want 0", got)
	}
}
