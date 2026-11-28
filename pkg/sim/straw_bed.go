package sim

import (
	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func (e *Engine) placeStrawBed(playerID int32, footPos world.BlockPos, footState int32) bool {
	dx, dz, headState, ok := inventory.StrawBedOtherHalf(footState)
	if !ok {
		return false
	}
	headPos := world.BlockPos{X: footPos.X + dx, Y: footPos.Y, Z: footPos.Z + dz}
	if !world.ValidBlockCoordinates(footPos.X, footPos.Y, footPos.Z) ||
		!world.ValidBlockCoordinates(headPos.X, headPos.Y, headPos.Z) || footPos.Y <= world.MinWorldY {
		return false
	}
	if !inventory.IsReplaceableBlock(e.GetBlock(footPos.X, footPos.Y, footPos.Z)) ||
		!inventory.IsReplaceableBlock(e.GetBlock(headPos.X, headPos.Y, headPos.Z)) ||
		!IsSolidBlock(e.GetBlock(footPos.X, footPos.Y-1, footPos.Z)) ||
		!IsSolidBlock(e.GetBlock(headPos.X, headPos.Y-1, headPos.Z)) {
		return false
	}
	if e.placementIntersectsPlayer(doorPosState{pos: footPos, state: footState}) ||
		e.placementIntersectsPlayer(doorPosState{pos: headPos, state: headState}) {
		return false
	}
	if !e.SetBlockChecked(footPos.X, footPos.Y, footPos.Z, footState) {
		return false
	}
	if !e.SetBlockChecked(headPos.X, headPos.Y, headPos.Z, headState) {
		_ = e.SetBlockChecked(footPos.X, footPos.Y, footPos.Z, 0)
		return false
	}
	e.consumePlacedBlockItem(playerID, inventory.ItemStrawBed)
	return true
}

func (e *Engine) removeStrawBedOtherHalf(pos world.BlockPos, state int32) {
	dx, dz, pairedState, ok := inventory.StrawBedOtherHalf(state)
	if !ok {
		return
	}
	other := world.BlockPos{X: pos.X + dx, Y: pos.Y, Z: pos.Z + dz}
	if e.GetBlock(other.X, other.Y, other.Z) == pairedState {
		_ = e.SetBlockChecked(other.X, other.Y, other.Z, 0)
	}
}

func (e *Engine) removeUnsupportedStrawBedAbove(supportPos world.BlockPos) int32 {
	if supportPos.Y >= world.MaxWorldY {
		return 0
	}
	bedPos := world.BlockPos{X: supportPos.X, Y: supportPos.Y + 1, Z: supportPos.Z}
	state := e.GetBlock(bedPos.X, bedPos.Y, bedPos.Z)
	itemID, _, _, isBed := inventory.StrawBedStateProperties(state)
	if !isBed || IsSolidBlock(e.GetBlock(supportPos.X, supportPos.Y, supportPos.Z)) {
		return 0
	}
	dx, dz, pairedState, ok := inventory.StrawBedOtherHalf(state)
	if !ok {
		return 0
	}
	other := world.BlockPos{X: bedPos.X + dx, Y: bedPos.Y, Z: bedPos.Z + dz}
	if e.GetBlock(other.X, other.Y, other.Z) == pairedState {
		_ = e.SetBlockChecked(other.X, other.Y, other.Z, 0)
	}
	if e.SetBlockChecked(bedPos.X, bedPos.Y, bedPos.Z, 0) {
		return itemID
	}
	return 0
}

// UseStrawBed reports whether sleep succeeded and whether the one-use bed was destroyed.
// Straw beds do not set a respawn point; attempts outside the Overworld destroy the bed.
func (e *Engine) UseStrawBed(playerID int32, pos world.BlockPos) (slept, destroyed bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	player := e.Players[playerID]
	if player == nil || !player.IsAlive {
		return false, false
	}
	state := e.getBlockLocked(pos.X, pos.Y, pos.Z)
	if _, _, _, ok := inventory.StrawBedStateProperties(state); !ok {
		return false, false
	}
	removeBed := func() {
		e.setBlockLocked(pos.X, pos.Y, pos.Z, 0)
		dx, dz, pairedState, ok := inventory.StrawBedOtherHalf(state)
		if !ok {
			return
		}
		other := world.BlockPos{X: pos.X + dx, Y: pos.Y, Z: pos.Z + dz}
		if e.getBlockLocked(other.X, other.Y, other.Z) == pairedState {
			e.setBlockLocked(other.X, other.Y, other.Z, 0)
		}
	}
	if e.CurrentDimension != 0 {
		removeBed()
		return false, true
	}
	if e.WorldTime < 12542 || e.WorldTime > 23459 {
		return false, false
	}
	removeBed()
	e.WorldTime = 0
	return true, true
}
