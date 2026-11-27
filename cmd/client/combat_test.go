package main

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

func TestMaceSmashBonusThresholdAndFallResetInputs(t *testing.T) {
	for _, tc := range []struct {
		fall     float32
		grounded bool
		want     float32
	}{
		{1.5, false, 0},
		{1.49, false, 0},
		{1.6, false, 4},
		{3.9, false, 12},
		{4.9, false, 14},
		{9.9, false, 23},
		{8, true, 0},
	} {
		if got := maceSmashBonus(tc.fall, tc.grounded); got != tc.want {
			t.Errorf("maceSmashBonus(%.2f, %t) = %.2f, want %.2f", tc.fall, tc.grounded, got, tc.want)
		}
	}
}

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
		{"axe loses two durability", inventory.ItemNetheriteAxe, 12, 14, false, inventory.ItemNetheriteAxe},
		{"sword breaks on final point", inventory.ItemNetheriteSword, 2030, 0, true, inventory.ItemAir},
		{"axe breaks on last two points", inventory.ItemNetheriteAxe, 2029, 0, true, inventory.ItemAir},
		{"pickaxe breaks on last point", inventory.ItemNetheritePickaxe, 2030, 0, true, inventory.ItemAir},
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

func TestAttackDoesNotDamageNonWeaponDurableItems(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	inv.Slots[inv.SelectedHotbar] = inventory.ItemStack{ItemID: inventory.ItemShears, Count: 1, Damage: 20}
	if applyAttackItemDurability(inv, false) {
		t.Fatal("non-weapon attack reported a broken item")
	}
	if got := inv.Slots[inv.SelectedHotbar]; got.ItemID != inventory.ItemShears || got.Count != 1 || got.Damage != 20 {
		t.Fatalf("non-weapon held stack changed: %+v", got)
	}
}
