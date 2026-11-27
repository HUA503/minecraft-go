package main

import (
	"math"

	"github.com/HUA503/minecraft-go/pkg/inventory"
)

// maceSmashBonus applies Java's Mace smash falloff to whole fallen blocks:
// the first 3 add 4 damage each, the next 5 add 2, and later blocks add 1.
// A smash only happens once the player has fallen more than 1.5 blocks.
func maceSmashBonus(fallDistance float32, onGround bool) float32 {
	if onGround || fallDistance <= 1.5 {
		return 0
	}
	blocks := int(math.Floor(float64(fallDistance)))
	bonus := min(blocks, 3) * 4
	if blocks > 3 {
		bonus += min(blocks-3, 5) * 2
	}
	if blocks > 8 {
		bonus += blocks - 8
	}
	return float32(bonus)
}

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
	damage := 0
	switch properties.Type {
	case inventory.ToolSword:
		damage = 1
	case inventory.ToolPickaxe, inventory.ToolAxe, inventory.ToolShovel, inventory.ToolHoe:
		damage = 2
	default:
		return false
	}
	if inventory.ApplyToolWearAmount(stack, damage) {
		return true
	}
	return false
}
