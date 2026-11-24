package main

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestSavedArmorRestoresOnlyMatchingEquipmentSlots(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	restoreSavedArmor(inv, []world.SavedItemStack{
		{Slot: 0, ItemID: inventory.ItemDiamondBoots, Count: 3, Damage: 4},
		{Slot: 1, ItemID: inventory.ItemDiamondHelmet, Count: 1}, // wrong position
		{Slot: 3, ItemID: inventory.ItemDiamondHelmet, Count: 1, Damage: 2},
		{Slot: 4, ItemID: inventory.ItemDiamondBoots, Count: 1}, // out of range
	})
	if got := inv.Armor[0]; got.ItemID != inventory.ItemDiamondBoots || got.Count != 1 || got.Damage != 4 {
		t.Fatalf("boots restore = %+v", got)
	}
	if got := inv.Armor[1]; !got.IsEmpty() {
		t.Fatalf("invalid armor slot was restored: %+v", got)
	}
	if got := inv.Armor[3]; got.ItemID != inventory.ItemDiamondHelmet || got.Count != 1 || got.Damage != 2 {
		t.Fatalf("helmet restore = %+v", got)
	}
	saved := savedArmorStacks(inv)
	if len(saved) != 2 || saved[0].Slot != 0 || saved[1].Slot != 3 {
		t.Fatalf("saved armor stacks = %+v", saved)
	}
}

func TestSavedOffhandRestoresFromLegacyPlayerSlot(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	restoreSavedInventory(inv, []world.SavedItemStack{
		{Slot: world.PlayerOffhandInventorySlot, ItemID: inventory.ItemTorch, Count: 80, Damage: 3},
		{Slot: world.PlayerOffhandInventorySlot, ItemID: inventory.ItemDiamond, Count: 1},
	})
	if got := inv.Offhand; got.ItemID != inventory.ItemTorch || got.Count != 64 || got.Damage != 3 {
		t.Fatalf("offhand restore = %+v, want first valid item clamped to 64", got)
	}
	inv = inventory.NewPlayerInventory()
	restoreSavedInventory(inv, []world.SavedItemStack{{Slot: world.PlayerOffhandInventorySlot, ItemID: inventory.ItemTorch, Count: 2}})
	if got := inv.Slots[0]; !got.IsEmpty() {
		t.Fatalf("offhand item was incorrectly restored to hotbar: %+v", got)
	}
}
