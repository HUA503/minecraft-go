package main

import "github.com/HUA503/minecraft-go/pkg/inventory"

// applyAttackItemDurability applies the Java weapon component's default cost of
// one durability point after a successful entity hit. Creative mode never
// damages the held stack. It returns true when the held item breaks.
func applyAttackItemDurability(inv *inventory.PlayerInventory, creative bool) bool {
	if inv == nil || creative || inv.SelectedHotbar < 0 || inv.SelectedHotbar >= 9 {
		return false
	}
	stack := &inv.Slots[inv.SelectedHotbar]
	if stack.IsEmpty() {
		return false
	}
	properties, ok := inventory.GetToolProperties(stack.ItemID)
	if !ok || properties.MaxDurability <= 0 {
		return false
	}
	if inventory.ApplyToolWear(stack) {
		return true
	}
	return false
}
