package main

import (
	"math"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/render"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/sound"
)

// updateSpearCharge owns secondary-button spear input. Returning true prevents
// the same button press from placing blocks or interacting with containers.
func (c *Client) updateSpearCharge(isRightDown bool) bool {
	selected := c.player.Inventory.GetSelectedItem()
	properties, selectedIsSpear := inventory.GetSpearProperties(selected.ItemID)
	if !selectedIsSpear {
		if c.spearChargeItem != 0 {
			c.spearChargeItem = 0
			c.spearChargeStarted = time.Time{}
			return true
		}
		return false
	}
	if !c.isLocal {
		return true
	}

	if isRightDown {
		if c.spearChargeItem == 0 {
			c.spearChargeItem = selected.ItemID
			c.spearChargeStarted = time.Now()
		}
		return true
	}
	if c.spearChargeItem == 0 {
		return true
	}

	started := c.spearChargeStarted
	chargedItem := c.spearChargeItem
	c.spearChargeItem = 0
	c.spearChargeStarted = time.Time{}
	if chargedItem != selected.ItemID || started.IsZero() {
		return true
	}
	if c.camera == nil {
		return true
	}

	elapsedTicks := int(time.Since(started) / (time.Second / 20))
	if elapsedTicks < properties.ChargeDelayTicks {
		return true
	}
	elapsedAfterDelay := elapsedTicks - properties.ChargeDelayTicks
	if c.isCreative {
		properties.MaxAttackReach = properties.CreativeMaxReach
	}
	lx, ly, lz := c.camera.GetLookVector()
	hits := c.engine.AttackMobsWithSpearCharge(
		sim.Vec3{X: c.camera.X, Y: c.camera.Y, Z: c.camera.Z},
		sim.Vec3{X: lx, Y: ly, Z: lz},
		sim.Vec3{X: float64(c.player.Velocity.X), Y: float64(c.player.Velocity.Y), Z: float64(c.player.Velocity.Z)},
		properties, elapsedAfterDelay,
	)
	if len(hits) > 0 {
		applyAttackItemDurability(c.player.Inventory, c.isCreative)
		c.isSwinging = true
		c.swingProgress = 0
		for _, hit := range hits {
			sound.PlayHardwareFeedback("minecraft:entity.generic.hurt")
			for i := 0; i < 8; i++ {
				c.particles = append(c.particles, render.Particle{
					X:     hit.Position.X + (randFloat()-0.5)*0.5,
					Y:     hit.Position.Y + 0.9 + randFloat()*0.45,
					Z:     hit.Position.Z + (randFloat()-0.5)*0.5,
					Vx:    (randFloat() - 0.5) * 1.6,
					Vy:    randFloat()*1.5 + 0.2,
					Vz:    (randFloat() - 0.5) * 1.6,
					Color: 0xFFFF2222, Life: 0.3, MaxLife: 0.3,
				})
			}
			if hit.Killed {
				sound.PlayHardwareFeedback("minecraft:entity.generic.death")
			}
		}
	}
	return true
}

// updateSpyglass applies the held-use behavior and consumes the right-click so
// the spyglass cannot also place or activate the targeted block.
func (c *Client) updateSpyglass(isRightDown bool) bool {
	selected := c.player.Inventory.GetSelectedItem()
	if selected.ItemID != inventory.ItemSpyglass {
		c.isUsingSpyglass = false
		return false
	}
	c.isUsingSpyglass = isRightDown && c.isLocal &&
		!c.showInventoryGUI && !c.isCraftingTableOpen && !c.isFurnaceOpen &&
		!c.showPauseMenu && !c.showAdvancementsGUI
	return true
}

func targetPlayerFOV(usingSpyglass, sprinting, moving, onGround bool) float64 {
	if usingSpyglass {
		return 7.0
	}
	if sprinting && moving && onGround {
		return 82.0
	}
	return 70.0
}

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
	case inventory.ToolSword, inventory.ToolSpear:
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
