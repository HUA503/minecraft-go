package main

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/render"
)

func TestCreativeHotbarNumberKeyCopiesCatalogItem(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	items := []inventory.CreativeItem{{ID: inventory.ItemStone}}
	if !applyCreativeHotbarKey(inv, render.GUISlotHit{Type: render.SlotCreativeItem, Index: 0}, items, 4) {
		t.Fatal("catalog item shortcut was not handled")
	}
	if got := inv.Slots[4]; got.ItemID != inventory.ItemStone || got.Count != inventory.GetMaxStackSize(inventory.ItemStone) {
		t.Fatalf("hotbar slot = %+v, want a maximum stone stack", got)
	}
}

func TestCreativeHotbarNumberKeySwapsInventorySlot(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	inv.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemStone, Count: 32}
	inv.Slots[12] = inventory.ItemStack{ItemID: inventory.ItemDiamond, Count: 7}
	if !applyCreativeHotbarKey(inv, render.GUISlotHit{Type: render.SlotInventory, Index: 12}, nil, 0) {
		t.Fatal("inventory swap shortcut was not handled")
	}
	if inv.Slots[0].ItemID != inventory.ItemDiamond || inv.Slots[0].Count != 7 || inv.Slots[12].ItemID != inventory.ItemStone || inv.Slots[12].Count != 32 {
		t.Fatalf("inventory/hotbar stacks were not swapped: hotbar=%+v inventory=%+v", inv.Slots[0], inv.Slots[12])
	}
}

func TestCreativeHotbarNumberKeyCopiesPresetStackData(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	var presets inventory.CreativeHotbarPresets
	presets[2][3] = inventory.ItemStack{ItemID: 267, Count: 1, Damage: 12}
	if !applyCreativeHotbarKey(inv, render.GUISlotHit{Type: render.SlotCreativeItem, Index: 21}, nil, 5, presets) {
		t.Fatal("preset item shortcut was not handled")
	}
	if got := inv.Slots[5]; got.ItemID != 267 || got.Count != 1 || got.Damage != 12 {
		t.Fatalf("hotbar item = %+v, want preset item with durability", got)
	}
}

func TestCreativeHotbarNumberKeyRejectsInvalidTargets(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	inv.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemStone, Count: 9}
	if applyCreativeHotbarKey(inv, render.GUISlotHit{Type: render.SlotInventory, Index: 0}, nil, 0) {
		t.Fatal("same-slot swap should be ignored")
	}
	if applyCreativeHotbarKey(inv, render.GUISlotHit{Type: render.SlotCreativeItem, Index: 2}, []inventory.CreativeItem{{ID: inventory.ItemStone}}, 1) {
		t.Fatal("out-of-range catalog index should be ignored")
	}
	if applyCreativeHotbarKey(inv, render.GUISlotHit{Type: render.SlotCreativeItem, Index: 0}, []inventory.CreativeItem{{ID: inventory.ItemStone}}, 9) {
		t.Fatal("out-of-range hotbar slot should be ignored")
	}
	if inv.Slots[0].Count != 9 || !inv.Slots[1].IsEmpty() {
		t.Fatalf("invalid shortcut changed inventory: %+v", inv.Slots[:2])
	}
}
