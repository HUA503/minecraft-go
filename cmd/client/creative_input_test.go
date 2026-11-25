package main

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/render"
)

func TestNewClientDefaultsCreativeInventoryToSearch(t *testing.T) {
	client := NewClient("", "test", false, 320, 240)
	if client.creativeTab != inventory.CreativeAll {
		t.Fatalf("initial creative tab = %d, want search (%d)", client.creativeTab, inventory.CreativeAll)
	}
}

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

func TestCreativeCursorDragLeftDistributesEvenly(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	cursor := inventory.ItemStack{ItemID: inventory.ItemStone, Count: 12}
	placed := distributeCreativeCursor(inv, &cursor, []int{0, 1, 2}, 1)
	if placed != 12 || !cursor.IsEmpty() {
		t.Fatalf("placed=%d cursor=%+v, want all 12 placed", placed, cursor)
	}
	for i := 0; i < 3; i++ {
		if got := inv.Slots[i]; got.ItemID != inventory.ItemStone || got.Count != 4 {
			t.Fatalf("slot %d = %+v, want 4 stone", i, got)
		}
	}
}

func TestCreativeCursorDragRightPlacesOnePerSlot(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	cursor := inventory.ItemStack{ItemID: inventory.ItemStone, Count: 3}
	placed := distributeCreativeCursor(inv, &cursor, []int{0, 1, 2, 3}, 2)
	if placed != 3 || cursor.Count != 0 || !cursor.IsEmpty() {
		t.Fatalf("placed=%d cursor=%+v, want 3 placed and empty cursor", placed, cursor)
	}
	for i := 0; i < 3; i++ {
		if got := inv.Slots[i]; got.ItemID != inventory.ItemStone || got.Count != 1 {
			t.Fatalf("slot %d = %+v, want 1 stone", i, got)
		}
	}
	if !inv.Slots[3].IsEmpty() {
		t.Fatalf("fourth target should stay empty: %+v", inv.Slots[3])
	}
}

func TestCreativeCursorDragMergesCompatibleStacksAndCapsSlots(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	inv.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemStone, Count: 63}
	cursor := inventory.ItemStack{ItemID: inventory.ItemStone, Count: 5}
	placed := distributeCreativeCursor(inv, &cursor, []int{0, 1}, 2)
	if placed != 2 || cursor.Count != 3 || inv.Slots[0].Count != 64 || inv.Slots[1].Count != 1 {
		t.Fatalf("placed=%d cursor=%+v slots=%+v", placed, cursor, inv.Slots[:2])
	}
}

func TestCreativeCursorDragSkipsIncompatibleAndInvalidTargets(t *testing.T) {
	inv := inventory.NewPlayerInventory()
	inv.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemDiamond, Count: 4}
	cursor := inventory.ItemStack{ItemID: inventory.ItemStone, Count: 2}
	placed := distributeCreativeCursor(inv, &cursor, []int{0, -1, 1, 1, len(inv.Slots)}, 2)
	if placed != 1 || cursor.Count != 1 || inv.Slots[0].Count != 4 || inv.Slots[1].Count != 1 {
		t.Fatalf("placed=%d cursor=%+v slots=%+v", placed, cursor, inv.Slots[:2])
	}
}

func TestCreativePresetSlotClickTakesAndPlacesWithVanillaSemantics(t *testing.T) {
	var presets inventory.CreativeHotbarPresets
	presets[0][2] = inventory.ItemStack{ItemID: inventory.ItemStone, Count: 12, Damage: 3}
	cursor := inventory.ItemStack{}
	if !applyCreativeHotbarPresetClick(&presets, &cursor, 2, 2) {
		t.Fatal("right-click did not take half of saved preset stack")
	}
	if cursor.ItemID != inventory.ItemStone || cursor.Count != 6 || cursor.Damage != 3 || presets[0][2].Count != 6 {
		t.Fatalf("right-click take cursor=%+v slot=%+v", cursor, presets[0][2])
	}
	if !applyCreativeHotbarPresetClick(&presets, &cursor, 3, 1) {
		t.Fatal("left-click did not place cursor into an empty preset slot")
	}
	if !cursor.IsEmpty() || presets[0][3] != (inventory.ItemStack{ItemID: inventory.ItemStone, Count: 6, Damage: 3}) {
		t.Fatalf("left-click place cursor=%+v target=%+v", cursor, presets[0][3])
	}
}

func TestCreativePresetDragDistributesAndPersistsStackData(t *testing.T) {
	var presets inventory.CreativeHotbarPresets
	presets[1][0] = inventory.ItemStack{ItemID: inventory.ItemStone, Count: 62, Damage: 4}
	cursor := inventory.ItemStack{ItemID: inventory.ItemStone, Count: 6, Damage: 4}
	placed := distributeCreativePresetCursor(&presets, &cursor, []int{9, 10, 10, 99}, 1)
	if placed != 5 || cursor.Count != 1 {
		t.Fatalf("placed=%d cursor=%+v, want 5 placed and one remaining", placed, cursor)
	}
	if presets[1][0] != (inventory.ItemStack{ItemID: inventory.ItemStone, Count: 64, Damage: 4}) || presets[1][1] != (inventory.ItemStack{ItemID: inventory.ItemStone, Count: 3, Damage: 4}) {
		t.Fatalf("preset stacks after drag = %+v %+v", presets[1][0], presets[1][1])
	}
}

func TestCreativePresetDragExcludesSourceAndDuplicateTargets(t *testing.T) {
	drag := creativeInventoryDrag{SourcePresetIndex: 4, SourceIsPreset: true}
	drag.addPresetTarget(4)
	drag.addPresetTarget(5)
	drag.addPresetTarget(5)
	if len(drag.TargetPresetSlots) != 1 || drag.TargetPresetSlots[0] != 5 {
		t.Fatalf("preset targets=%v, want only slot 5", drag.TargetPresetSlots)
	}
	drag.reset()
	if drag.Active || len(drag.TargetPresetSlots) != 0 || drag.SourceIsPreset {
		t.Fatalf("drag reset left stale state: %+v", drag)
	}
}

func TestCreativeCatalogWheelUsesWin32NotchesAndRetainsRemainder(t *testing.T) {
	row, remainder := scrollCreativeCatalog(5, 60, 20, 0)
	if row != 5 || remainder != 60 {
		t.Fatalf("half-notch up = row %d remainder %d, want row 5 remainder 60", row, remainder)
	}
	row, remainder = scrollCreativeCatalog(row, 60, 20, remainder)
	if row != 4 || remainder != 0 {
		t.Fatalf("second half-notch up = row %d remainder %d, want row 4 remainder 0", row, remainder)
	}
	row, remainder = scrollCreativeCatalog(row, -360, 20, remainder)
	if row != 7 || remainder != 0 {
		t.Fatalf("three notches down = row %d remainder %d, want row 7 remainder 0", row, remainder)
	}
}

func TestCreativeCatalogWheelClampsAndClearsOutwardRemainder(t *testing.T) {
	row, remainder := scrollCreativeCatalog(1, 360, 20, 0)
	if row != 0 || remainder != 0 {
		t.Fatalf("scroll past top = row %d remainder %d, want both zero", row, remainder)
	}
	row, remainder = scrollCreativeCatalog(19, -360, 20, 0)
	if row != 20 || remainder != 0 {
		t.Fatalf("scroll past bottom = row %d remainder %d, want row 20 remainder 0", row, remainder)
	}
	row, remainder = scrollCreativeCatalog(9, 120, 0, 60)
	if row != 0 || remainder != 0 {
		t.Fatalf("non-scrollable catalog = row %d remainder %d, want both zero", row, remainder)
	}
}
