package main

import (
	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
)

// strawBedPartsForBreak captures both bed cells when mining either half or
// the support block directly underneath either half.
func strawBedPartsForBreak(engine *sim.Engine, brokenPos world.BlockPos) []world.BlockPos {
	if engine == nil {
		return nil
	}
	seen := make(map[world.BlockPos]struct{}, 2)
	parts := make([]world.BlockPos, 0, 2)
	for _, candidate := range []world.BlockPos{
		brokenPos,
		{X: brokenPos.X, Y: brokenPos.Y + 1, Z: brokenPos.Z},
	} {
		state := engine.GetBlockIfLoaded(candidate.X, candidate.Y, candidate.Z)
		if _, _, _, isBed := inventory.StrawBedStateProperties(state); !isBed {
			continue
		}
		dx, dz, otherState, ok := inventory.StrawBedOtherHalf(state)
		if !ok {
			continue
		}
		other := world.BlockPos{X: candidate.X + dx, Y: candidate.Y, Z: candidate.Z + dz}
		if engine.GetBlockIfLoaded(other.X, other.Y, other.Z) != otherState {
			continue
		}
		for _, part := range []world.BlockPos{candidate, other} {
			if _, exists := seen[part]; exists {
				continue
			}
			seen[part] = struct{}{}
			parts = append(parts, part)
		}
	}
	return parts
}
