package world

import (
	"math/bits"
)

// PalettedContainer stores indexed data (blocks or biomes) with dynamic palettes.
type PalettedContainer struct {
	Capacity int
	Palette  []int32
	Data     []int32 // Direct unpacked state IDs for fast in-memory mutation
}

func NewPalettedContainer(capacity int, defaultValue int32) *PalettedContainer {
	data := make([]int32, capacity)
	if defaultValue != 0 {
		for i := range data {
			data[i] = defaultValue
		}
	}
	return &PalettedContainer{
		Capacity: capacity,
		Palette:  []int32{defaultValue},
		Data:     data,
	}
}

func (pc *PalettedContainer) Get(index int) int32 {
	if pc == nil || index < 0 || index >= pc.Capacity || index >= len(pc.Data) {
		return 0
	}
	return pc.Data[index]
}

func (pc *PalettedContainer) Set(index int, value int32) {
	if pc == nil || index < 0 || index >= pc.Capacity || index >= len(pc.Data) {
		return
	}
	pc.Data[index] = value
}

// CompactPalette computes unique palette entries and bits per entry.
func (pc *PalettedContainer) CompactPalette() ([]int32, int) {
	if pc == nil || pc.Capacity < 0 || len(pc.Data) < pc.Capacity {
		return []int32{0}, 0
	}
	seen := make(map[int32]struct{})
	var palette []int32
	for _, v := range pc.Data[:pc.Capacity] {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			palette = append(palette, v)
		}
	}
	if len(palette) == 0 {
		palette = []int32{0}
	}
	if len(palette) == 1 {
		return palette, 0 // Single-valued
	}

	bpe := bits.Len(uint(len(palette) - 1))
	if bpe < 4 {
		bpe = 4 // Minimum indirect palette size in Minecraft Java
	}
	return palette, bpe
}

// PackToBitStorage packs the container data into uint64 array given bitsPerEntry.
func (pc *PalettedContainer) PackToBitStorage(palette []int32, bpe int) []uint64 {
	if pc == nil || pc.Capacity <= 0 || len(pc.Data) < pc.Capacity || bpe <= 0 || bpe > 32 || len(palette) == 0 {
		return nil
	}

	paletteIndexMap := make(map[int32]uint64, len(palette))
	for idx, val := range palette {
		paletteIndexMap[val] = uint64(idx)
	}

	entriesPerLong := 64 / bpe
	numLongs := (pc.Capacity + entriesPerLong - 1) / entriesPerLong
	storage := make([]uint64, numLongs)

	for i, val := range pc.Data[:pc.Capacity] {
		longIdx := i / entriesPerLong
		entryIdx := i % entriesPerLong
		shift := entryIdx * bpe
		palIdx := paletteIndexMap[val]
		storage[longIdx] |= (palIdx << shift)
	}
	return storage
}

// UnpackFromBitStorage unpacks uint64 array into container data.
func (pc *PalettedContainer) UnpackFromBitStorage(storage []uint64, palette []int32, bpe int) {
	if pc == nil || pc.Capacity < 0 || len(pc.Data) < pc.Capacity || bpe < 0 || bpe > 32 {
		return
	}
	if bpe == 0 {
		val := int32(0)
		if len(palette) > 0 {
			val = palette[0]
		}
		for i := range pc.Data {
			pc.Data[i] = val
		}
		return
	}
	if len(palette) == 0 {
		clear(pc.Data[:pc.Capacity])
		return
	}

	entriesPerLong := 64 / bpe
	mask := uint64((1 << bpe) - 1)

	for i := 0; i < pc.Capacity; i++ {
		longIdx := i / entriesPerLong
		entryIdx := i % entriesPerLong
		shift := entryIdx * bpe
		if longIdx < len(storage) {
			palIdx := int((storage[longIdx] >> shift) & mask)
			if palIdx < len(palette) {
				pc.Data[i] = palette[palIdx]
			}
		}
	}
}
