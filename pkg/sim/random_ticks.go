package sim

import (
	"math"

	"github.com/HUA503/minecraft-go/pkg/world"
)

const (
	poplarLogID     int32 = 10010
	poplarSaplingID int32 = 10063
)

// tickRandomBlocksLocked applies Minecraft-style random samples to loaded
// chunks within simulation distance. The caller owns Engine.mu.
func (e *Engine) tickRandomBlocksLocked() {
	if e.CurrentDimension != 0 || len(e.Players) == 0 || len(e.World) == 0 {
		return
	}

	active := make(map[world.ChunkPos]struct{}, len(e.World))
	for pos := range e.World {
		for _, player := range e.Players {
			if player == nil || !player.IsAlive {
				continue
			}
			centerX := int32(math.Floor(player.Position.X)) >> 4
			centerZ := int32(math.Floor(player.Position.Z)) >> 4
			if absInt32(pos.X-centerX) <= 8 && absInt32(pos.Z-centerZ) <= 8 {
				active[pos] = struct{}{}
				break
			}
		}
	}

	for pos := range active {
		for section := 0; section < world.NumSections; section++ {
			baseY := world.MinWorldY + int32(section*16)
			for sample := 0; sample < 3; sample++ {
				random := e.nextRandomTickValueLocked()
				x := pos.X*world.ChunkWidth + int32(random&15)
				y := baseY + int32((random>>4)&15)
				z := pos.Z*world.ChunkWidth + int32((random>>8)&15)
				if e.getBlockLocked(x, y, z) != poplarSaplingID || e.nextRandomTickValueLocked()%7 != 0 {
					continue
				}
				e.growPoplarSaplingLocked(x, y, z)
			}
		}
	}
}

func (e *Engine) nextRandomTickValueLocked() uint64 {
	x := e.randomTickState
	if x == 0 {
		x = 0x9e3779b97f4a7c15
	}
	x ^= x << 13
	x ^= x >> 7
	x ^= x << 17
	e.randomTickState = x
	return x
}

// growPoplarSaplingLocked replaces a mature, unobstructed sapling with a
// small poplar tree. All touched chunks must already be resident so a random
// tick can never create a blank chunk over unloaded saved data.
func (e *Engine) growPoplarSaplingLocked(x, y, z int32) bool {
	if e.CurrentDimension != 0 || y <= world.MinWorldY || y+5 > world.MaxWorldY || e.getBlockLocked(x, y, z) != poplarSaplingID {
		return false
	}
	soil := e.getBlockLocked(x, y-1, z)
	if soil != 2 && soil != 3 {
		return false
	}
	for skyY := y + 1; skyY <= world.MaxWorldY; skyY++ {
		block := e.getBlockLocked(x, skyY, z)
		if block != 0 && block != 6 && block != 18 && block != 31 && block != 32 &&
			block != poplarSaplingID && !between(block, 10060, 10062) {
			return false
		}
	}

	type placedBlock struct {
		pos world.BlockPos
		id  int32
	}
	blocks := make([]placedBlock, 0, 60)
	for dy := int32(0); dy < 5; dy++ {
		blocks = append(blocks, placedBlock{pos: world.BlockPos{X: x, Y: y + dy, Z: z}, id: poplarLogID})
	}
	for _, layer := range []struct {
		y      int32
		radius int32
	}{{y + 3, 2}, {y + 4, 2}, {y + 5, 1}} {
		for dx := -layer.radius; dx <= layer.radius; dx++ {
			for dz := -layer.radius; dz <= layer.radius; dz++ {
				if layer.radius == 2 && absInt32(dx) == 2 && absInt32(dz) == 2 {
					continue
				}
				if dx == 0 && dz == 0 && layer.y < y+5 {
					continue
				}
				color := int32(10060 + ((dx-dz+int32(layer.y))&3)%3)
				blocks = append(blocks, placedBlock{pos: world.BlockPos{X: x + dx, Y: layer.y, Z: z + dz}, id: color})
			}
		}
	}

	for _, block := range blocks {
		if !world.ValidBlockCoordinates(block.pos.X, block.pos.Y, block.pos.Z) || e.World[block.pos.ToChunkPos()] == nil {
			return false
		}
		current := e.getBlockLocked(block.pos.X, block.pos.Y, block.pos.Z)
		if current != 0 && current != poplarSaplingID && current != 18 && (current < 10060 || current > 10062) {
			return false
		}
	}
	for _, block := range blocks {
		e.setBlockLocked(block.pos.X, block.pos.Y, block.pos.Z, block.id)
	}
	return true
}

func between(value, minimum, maximum int32) bool {
	return value >= minimum && value <= maximum
}

func absInt32(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}
