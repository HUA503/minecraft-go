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

func TestRegionInPlaceOverwriteNoBloat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "region_bloat_test")
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

	chunkNBT := nbt.NewCompound()
	chunkNBT.Put("DataVersion", &nbt.IntTag{Value: 5023})
	chunkNBT.Put("xPos", &nbt.IntTag{Value: 1})
	chunkNBT.Put("zPos", &nbt.IntTag{Value: 1})
	chunkNBT.Put("Status", &nbt.StringTag{Value: "minecraft:full"})

	// First write
	if err := region.WriteChunkData(1, 1, chunkNBT); err != nil {
		t.Fatalf("WriteChunkData 1 failed: %v", err)
	}

	fi, err := os.Stat(mcaPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	size1 := fi.Size()

	// Write 2nd, 3rd, and 4th times
	for i := 0; i < 3; i++ {
		if err := region.WriteChunkData(1, 1, chunkNBT); err != nil {
			t.Fatalf("WriteChunkData %d failed: %v", i+2, err)
		}
	}

	fi2, err := os.Stat(mcaPath)
	if err != nil {
		t.Fatalf("Stat 2 failed: %v", err)
	}
	sizeFinal := fi2.Size()

	if sizeFinal != size1 {
		t.Fatalf("file size bloated on repeated chunk writes! size1=%d, sizeFinal=%d", size1, sizeFinal)
	}
}

func TestSaveAndLoadLevelWithPlayer(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "level_save_test")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm, err := NewSaveManager(tmpDir)
	if err != nil {
		t.Fatalf("NewSaveManager failed: %v", err)
	}
	defer sm.Close()

	original := &LevelData{
		Seed:      123456789,
		Time:      5000,
		DayTime:   1200,
		SpawnX:    10,
		SpawnY:    64,
		SpawnZ:    -20,
		LevelName: "Test World",
		GameMode:  0,
		Player: &PlayerSaveData{
			X:        12.5,
			Y:        65.0,
			Z:        -18.3,
			Yaw:      45.0,
			Pitch:    10.0,
			Health:   18.5,
			Hunger:   16,
			GameMode: 0,
			Inventory: []SavedItemStack{
				{Slot: 0, ItemID: 270, Count: 1, Damage: 5},
				{Slot: 1, ItemID: 4, Count: 64, Damage: 0},
				{Slot: 9, ItemID: 265, Count: 12, Damage: 0},
			},
		},
	}

	if err := sm.SaveLevel(original); err != nil {
		t.Fatalf("SaveLevel failed: %v", err)
	}

	loaded, err := sm.LoadLevel()
	if err != nil {
		t.Fatalf("LoadLevel failed: %v", err)
	}

	if loaded.Seed != original.Seed {
		t.Errorf("Seed mismatch: got %d, want %d", loaded.Seed, original.Seed)
	}
	if loaded.LevelName != original.LevelName {
		t.Errorf("LevelName mismatch: got %s, want %s", loaded.LevelName, original.LevelName)
	}
	if loaded.Player == nil {
		t.Fatalf("Player data was not loaded")
	}
	if loaded.Player.X != original.Player.X || loaded.Player.Health != original.Player.Health {
		t.Errorf("Player pos/health mismatch: got X=%.1f, Health=%.1f", loaded.Player.X, loaded.Player.Health)
	}
	if len(loaded.Player.Inventory) != len(original.Player.Inventory) {
		t.Fatalf("Player inventory length mismatch: got %d, want %d", len(loaded.Player.Inventory), len(original.Player.Inventory))
	}
	if loaded.Player.Inventory[2].ItemID != 265 || loaded.Player.Inventory[2].Count != 12 {
		t.Errorf("Player inventory item mismatch: %+v", loaded.Player.Inventory[2])
	}
}
