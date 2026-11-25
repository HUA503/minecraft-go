package main

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestAttackDurabilityForWeaponsAndTools(t *testing.T) {
	tests := []struct {
		name       string
		itemID     int32
		before     int
		wantDamage int
		wantBroken bool
		wantItemID int32
	}{
		{"sword loses one durability", inventory.ItemNetheriteSword, 12, 13, false, inventory.ItemNetheriteSword},
		{"axe loses one durability", inventory.ItemNetheriteAxe, 12, 13, false, inventory.ItemNetheriteAxe},
		{"sword breaks on final point", inventory.ItemNetheriteSword, 2030, 0, true, inventory.ItemAir},
		{"tool breaks on final point", inventory.ItemNetheritePickaxe, 2030, 0, true, inventory.ItemAir},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inv := inventory.NewPlayerInventory()
			inv.Slots[inv.SelectedHotbar] = inventory.ItemStack{ItemID: tc.itemID, Count: 1, Damage: tc.before}
			if broken := applyAttackItemDurability(inv, false); broken != tc.wantBroken {
				t.Fatalf("broken = %t, want %t", broken, tc.wantBroken)
			}
			got := inv.Slots[inv.SelectedHotbar]
			if got.Damage != tc.wantDamage || got.ItemID != tc.wantItemID {
				t.Fatalf("held stack after hit = %+v, want item %d damage %d", got, tc.wantItemID, tc.wantDamage)
			}
		})
	}
}

func TestAttackDoesNotDamageItemsInCreativeMode(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	inv.Slots[inv.SelectedHotbar] = inventory.ItemStack{ItemID: inventory.ItemNetheriteSword, Count: 1, Damage: 100}
	if applyAttackItemDurability(inv, true) {
		t.Fatal("creative attack reported a broken item")
	}
	if got := inv.Slots[inv.SelectedHotbar].Damage; got != 100 {
		t.Fatalf("creative attack damage = %d, want unchanged 100", got)
	}
}

func TestAttackWithoutRegisteredToolDoesNotAddDurabilityDamage(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	inv.Slots[inv.SelectedHotbar] = inventory.ItemStack{ItemID: inventory.ItemApple, Count: 1}
	if applyAttackItemDurability(inv, false) {
		t.Fatal("non-tool attack reported a broken item")
	}
	if got := inv.Slots[inv.SelectedHotbar]; got.ItemID != inventory.ItemApple || got.Count != 1 || got.Damage != 0 {
		t.Fatalf("non-tool held stack changed: %+v", got)
	}
}
