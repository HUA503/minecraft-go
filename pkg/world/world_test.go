package world

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HUA503/minecraft-go/pkg/nbt"
)

func TestChunkBlockAccess(t *testing.T) {
	c := NewChunk(ChunkPos{X: 0, Z: 0})

	// Test deepslate layer at Y = -50
	c.SetBlock(5, -50, 7, 100)
	if val := c.GetBlock(5, -50, 7); val != 100 {
		t.Fatalf("deepslate block mismatch: got %d, expected 100", val)
	}

	// Test surface layer at Y = 64
	c.SetBlock(10, 64, 12, 2) // Grass
	if val := c.GetBlock(10, 64, 12); val != 2 {
		t.Fatalf("surface block mismatch: got %d, expected 2", val)
	}

	// Test sky build limit at Y = 319
	c.SetBlock(0, 319, 0, 1) // Stone
	if val := c.GetBlock(0, 319, 0); val != 1 {
		t.Fatalf("build limit block mismatch: got %d, expected 1", val)
	}

	// Test out-of-bounds Y
	if val := c.GetBlock(0, -65, 0); val != 0 {
		t.Fatalf("out of bounds below should be air (0): got %d", val)
	}
	if val := c.GetBlock(0, 320, 0); val != 0 {
		t.Fatalf("out of bounds above should be air (0): got %d", val)
	}
}

func TestPalettedContainerPacking(t *testing.T) {
	pc := NewPalettedContainer(4096, 0)
	pc.Set(0, 1)
	pc.Set(100, 2)
	pc.Set(500, 3)
	pc.Set(4095, 1)

	palette, bpe := pc.CompactPalette()
	if len(palette) < 4 {
		t.Fatalf("expected at least 4 unique entries, got %d", len(palette))
	}
	if bpe < 4 {
		t.Fatalf("expected bpe >= 4, got %d", bpe)
	}

	packed := pc.PackToBitStorage(palette, bpe)
	if len(packed) == 0 {
		t.Fatalf("packed storage is empty")
	}

	pc2 := NewPalettedContainer(4096, 0)
	pc2.UnpackFromBitStorage(packed, palette, bpe)

	if pc2.Get(0) != 1 || pc2.Get(100) != 2 || pc2.Get(500) != 3 || pc2.Get(4095) != 1 {
		t.Fatalf("unpacked values mismatch: [0]=%d, [100]=%d, [500]=%d, [4095]=%d",
			pc2.Get(0), pc2.Get(100), pc2.Get(500), pc2.Get(4095))
	}
}

func TestRegionFileRoundTrip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "region_test")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mcaPath := filepath.Join(tmpDir, "r.0.0.mca")
	region, err := OpenRegion(mcaPath)
	if err != nil {
		t.Fatalf("OpenRegion failed: %v", err)
	}
	defer region.Close()

	// Create test chunk NBT
	chunkNBT := nbt.NewCompound()
	chunkNBT.Put("DataVersion", &nbt.IntTag{Value: 3953})
	chunkNBT.Put("xPos", &nbt.IntTag{Value: 5})
	chunkNBT.Put("zPos", &nbt.IntTag{Value: 8})
	chunkNBT.Put("Status", &nbt.StringTag{Value: "minecraft:full"})

	if err := region.WriteChunkData(5, 8, chunkNBT); err != nil {
		t.Fatalf("WriteChunkData failed: %v", err)
	}

	// Read back chunk
	readNBT, err := region.ReadChunkData(5, 8)
	if err != nil {
		t.Fatalf("ReadChunkData failed: %v", err)
	}

	if readNBT.GetInt("DataVersion") != 3953 {
		t.Fatalf("DataVersion mismatch: got %d", readNBT.GetInt("DataVersion"))
	}
	if readNBT.GetString("Status") != "minecraft:full" {
		t.Fatalf("Status mismatch: got %s", readNBT.GetString("Status"))
	}
}
