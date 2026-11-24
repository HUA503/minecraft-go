package main

import (
	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/render"
)

// scrollCreativeCatalog converts Win32 wheel deltas (120 units per notch) into
// catalog rows while retaining high-resolution partial-wheel movement.
func scrollCreativeCatalog(current, delta, maxScroll, remainder int) (int, int) {
	if maxScroll <= 0 {
		return 0, 0
	}
	total := delta + remainder
	steps := total / 120
	remainder = total - steps*120
	current -= steps
	if current < 0 {
		current = 0
		if remainder > 0 {
			remainder = 0
		}
	} else if current > maxScroll {
		current = maxScroll
		if remainder < 0 {
			remainder = 0
		}
	}
	return current, remainder
}

type creativeInventoryDrag struct {
	Active      bool
	Button      int
	StartIndex  int
	TargetSlots []int
}

func (drag *creativeInventoryDrag) addTarget(index int) {
	if drag == nil || index < 0 {
		return
	}
	for _, existing := range drag.TargetSlots {
		if existing == index {
			return
		}
	}
	drag.TargetSlots = append(drag.TargetSlots, index)
}

func (drag *creativeInventoryDrag) reset() {
	if drag == nil {
		return
	}
	*drag = creativeInventoryDrag{}
}

// applyCreativeHotbarKey applies the creative inventory number-key shortcut to
// the hovered catalog item or player inventory slot.
func applyCreativeHotbarKey(inv *inventory.PlayerInventory, hit render.GUISlotHit, items []inventory.CreativeItem, hotbarSlot int, presets ...inventory.CreativeHotbarPresets) bool {
	if inv == nil || hotbarSlot < 0 || hotbarSlot >= 9 {
		return false
	}
	switch hit.Type {
	case render.SlotCreativeItem:
		if len(presets) > 0 && hit.Index >= 0 && hit.Index < 81 {
			stack := presets[0][hit.Index/9][hit.Index%9]
			if stack.IsEmpty() {
				return false
			}
			stack.Count = inventory.GetMaxStackSize(stack.ItemID)
			inv.Slots[hotbarSlot] = stack
			return true
		}
		if hit.Index < 0 || hit.Index >= len(items) {
			return false
		}
		itemID := items[hit.Index].ID
		if itemID <= 0 {
			return false
		}
		inv.Slots[hotbarSlot] = inventory.CreativeItemStack(itemID, 1)
		return true
	case render.SlotInventory:
		if hit.Index < 0 || hit.Index >= len(inv.Slots) || hit.Index == hotbarSlot {
			return false
		}
		inv.Slots[hotbarSlot], inv.Slots[hit.Index] = inv.Slots[hit.Index], inv.Slots[hotbarSlot]
		return true
	default:
		return false
	}
}

// distributeCreativeCursor applies the vanilla creative quick-craft modes:
// left-drag divides the cursor stack evenly; right-drag places one per slot.
func distributeCreativeCursor(inv *inventory.PlayerInventory, cursor *inventory.ItemStack, targets []int, button int) int {
	if inv == nil || cursor == nil || cursor.IsEmpty() || (button != 1 && button != 2) {
		return 0
	}
	maxStack := inventory.GetMaxStackSize(cursor.ItemID)
	if maxStack <= 0 {
		return 0
	}
	eligible := make([]int, 0, len(targets))
	seen := make(map[int]struct{}, len(targets))
	for _, index := range targets {
		if index < 0 || index >= len(inv.Slots) {
			continue
		}
		if _, exists := seen[index]; exists {
			continue
		}
		seen[index] = struct{}{}
		slot := inv.Slots[index]
		if !slot.IsEmpty() && (slot.ItemID != cursor.ItemID || slot.Damage != cursor.Damage) {
			continue
		}
		if slot.Count >= maxStack {
			continue
		}
		eligible = append(eligible, index)
	}
	if len(eligible) > cursor.Count {
		eligible = eligible[:cursor.Count]
	}
	if len(eligible) == 0 {
		return 0
	}
	perSlot := 1
	if button == 1 {
		perSlot = cursor.Count / len(eligible)
		if perSlot == 0 {
			return 0
		}
	}
	placed := 0
	for _, index := range eligible {
		slot := &inv.Slots[index]
		room := maxStack - slot.Count
		add := min(perSlot, room, cursor.Count-placed)
		if add <= 0 {
			continue
		}
		if slot.IsEmpty() {
			*slot = inventory.ItemStack{ItemID: cursor.ItemID, Damage: cursor.Damage}
		}
		slot.Count += add
		placed += add
	}
	cursor.Count -= placed
	if cursor.Count <= 0 {
		*cursor = inventory.ItemStack{}
	}
	return placed
}
