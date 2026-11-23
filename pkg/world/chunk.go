package world

import (
	"errors"
	"fmt"
	"math"

	"github.com/HUA503/minecraft-go/pkg/nbt"
)

const (
	MinSectionY = -4
	MaxSectionY = 19
	NumSections = 24 // -4 to 19 inclusive (-64 to 319 height)
	ChunkWidth  = 16
	ChunkHeight = 384
	MinWorldY   = -64
	MaxWorldY   = 319
	// WorldBorderLimit is the vanilla horizontal world border in block coordinates.
	WorldBorderLimit = 29_999_984
)

// ValidBlockCoordinates reports whether a block position is inside the supported
// world height and horizontal world border. Keep this check at mutation and
// network boundaries so malformed coordinates cannot allocate distant chunks.
func ValidBlockCoordinates(x, y, z int32) bool {
	return y >= MinWorldY && y <= MaxWorldY &&
		x >= -WorldBorderLimit && x <= WorldBorderLimit &&
		z >= -WorldBorderLimit && z <= WorldBorderLimit
}

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
	if s == nil || s.BlockStates == nil || localX < 0 || localX >= ChunkWidth || localY < 0 || localY >= ChunkWidth || localZ < 0 || localZ >= ChunkWidth {
		return 0
	}
	idx := (localY << 8) | (localZ << 4) | localX
	return s.BlockStates.Get(idx)
}

func (s *Section) SetBlock(localX, localY, localZ int, state int32) {
	if s == nil || localX < 0 || localX >= ChunkWidth || localY < 0 || localY >= ChunkWidth || localZ < 0 || localZ >= ChunkWidth || state < 0 {
		return
	}
	if s.BlockStates == nil {
		s.BlockStates = NewPalettedContainer(4096, 0)
	}
	idx := (localY << 8) | (localZ << 4) | localX
	s.BlockStates.Set(idx, state)
}

func (s *Section) SetBiome(bx, by, bz int, biomeID int32) {
	if s == nil || bx < 0 || bx >= 4 || by < 0 || by >= 4 || bz < 0 || bz >= 4 || biomeID < 0 {
		return
	}
	idx := (by << 4) | (bz << 2) | bx
	if s.Biomes == nil {
		s.Biomes = NewPalettedContainer(64, 1)
	}
	s.Biomes.Set(idx, biomeID)
}

func (s *Section) GetBiome(bx, by, bz int) int32 {
	if s == nil || bx < 0 || bx >= 4 || by < 0 || by >= 4 || bz < 0 || bz >= 4 {
		return 1
	}
	idx := (by << 4) | (bz << 2) | bx
	if s.Biomes != nil {
		return s.Biomes.Get(idx)
	}
	return 1
}

// Chunk represents a full 16x384x16 vertical column of 24 sections.
type Chunk struct {
	Pos            ChunkPos
	Sections       [NumSections]*Section
	IsDirty        bool
	Revision       uint64
	ModifiedBlocks map[BlockPos]int32
}

func NewChunk(pos ChunkPos) *Chunk {
	c := &Chunk{
		Pos:            pos,
		ModifiedBlocks: make(map[BlockPos]int32),
	}
	for i := 0; i < NumSections; i++ {
		c.Sections[i] = NewSection(i)
	}
	return c
}

// Clone creates a detached chunk snapshot suitable for serialization while
// the live world continues to receive edits.
func (c *Chunk) Clone() *Chunk {
	if c == nil {
		return nil
	}
	clone := &Chunk{
		Pos:      c.Pos,
		IsDirty:  c.IsDirty,
		Revision: c.Revision,
	}
	if len(c.ModifiedBlocks) > 0 {
		clone.ModifiedBlocks = make(map[BlockPos]int32, len(c.ModifiedBlocks))
		for pos, state := range c.ModifiedBlocks {
			clone.ModifiedBlocks[pos] = state
		}
	}
	clonePalette := func(source *PalettedContainer) *PalettedContainer {
		if source == nil {
			return nil
		}
		return &PalettedContainer{
			Capacity: source.Capacity,
			Palette:  append([]int32(nil), source.Palette...),
			Data:     append([]int32(nil), source.Data...),
		}
	}
	for i, section := range c.Sections {
		if section == nil {
			continue
		}
		clone.Sections[i] = &Section{
			YIndex:      section.YIndex,
			BlockStates: clonePalette(section.BlockStates),
			Biomes:      clonePalette(section.Biomes),
		}
	}
	return clone
}

// GetBlock returns the block state ID at absolute world coordinates.
func (c *Chunk) GetBlock(x, y, z int32) int32 {
	if c == nil || !ValidBlockCoordinates(x, y, z) || (BlockPos{X: x, Y: y, Z: z}).ToChunkPos() != c.Pos {
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
	if c == nil || !ValidBlockCoordinates(x, y, z) || (BlockPos{X: x, Y: y, Z: z}).ToChunkPos() != c.Pos || state < 0 {
		return
	}
	secIdx := int(math.Floor(float64(y)/16.0)) - MinSectionY
	if secIdx < 0 || secIdx >= NumSections {
		return
	}
	localX := int(x & 15)
	localY := int(y & 15)
	localZ := int(z & 15)
	if c.Sections[secIdx] == nil {
		c.Sections[secIdx] = NewSection(secIdx)
	}
	c.Sections[secIdx].SetBlock(localX, localY, localZ, state)
	c.IsDirty = true
	c.Revision++
	if c.ModifiedBlocks == nil {
		c.ModifiedBlocks = make(map[BlockPos]int32)
	}
	c.ModifiedBlocks[BlockPos{X: x, Y: y, Z: z}] = state
}

// SetBiome sets the biome ID for the 4x4 vertical column containing (x, z) at section y.
func (c *Chunk) SetBiome(x, y, z int32, biomeID int32) {
	if c == nil || !ValidBlockCoordinates(x, y, z) || (BlockPos{X: x, Y: y, Z: z}).ToChunkPos() != c.Pos || biomeID < 0 {
		return
	}
	secIdx := int(math.Floor(float64(y)/16.0)) - MinSectionY
	if secIdx < 0 || secIdx >= NumSections {
		return
	}
	sec := c.Sections[secIdx]
	if sec == nil {
		sec = NewSection(secIdx)
		c.Sections[secIdx] = sec
	}
	bx := int((x & 15) >> 2)
	bz := int((z & 15) >> 2)
	for by := 0; by < 4; by++ {
		sec.SetBiome(bx, by, bz, biomeID)
	}
	c.IsDirty = true
	c.Revision++
}

// GetBiome returns the biome ID for the 4x4x4 block volume containing (x, y, z).
func (c *Chunk) GetBiome(x, y, z int32) int32 {
	if c == nil || !ValidBlockCoordinates(x, y, z) || (BlockPos{X: x, Y: y, Z: z}).ToChunkPos() != c.Pos {
		return 1 // Plains
	}
	secIdx := int(math.Floor(float64(y)/16.0)) - MinSectionY
	if secIdx < 0 || secIdx >= NumSections {
		return 1
	}
	sec := c.Sections[secIdx]
	if sec == nil {
		return 1
	}
	bx := int((x & 15) >> 2)
	by := int((y & 15) >> 2)
	bz := int((z & 15) >> 2)
	return sec.GetBiome(bx, by, bz)
}

// ToNBT serializes a Chunk to standard Minecraft Anvil NBT format (DataVersion 5023).
func (c *Chunk) ToNBT() *nbt.CompoundTag {
	root := nbt.NewCompound()
	root.Put("DataVersion", &nbt.IntTag{Value: 5023}) // Minecraft Java Edition 26.3 standard
	root.Put("xPos", &nbt.IntTag{Value: c.Pos.X})
	root.Put("yPos", &nbt.IntTag{Value: MinSectionY})
	root.Put("zPos", &nbt.IntTag{Value: c.Pos.Z})
	root.Put("Status", &nbt.StringTag{Value: "minecraft:full"})

	sectionsList := &nbt.ListTag{
		ElementType: nbt.TagCompound,
		Elements:    make([]nbt.Tag, 0, NumSections),
	}

	for i := 0; i < NumSections; i++ {
		sec := c.Sections[i]
		if sec == nil {
			sec = NewSection(i)
			c.Sections[i] = sec
		}
		secTag := nbt.NewCompound()
		secY := int32(i + MinSectionY) // -4 to 19
		secTag.Put("Y", &nbt.ByteTag{Value: int8(secY)})

		// BlockStates
		blockData := make([]int32, 4096)
		hasBlock := false
		for idx := 0; idx < 4096; idx++ {
			b := sec.BlockStates.Get(idx)
			blockData[idx] = b
			if b != 0 {
				hasBlock = true
			}
		}
		if hasBlock {
			secTag.Put("BlockStates", &nbt.IntArrayTag{Value: blockData})
		}

		// Biomes
		biomeData := make([]int32, 64)
		for idx := 0; idx < 64; idx++ {
			biomeData[idx] = sec.Biomes.Get(idx)
		}
		secTag.Put("Biomes", &nbt.IntArrayTag{Value: biomeData})

		sectionsList.Elements = append(sectionsList.Elements, secTag)
	}

	root.Put("sections", sectionsList)

	if len(c.ModifiedBlocks) > 0 {
		modList := &nbt.ListTag{
			ElementType: nbt.TagCompound,
			Elements:    make([]nbt.Tag, 0, len(c.ModifiedBlocks)),
		}
		for pos, state := range c.ModifiedBlocks {
			bt := nbt.NewCompound()
			bt.Put("x", &nbt.IntTag{Value: pos.X})
			bt.Put("y", &nbt.IntTag{Value: pos.Y})
			bt.Put("z", &nbt.IntTag{Value: pos.Z})
			bt.Put("state", &nbt.IntTag{Value: state})
			modList.Elements = append(modList.Elements, bt)
		}
		root.Put("ModifiedBlocks", modList)
	}

	return root
}

// ChunkFromNBT parses a Chunk from NBT tag data.
func ChunkFromNBT(tag *nbt.CompoundTag) (*Chunk, error) {
	if tag == nil {
		return nil, errors.New("nil chunk tag")
	}
	x := tag.GetInt("xPos")
	z := tag.GetInt("zPos")
	c := NewChunk(ChunkPos{X: x, Z: z})

	if secValue := tag.Get("sections"); secValue != nil {
		secList, ok := secValue.(*nbt.ListTag)
		if !ok || (len(secList.Elements) > 0 && secList.ElementType != nbt.TagCompound) {
			return nil, errors.New("invalid chunk sections list")
		}
		seenSections := make(map[int]struct{}, len(secList.Elements))
		for sectionIndex, elem := range secList.Elements {
			secTag, ok := elem.(*nbt.CompoundTag)
			if !ok {
				return nil, fmt.Errorf("invalid chunk section %d", sectionIndex)
			}
			var secY int
			if byteY, ok := secTag.Get("Y").(*nbt.ByteTag); ok {
				secY = int(byteY.Value)
			} else if intY, ok := secTag.Get("Y").(*nbt.IntTag); ok {
				secY = int(intY.Value)
			} else {
				return nil, fmt.Errorf("chunk section %d has invalid Y", sectionIndex)
			}
			secIdx := secY - MinSectionY
			if secIdx < 0 || secIdx >= NumSections {
				return nil, fmt.Errorf("chunk section %d has out-of-range Y %d", sectionIndex, secY)
			}
			if _, duplicate := seenSections[secIdx]; duplicate {
				return nil, fmt.Errorf("chunk contains duplicate section Y %d", secY)
			}
			seenSections[secIdx] = struct{}{}
			sec := c.Sections[secIdx]

			if statesValue := secTag.Get("BlockStates"); statesValue != nil {
				bArr, ok := statesValue.(*nbt.IntArrayTag)
				if !ok || len(bArr.Value) != 4096 {
					return nil, fmt.Errorf("chunk section Y %d has invalid BlockStates", secY)
				}
				for idx, val := range bArr.Value {
					if val < 0 {
						return nil, fmt.Errorf("chunk section Y %d has negative block state", secY)
					}
					sec.BlockStates.Set(idx, val)
				}
			}
			if biomesValue := secTag.Get("Biomes"); biomesValue != nil {
				bArr, ok := biomesValue.(*nbt.IntArrayTag)
				if !ok || len(bArr.Value) != 64 {
					return nil, fmt.Errorf("chunk section Y %d has invalid Biomes", secY)
				}
				for idx, val := range bArr.Value {
					if val < 0 {
						return nil, fmt.Errorf("chunk section Y %d has negative biome ID", secY)
					}
					sec.Biomes.Set(idx, val)
				}
			}
		}
	}

	// Also support backwards-compatible "ModifiedBlocks" list tag
	if modValue := tag.Get("ModifiedBlocks"); modValue != nil {
		modBlocks, ok := modValue.(*nbt.ListTag)
		if !ok || (len(modBlocks.Elements) > 0 && modBlocks.ElementType != nbt.TagCompound) {
			return nil, errors.New("invalid ModifiedBlocks list")
		}
		const maxModifiedBlocks = ChunkWidth * ChunkWidth * ChunkHeight
		if len(modBlocks.Elements) > maxModifiedBlocks {
			return nil, fmt.Errorf("ModifiedBlocks count %d exceeds chunk capacity", len(modBlocks.Elements))
		}
		for i, elem := range modBlocks.Elements {
			if bTag, ok := elem.(*nbt.CompoundTag); ok {
				if _, ok := bTag.Get("x").(*nbt.IntTag); !ok {
					return nil, fmt.Errorf("ModifiedBlocks entry %d has invalid x", i)
				}
				if _, ok := bTag.Get("y").(*nbt.IntTag); !ok {
					return nil, fmt.Errorf("ModifiedBlocks entry %d has invalid y", i)
				}
				if _, ok := bTag.Get("z").(*nbt.IntTag); !ok {
					return nil, fmt.Errorf("ModifiedBlocks entry %d has invalid z", i)
				}
				if _, ok := bTag.Get("state").(*nbt.IntTag); !ok {
					return nil, fmt.Errorf("ModifiedBlocks entry %d has invalid state", i)
				}
				bx := bTag.GetInt("x")
				by := bTag.GetInt("y")
				bz := bTag.GetInt("z")
				state := bTag.GetInt("state")
				pos := BlockPos{X: bx, Y: by, Z: bz}
				if !ValidBlockCoordinates(bx, by, bz) || pos.ToChunkPos() != c.Pos || state < 0 {
					return nil, fmt.Errorf("ModifiedBlocks entry %d has invalid position or state", i)
				}
				c.SetBlock(bx, by, bz, state)
			} else {
				return nil, fmt.Errorf("invalid ModifiedBlocks entry %d", i)
			}
		}
	}

	c.IsDirty = false
	return c, nil
}
