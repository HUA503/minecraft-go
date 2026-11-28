package sim

import (
	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

// tickConcretePowderLocked applies gravity and water hardening to loaded powder
// blocks. Engine.mu must be held by the caller.
func (e *Engine) tickConcretePowderLocked() {
	for pos := range e.concretePowderTicks {
		blockID := e.getBlockLocked(pos.X, pos.Y, pos.Z)
		if !inventory.IsConcretePowderBlock(blockID) {
			delete(e.concretePowderTicks, pos)
			continue
		}
		if concreteID, touchesWater := e.concretePowderWaterContactLocked(pos); touchesWater {
			e.setBlockLocked(pos.X, pos.Y, pos.Z, concreteID)
			continue
		}
		if pos.Y <= world.MinWorldY {
			delete(e.concretePowderTicks, pos)
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
			delete(e.concretePowderTicks, pos)
			continue
		}
		e.setBlockLocked(pos.X, pos.Y, pos.Z, 0)
		e.setBlockLocked(below.X, below.Y, below.Z, blockID)
	}
}

func (e *Engine) concretePowderWaterContactLocked(pos world.BlockPos) (int32, bool) {
	for _, offset := range [6][3]int32{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
		neighbor := e.getBlockLocked(pos.X+offset[0], pos.Y+offset[1], pos.Z+offset[2])
		if neighbor != 8 && neighbor != 9 {
			continue
		}
		concreteID, ok := inventory.ConcretePowderHardenedBlock(e.getBlockLocked(pos.X, pos.Y, pos.Z))
		return concreteID, ok
	}
	return 0, false
}
