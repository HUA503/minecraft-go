package main

import (
	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/render"
)

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
