package sim

import (
	"math"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func activeEffectIndex(player *PlayerEntity, id string) int {
	for i := 0; i < player.EffectCount; i++ {
		if player.ActiveEffects[i].ID == id {
			return i
		}
	}
	return -1
}

func (e *Engine) addPlayerEffectLocked(player *PlayerEntity, effect inventory.FoodStatusEffect) {
	if effect.ID == "" || effect.DurationTicks <= 0 || effect.Amplifier < 0 {
		return
	}
	index := activeEffectIndex(player, effect.ID)
	if index >= 0 {
		current := player.ActiveEffects[index]
		if effect.Amplifier < current.Amplifier || (effect.Amplifier == current.Amplifier && effect.DurationTicks <= current.DurationTicks) {
			return
		}
		if effect.ID == inventory.EffectAbsorption {
			player.Absorption = float32(math.Max(0, float64(player.Absorption-4*float32(current.Amplifier+1))))
		}
	} else if player.EffectCount == len(player.ActiveEffects) {
		return
	} else {
		index = player.EffectCount
		player.EffectCount++
	}
	player.ActiveEffects[index] = world.PlayerStatusEffect{ID: effect.ID, DurationTicks: effect.DurationTicks, Amplifier: effect.Amplifier}
	if effect.ID == inventory.EffectAbsorption {
		player.Absorption = float32(math.Max(float64(player.Absorption), float64(4*(effect.Amplifier+1))))
	}
}

func (e *Engine) consumeFoodEffectsLocked(player *PlayerEntity, food inventory.FoodProperties) {
	if food.ClearAllEffects {
		player.EffectCount = 0
		player.ActiveEffects = [16]world.PlayerStatusEffect{}
		player.Absorption = 0
	}
	for _, id := range food.RemoveEffects {
		if index := activeEffectIndex(player, id); index >= 0 {
			e.removePlayerEffectLocked(player, index)
		}
	}
	for _, effect := range food.Effects {
		chance := effect.Chance
		if chance <= 0 || chance > 1 {
			chance = 1
		}
		if float64(e.nextRandomTickValueLocked()%1_000_000)/1_000_000 < chance {
			e.addPlayerEffectLocked(player, effect)
		}
	}
	if food.TeleportRandomly {
		e.teleportPlayerRandomlyLocked(player)
	}
}

func (e *Engine) removePlayerEffectLocked(player *PlayerEntity, index int) {
	if index < 0 || index >= player.EffectCount {
		return
	}
	removed := player.ActiveEffects[index]
	if removed.ID == inventory.EffectAbsorption {
		player.Absorption = float32(math.Max(0, float64(player.Absorption-4*float32(removed.Amplifier+1))))
	}
	copy(player.ActiveEffects[index:], player.ActiveEffects[index+1:player.EffectCount])
	player.EffectCount--
	player.ActiveEffects[player.EffectCount] = world.PlayerStatusEffect{}
}

func (e *Engine) tickPlayerStatusEffectsLocked(player *PlayerEntity) {
	for index := 0; index < player.EffectCount; {
		effect := &player.ActiveEffects[index]
		effect.DurationTicks--
		interval := 0
		switch effect.ID {
		case inventory.EffectHunger:
			if player.GameMode != 1 {
				player.Exhaustion += 0.005 * float32(effect.Amplifier+1)
			}
		case inventory.EffectPoison:
			interval = max(1, 25>>effect.Amplifier)
			if effect.DurationTicks > 0 && effect.DurationTicks%interval == 0 && player.Health > 1 {
				e.applyPlayerDamageLocked(player, min(float32(1), player.Health-1), "magic")
			}
		case inventory.EffectRegeneration:
			interval = max(1, 50>>effect.Amplifier)
			if effect.DurationTicks > 0 && effect.DurationTicks%interval == 0 && player.Health < 20 {
				player.Health = min(20, player.Health+1)
			}
		}
		if effect.DurationTicks <= 0 {
			e.removePlayerEffectLocked(player, index)
			continue
		}
		index++
	}
}

func (e *Engine) teleportPlayerRandomlyLocked(player *PlayerEntity) bool {
	if player == nil {
		return false
	}
	for attempt := 0; attempt < 16; attempt++ {
		random := e.nextRandomTickValueLocked()
		x := int32(math.Floor(player.Position.X)) + int32(random%17) - 8
		y := int32(math.Floor(player.Position.Y)) + int32((random>>8)%17) - 8
		z := int32(math.Floor(player.Position.Z)) + int32((random>>16)%17) - 8
		if y <= world.MinWorldY || y+2 >= world.MaxWorldY {
			continue
		}
		feet := Vec3{X: float64(x) + 0.5, Y: float64(y), Z: float64(z) + 0.5}
		chunk, loaded := e.World[world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()]
		if !loaded || chunk == nil || !IsSolidBlock(e.getBlockLocked(x, y-1, z)) {
			continue
		}
		box := NewPlayerAABB(feet)
		blocked := false
		for by := int32(math.Floor(box.Min.Y)); by < int32(math.Ceil(box.Max.Y)) && !blocked; by++ {
			for bz := int32(math.Floor(box.Min.Z)); bz < int32(math.Ceil(box.Max.Z)) && !blocked; bz++ {
				for bx := int32(math.Floor(box.Min.X)); bx < int32(math.Ceil(box.Max.X)); bx++ {
					if IsSolidBlock(e.getBlockLocked(bx, by, bz)) {
						blocked = true
						break
					}
				}
			}
		}
		if blocked {
			continue
		}
		player.Position = feet
		player.Velocity = Vec3{}
		player.FallDistance = 0
		player.OnGround = true
		select {
		case e.Events <- WorldEvent{Type: "player_teleport", PlayerID: player.ID, Position: feet}:
		default:
		}
		return true
	}
	return false
}
