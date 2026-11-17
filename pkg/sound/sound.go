package sound

import (
	"sync"
)

// SoundCategory categorizes audio sources according to Minecraft standards.
type SoundCategory int

const (
	CategoryMaster SoundCategory = iota
	CategoryMusic
	CategoryRecords
	CategoryWeather
	CategoryBlocks
	CategoryHostile
	CategoryNeutral
	CategoryPlayers
	CategoryAmbient
	CategoryVoice
)

// Standard Minecraft Sound Event Identifiers.
const (
	SoundBlockGrassBreak   = "minecraft:block.grass.break"
	SoundBlockGrassStep    = "minecraft:block.grass.step"
	SoundBlockGrassPlace   = "minecraft:block.grass.place"
	SoundBlockStoneBreak   = "minecraft:block.stone.break"
	SoundBlockStoneStep    = "minecraft:block.stone.step"
	SoundBlockStonePlace   = "minecraft:block.stone.place"
	SoundBlockWoodBreak    = "minecraft:block.wood.break"
	SoundBlockWoodStep     = "minecraft:block.wood.step"
	SoundBlockWoodPlace    = "minecraft:block.wood.place"
	SoundPlayerHurt        = "minecraft:entity.player.hurt"
	SoundPlayerDeath       = "minecraft:entity.player.death"
	SoundPlayerLevelUp     = "minecraft:entity.player.levelup"
	SoundZombieAmbient     = "minecraft:entity.zombie.ambient"
	SoundZombieAttack      = "minecraft:entity.zombie.attack_wooden_door"
	SoundSkeletonShoot     = "minecraft:entity.skeleton.shoot"
	SoundCreeperPrimed     = "minecraft:entity.creeper.primed"
	SoundGenericExplode    = "minecraft:entity.generic.explode"
	SoundItemToolBreak     = "minecraft:entity.item.break"
	SoundItemPickup        = "minecraft:entity.item.pickup"
)

// SoundEvent carries parameters for an emitted sound.
type SoundEvent struct {
	Name     string
	Category SoundCategory
	X, Y, Z  float64
	Volume   float32
	Pitch    float32
}

// SoundListener receives emitted sounds.
type SoundListener func(event SoundEvent)

// SoundManager manages sound registration, volume, and dispatching.
type SoundManager struct {
	mu        sync.RWMutex
	listeners []SoundListener
	MasterVol float32
}

// NewSoundManager creates an active sound manager.
func NewSoundManager() *SoundManager {
	return &SoundManager{
		MasterVol: 1.0,
	}
}

// AddListener registers a receiver for sound events.
func (sm *SoundManager) AddListener(listener SoundListener) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listeners = append(sm.listeners, listener)
}

// PlaySound dispatches a sound event to all registered audio listeners.
func (sm *SoundManager) PlaySound(name string, cat SoundCategory, x, y, z float64, vol, pitch float32) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	effectiveVol := vol * sm.MasterVol
	event := SoundEvent{
		Name:     name,
		Category: cat,
		X:        x,
		Y:        y,
		Z:        z,
		Volume:   effectiveVol,
		Pitch:    pitch,
	}

	for _, l := range sm.listeners {
		l(event)
	}
}

// GetBlockBreakSound returns the authentic breaking sound event for a block ID.
func GetBlockBreakSound(blockID int32) string {
	switch blockID {
	case 2, 3, 12: // Grass, Dirt, Sand
		return SoundBlockGrassBreak
	case 1, 4, 14, 15, 16, 56: // Stone, Cobblestone, Ores
		return SoundBlockStoneBreak
	case 5, 17, 58: // Planks, Log, Crafting Table
		return SoundBlockWoodBreak
	default:
		return SoundBlockStoneBreak
	}
}

// GetBlockPlaceSound returns the authentic placement sound event for a block ID.
func GetBlockPlaceSound(blockID int32) string {
	switch blockID {
	case 2, 3, 12:
		return SoundBlockGrassPlace
	case 1, 4, 14, 15, 16, 56:
		return SoundBlockStonePlace
	case 5, 17, 58:
		return SoundBlockWoodPlace
	default:
		return SoundBlockStonePlace
	}
}
