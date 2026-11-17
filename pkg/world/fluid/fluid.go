package fluid

import (
	"github.com/HUA503/minecraft-go/pkg/world/gen"
)

// FluidType identifies water or lava.
type FluidType int

const (
	FluidWater FluidType = iota
	FluidLava
)

// Simulator manages tick-based fluid spreading and interaction.
type Simulator struct {
	worldGetter func(x, y, z int32) int32
	worldSetter func(x, y, z int32, blockID int32)
}

func NewSimulator(getter func(x, y, z int32) int32, setter func(x, y, z int32, blockID int32)) *Simulator {
	return &Simulator{
		worldGetter: getter,
		worldSetter: setter,
	}
}

// SpreadStep simulates one step of fluid spreading from source at (x, y, z).
func (s *Simulator) SpreadStep(x, y, z int32, fluid FluidType) {
	currBlock := s.worldGetter(x, y, z)
	if fluid == FluidWater && currBlock != gen.BlockWater && currBlock != gen.BlockFlowingWater {
		return
	}
	if fluid == FluidLava && currBlock != gen.BlockLava && currBlock != gen.BlockFlowingLava {
		return
	}

	// 1. Check downwards flow first
	belowY := y - 1
	belowBlock := s.worldGetter(x, belowY, z)
	if belowBlock == gen.BlockAir {
		flowingID := gen.BlockFlowingWater
		if fluid == FluidLava {
			flowingID = gen.BlockFlowingLava
		}
		s.worldSetter(x, belowY, z, flowingID)
		return
	}

	// 2. Check fluid interactions (Water meeting Lava)
	if fluid == FluidWater {
		if belowBlock == gen.BlockLava {
			s.worldSetter(x, belowY, z, gen.BlockObsidian)
			return
		} else if belowBlock == gen.BlockFlowingLava {
			s.worldSetter(x, belowY, z, gen.BlockCobblestone)
			return
		}
	}

	// 3. Horizontal spreading
	neighbors := [][2]int32{
		{1, 0}, {-1, 0}, {0, 1}, {0, -1},
	}

	for _, off := range neighbors {
		nx := x + off[0]
		nz := z + off[1]
		nb := s.worldGetter(nx, y, nz)

		if nb == gen.BlockAir {
			flowID := gen.BlockFlowingWater
			if fluid == FluidLava {
				flowID = gen.BlockFlowingLava
			}
			s.worldSetter(nx, y, nz, flowID)
		} else if fluid == FluidWater && nb == gen.BlockLava {
			// Water touching lava source -> Obsidian
			s.worldSetter(nx, y, nz, gen.BlockObsidian)
		} else if fluid == FluidWater && nb == gen.BlockFlowingLava {
			// Water touching flowing lava -> Cobblestone
			s.worldSetter(nx, y, nz, gen.BlockCobblestone)
		}
	}
}

// CheckInfiniteWaterSource checks if (x, y, z) has >= 2 adjacent water source blocks.
func (s *Simulator) CheckInfiniteWaterSource(x, y, z int32) bool {
	if s.worldGetter(x, y, z) != gen.BlockAir && s.worldGetter(x, y, z) != gen.BlockFlowingWater {
		return false
	}
	// Must have a solid block below to form infinite water
	if s.worldGetter(x, y-1, z) == gen.BlockAir {
		return false
	}

	adjacentSources := 0
	neighbors := [][2]int32{
		{1, 0}, {-1, 0}, {0, 1}, {0, -1},
	}
	for _, off := range neighbors {
		if s.worldGetter(x+off[0], y, z+off[1]) == gen.BlockWater {
			adjacentSources++
		}
	}

	if adjacentSources >= 2 {
		s.worldSetter(x, y, z, gen.BlockWater)
		return true
	}
	return false
}
