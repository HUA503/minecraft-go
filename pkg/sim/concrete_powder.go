package sim

import (
	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

// tickConcretePowderLocked applies gravity and water hardening to loaded powder
// blocks. Engine.mu must be held by the caller.
func (e *Engine) tickConcretePowderLocked() {
	if e.CurrentDimension < 0 || e.CurrentDimension >= len(e.concretePowderTicks) {
		return
	}
	for pos := range e.concretePowderTicks[e.CurrentDimension] {
		blockID := e.getBlockLocked(pos.X, pos.Y, pos.Z)
		if !inventory.IsConcretePowderBlock(blockID) {
			delete(e.concretePowderTicks[e.CurrentDimension], pos)
			continue
		}
		if concreteID, touchesWater := e.concretePowderWaterContactLocked(pos, blockID); touchesWater {
			e.setBlockLocked(pos.X, pos.Y, pos.Z, concreteID)
			continue
		}
		if pos.Y <= world.MinWorldY {
			delete(e.concretePowderTicks[e.CurrentDimension], pos)
			continue
		}
		below := world.BlockPos{X: pos.X, Y: pos.Y - 1, Z: pos.Z}
		if e.World[below.ToChunkPos()] == nil {
			continue
		}
		belowID := e.getBlockLocked(below.X, below.Y, below.Z)
		if belowID == 8 || belowID == 9 {
			concreteID, _ := inventory.ConcretePowderHardenedBlock(blockID)
			e.setBlockLocked(pos.X, pos.Y, pos.Z, concreteID)
			continue
		}
		if !inventory.IsReplaceableBlock(belowID) {
			delete(e.concretePowderTicks[e.CurrentDimension], pos)
			continue
		}
		e.setBlockLocked(pos.X, pos.Y, pos.Z, 0)
		e.setBlockLocked(below.X, below.Y, below.Z, blockID)
	}
}

// scheduleConcretePowderChunkLocked restores transient block-tick state after a
// chunk is loaded. Scan packed in-memory section data because the local chunk
// decoder stores block IDs in Data while its optional Palette cache can be stale.
// Engine.mu must be held by the caller.
func (e *Engine) scheduleConcretePowderChunkLocked(dim int, chunk *world.Chunk) {
	if dim < 0 || dim >= len(e.concretePowderTicks) || chunk == nil {
		return
	}
	ticks := e.concretePowderTicks[dim]
	if ticks == nil {
		ticks = make(map[world.BlockPos]struct{})
		e.concretePowderTicks[dim] = ticks
	}
	for sectionIndex, section := range chunk.Sections {
		if section == nil || section.BlockStates == nil {
			continue
		}
		data := section.BlockStates.Data
		if capacity := section.BlockStates.Capacity; len(data) > capacity {
			data = data[:capacity]
		}
		baseY := int32((world.MinSectionY + sectionIndex) * 16)
		for index, state := range data {
			if !inventory.IsConcretePowderBlock(state) {
				continue
			}
			localX := int32(index & 15)
			localZ := int32((index >> 4) & 15)
			localY := int32(index >> 8)
			ticks[world.BlockPos{
				X: chunk.Pos.X*world.ChunkWidth + localX,
				Y: baseY + localY,
				Z: chunk.Pos.Z*world.ChunkWidth + localZ,
			}] = struct{}{}
		}
	}
}

func (e *Engine) concretePowderWaterContactLocked(pos world.BlockPos, powderID int32) (int32, bool) {
	for _, offset := range [6][3]int32{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
		neighbor := e.getBlockLocked(pos.X+offset[0], pos.Y+offset[1], pos.Z+offset[2])
		if neighbor != 8 && neighbor != 9 {
			continue
		}
		concreteID, ok := inventory.ConcretePowderHardenedBlock(powderID)
		return concreteID, ok
	}
	return 0, false
}
