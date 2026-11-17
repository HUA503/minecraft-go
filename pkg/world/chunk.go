package world

import (
	"fmt"
	"math"
)

const (
	MinSectionY = -4
	MaxSectionY = 19
	NumSections = 24 // -4 to 19 inclusive (-64 to 319 height)
	ChunkWidth  = 16
	ChunkHeight = 384
	MinWorldY   = -64
	MaxWorldY   = 319
)

// ChunkPos represents the (X, Z) coordinates of a chunk in chunk space.
type ChunkPos struct {
	X int32
	Z int32
}

func (cp ChunkPos) String() string {
	return fmt.Sprintf("[%d, %d]", cp.X, cp.Z)
}

// BlockPos represents an absolute 3D position in world space.
type BlockPos struct {
	X int32
	Y int32
	Z int32
}

func (bp BlockPos) ToChunkPos() ChunkPos {
	return ChunkPos{
		X: int32(math.Floor(float64(bp.X) / 16.0)),
		Z: int32(math.Floor(float64(bp.Z) / 16.0)),
	}
}

func (bp BlockPos) ToSectionY() int {
	return int(math.Floor(float64(bp.Y)/16.0)) - MinSectionY
}

// Section represents a 16x16x16 sub-chunk section.
type Section struct {
	YIndex      int // 0 to 23
	BlockStates *PalettedContainer
	Biomes      *PalettedContainer
}

func NewSection(yIndex int) *Section {
	return &Section{
		YIndex:      yIndex,
		BlockStates: NewPalettedContainer(4096, 0), // 4096 blocks, default air (0)
		Biomes:      NewPalettedContainer(64, 1),   // 64 biomes (4x4x4), default plains (1)
	}
}

func (s *Section) GetBlock(localX, localY, localZ int) int32 {
	idx := (localY << 8) | (localZ << 4) | localX
	return s.BlockStates.Get(idx)
}

func (s *Section) SetBlock(localX, localY, localZ int, state int32) {
	idx := (localY << 8) | (localZ << 4) | localX
	s.BlockStates.Set(idx, state)
}

// Chunk represents a full 16x384x16 vertical column of 24 sections.
type Chunk struct {
	Pos      ChunkPos
	Sections [NumSections]*Section
}

func NewChunk(pos ChunkPos) *Chunk {
	c := &Chunk{Pos: pos}
	for i := 0; i < NumSections; i++ {
		c.Sections[i] = NewSection(i)
	}
	return c
}

// GetBlock returns the block state ID at absolute world coordinates.
func (c *Chunk) GetBlock(x, y, z int32) int32 {
	if y < MinWorldY || y > MaxWorldY {
		return 0 // Air
	}
	secIdx := int(math.Floor(float64(y)/16.0)) - MinSectionY
	if secIdx < 0 || secIdx >= NumSections {
		return 0
	}
	localX := int(x & 15)
	localY := int(y & 15)
	localZ := int(z & 15)
	return c.Sections[secIdx].GetBlock(localX, localY, localZ)
}

// SetBlock sets the block state ID at absolute world coordinates.
func (c *Chunk) SetBlock(x, y, z int32, state int32) {
	if y < MinWorldY || y > MaxWorldY {
		return
	}
	secIdx := int(math.Floor(float64(y)/16.0)) - MinSectionY
	if secIdx < 0 || secIdx >= NumSections {
		return
	}
	localX := int(x & 15)
	localY := int(y & 15)
	localZ := int(z & 15)
	c.Sections[secIdx].SetBlock(localX, localY, localZ, state)
}
