package world

import (
	"bytes"
	"errors"
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

func Test26xRedShrubRuntimeIDSurvivesChunkNBT(t *testing.T) {
	chunk := NewChunk(ChunkPos{X: 0, Z: 0})
	chunk.SetBlock(7, 64, 9, 10100)

	reloaded, err := ChunkFromNBT(chunk.ToNBT())
	if err != nil {
		t.Fatalf("decode chunk containing 26.3 red shrub: %v", err)
	}
	if got := reloaded.GetBlock(7, 64, 9); got != 10100 {
		t.Fatalf("red shrub runtime ID after chunk NBT round trip = %d, want 10100", got)
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

func TestOpenRegionInitializesCompleteHeaders(t *testing.T) {
	mcaPath := filepath.Join(t.TempDir(), "r.0.0.mca")
	region, err := OpenRegion(mcaPath)
	if err != nil {
		t.Fatalf("OpenRegion failed: %v", err)
	}
	defer region.Close()

	info, err := os.Stat(mcaPath)
	if err != nil {
		t.Fatalf("stat region file: %v", err)
	}
	if info.Size() != 8192 {
		t.Fatalf("new region file size = %d, want complete 8192-byte header", info.Size())
	}

	data, err := os.ReadFile(mcaPath)
	if err != nil {
		t.Fatalf("read region header: %v", err)
	}
	if len(data) != 8192 {
		t.Fatalf("read header length = %d, want 8192", len(data))
	}
	for i, b := range data {
		if b != 0 {
			t.Fatalf("new region header byte %d = %d, want zero", i, b)
		}
	}
}

type injectedRegionFile struct {
	regionFile
	shortWriteOffset    int64
	shortWriteRemaining int
	failSyncCall        int
	syncCalls           int
	syncErr             error
}

func (f *injectedRegionFile) WriteAt(p []byte, offset int64) (int, error) {
	if offset == f.shortWriteOffset && f.shortWriteRemaining > 0 {
		f.shortWriteRemaining--
		if len(p) <= 1 {
			return 0, nil
		}
		return f.regionFile.WriteAt(p[:len(p)-1], offset)
	}
	return f.regionFile.WriteAt(p, offset)
}

func (f *injectedRegionFile) Sync() error {
	f.syncCalls++
	if f.syncCalls == f.failSyncCall {
		return f.syncErr
	}
	return f.regionFile.Sync()
}

func testChunkTag(status string) *nbt.CompoundTag {
	tag := nbt.NewCompound()
	tag.Put("xPos", &nbt.IntTag{Value: 1})
	tag.Put("zPos", &nbt.IntTag{Value: 1})
	tag.Put("Status", &nbt.StringTag{Value: status})
	return tag
}

func TestRegionRollbackPreservesChunkAfterLocationShortWrite(t *testing.T) {
	region, err := OpenRegion(filepath.Join(t.TempDir(), "r.0.0.mca"))
	if err != nil {
		t.Fatal(err)
	}
	defer region.Close()
	if err := region.WriteChunkData(1, 1, testChunkTag("old")); err != nil {
		t.Fatal(err)
	}
	index := chunkIndex(1, 1)
	oldLocation := make([]byte, 4)
	if _, err := region.file.ReadAt(oldLocation, int64(index*4)); err != nil {
		t.Fatal(err)
	}

	region.file = &injectedRegionFile{
		regionFile:          region.file,
		shortWriteOffset:    int64(index * 4),
		shortWriteRemaining: 1,
	}
	if err := region.WriteChunkData(1, 1, testChunkTag("new")); err == nil {
		t.Fatal("expected short location-header write to fail")
	}
	location := make([]byte, 4)
	if _, err := region.file.ReadAt(location, int64(index*4)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(location, oldLocation) {
		t.Fatalf("location header after rollback = %v, want %v", location, oldLocation)
	}
	got, err := region.ReadChunkData(1, 1)
	if err != nil {
		t.Fatalf("old chunk became unreadable after rollback: %v", err)
	}
	if got.GetString("Status") != "old" {
		t.Fatalf("chunk after rollback has status %q, want old", got.GetString("Status"))
	}
}

func TestRegionRollbackPreservesChunkAfterLocationSyncFailure(t *testing.T) {
	region, err := OpenRegion(filepath.Join(t.TempDir(), "r.0.0.mca"))
	if err != nil {
		t.Fatal(err)
	}
	defer region.Close()
	if err := region.WriteChunkData(1, 1, testChunkTag("old")); err != nil {
		t.Fatal(err)
	}
	index := chunkIndex(1, 1)
	oldLocation := make([]byte, 4)
	if _, err := region.file.ReadAt(oldLocation, int64(index*4)); err != nil {
		t.Fatal(err)
	}
	region.file = &injectedRegionFile{
		regionFile:   region.file,
		failSyncCall: 2, // payload sync succeeds; location-table sync fails
		syncErr:      errors.New("injected location sync failure"),
	}
	if err := region.WriteChunkData(1, 1, testChunkTag("new")); err == nil {
		t.Fatal("expected location-table sync failure")
	}
	location := make([]byte, 4)
	if _, err := region.file.ReadAt(location, int64(index*4)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(location, oldLocation) {
		t.Fatalf("location header after rollback = %v, want %v", location, oldLocation)
	}
	got, err := region.ReadChunkData(1, 1)
	if err != nil {
		t.Fatalf("old chunk became unreadable after rollback: %v", err)
	}
	if got.GetString("Status") != "old" {
		t.Fatalf("chunk after rollback has status %q, want old", got.GetString("Status"))
	}
}

func TestRegionRecoveryRollsBackPreparedHeaderAfterCrash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.0.0.mca")
	region, err := OpenRegion(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := region.WriteChunkData(1, 1, testChunkTag("old")); err != nil {
		t.Fatal(err)
	}
	index := chunkIndex(1, 1)
	txn := regionTransaction{index: index}
	if _, err := region.file.ReadAt(txn.oldLocation[:], int64(index*4)); err != nil {
		t.Fatal(err)
	}
	if _, err := region.file.ReadAt(txn.oldTimestamp[:], int64(4096+index*4)); err != nil {
		t.Fatal(err)
	}
	txn.newLocation = [4]byte{0, 0, 50, 1}
	txn.newTimestamp = [4]byte{1, 2, 3, 4}
	if err := writeRegionJournal(path+".txn", regionJournalPrepared, txn); err != nil {
		t.Fatal(err)
	}
	if err := writeAtFull(region.file, txn.newLocation[:], int64(index*4)); err != nil {
		t.Fatal(err)
	}
	if err := writeAtFull(region.file, txn.newTimestamp[:], int64(4096+index*4)); err != nil {
		t.Fatal(err)
	}
	if err := region.file.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := region.file.Close(); err != nil {
		t.Fatal(err)
	}
	region.file = nil

	recovered, err := OpenRegion(path)
	if err != nil {
		t.Fatalf("reopen and recover prepared transaction: %v", err)
	}
	defer recovered.Close()
	got, err := recovered.ReadChunkData(1, 1)
	if err != nil {
		t.Fatalf("read chunk after rollback recovery: %v", err)
	}
	if status := got.GetString("Status"); status != "old" {
		t.Fatalf("recovered chunk status = %q, want old", status)
	}
	location := make([]byte, 4)
	if _, err := recovered.file.ReadAt(location, int64(index*4)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(location, txn.oldLocation[:]) {
		t.Fatalf("recovered location = %v, want old header %v", location, txn.oldLocation)
	}
}

func TestRegionRecoveryReplaysCommittedHeaderAfterCrash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.0.0.mca")
	region, err := OpenRegion(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := region.WriteChunkData(1, 1, testChunkTag("old")); err != nil {
		t.Fatal(err)
	}
	if err := region.WriteChunkData(2, 2, testChunkTag("new")); err != nil {
		t.Fatal(err)
	}
	targetIndex, sourceIndex := chunkIndex(1, 1), chunkIndex(2, 2)
	txn := regionTransaction{index: targetIndex}
	if _, err := region.file.ReadAt(txn.oldLocation[:], int64(targetIndex*4)); err != nil {
		t.Fatal(err)
	}
	if _, err := region.file.ReadAt(txn.oldTimestamp[:], int64(4096+targetIndex*4)); err != nil {
		t.Fatal(err)
	}
	if _, err := region.file.ReadAt(txn.newLocation[:], int64(sourceIndex*4)); err != nil {
		t.Fatal(err)
	}
	txn.newTimestamp = [4]byte{5, 6, 7, 8}
	if err := writeRegionJournal(path+".txn", regionJournalPrepared, txn); err != nil {
		t.Fatal(err)
	}
	if err := writeRegionJournal(path+".txn", regionJournalCommitted, txn); err != nil {
		t.Fatal(err)
	}
	if err := writeAtFull(region.file, make([]byte, 4), int64(sourceIndex*4)); err != nil {
		t.Fatal(err)
	}
	if err := region.file.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := region.file.Close(); err != nil {
		t.Fatal(err)
	}
	region.file = nil

	recovered, err := OpenRegion(path)
	if err != nil {
		t.Fatalf("reopen and replay committed transaction: %v", err)
	}
	defer recovered.Close()
	got, err := recovered.ReadChunkData(1, 1)
	if err != nil {
		t.Fatalf("read chunk after commit recovery: %v", err)
	}
	if status := got.GetString("Status"); status != "new" {
		t.Fatalf("recovered chunk status = %q, want new", status)
	}
}

func TestRegionWriteRecoversRealInterruptedCommitStages(t *testing.T) {
	tests := []struct {
		stage      string
		wantStatus string
	}{
		{stage: "prepared", wantStatus: "old"},
		{stage: "header", wantStatus: "old"},
		{stage: "committed", wantStatus: "new"},
	}
	for _, test := range tests {
		t.Run(test.stage, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "r.0.0.mca")
			region, err := OpenRegion(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := region.WriteChunkData(1, 1, testChunkTag("old")); err != nil {
				t.Fatal(err)
			}
			region.crashHook = func(stage string) error {
				if stage == test.stage {
					return errors.New("simulated process termination")
				}
				return nil
			}
			if err := region.WriteChunkData(1, 1, testChunkTag("new")); err == nil {
				t.Fatal("expected simulated interruption")
			}
			// A real process exit bypasses Region.Close's recovery path.
			if err := region.file.Close(); err != nil {
				t.Fatal(err)
			}
			region.file = nil

			recovered, err := OpenRegion(path)
			if err != nil {
				t.Fatalf("reopen after interruption at %q: %v", test.stage, err)
			}
			defer recovered.Close()
			got, err := recovered.ReadChunkData(1, 1)
			if err != nil {
				t.Fatalf("read chunk after recovery: %v", err)
			}
			if status := got.GetString("Status"); status != test.wantStatus {
				t.Fatalf("recovered status = %q, want %q", status, test.wantStatus)
			}
			if _, err := os.Stat(path + ".txn"); !os.IsNotExist(err) {
				t.Fatalf("transaction journal still exists after recovery: %v", err)
			}
		})
	}
}

func TestRegionOpenPreservesInvalidFullJournalInsteadOfIgnoringIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.0.0.mca")
	region, err := OpenRegion(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := region.WriteChunkData(1, 1, testChunkTag("preserve")); err != nil {
		t.Fatal(err)
	}
	if err := region.file.Close(); err != nil {
		t.Fatal(err)
	}
	region.file = nil

	journal := encodeRegionJournalRecord(regionJournalPrepared, regionTransaction{index: chunkIndex(1, 1)})
	journal[8] ^= 0xFF // invalidate the checksum after a complete record exists
	if err := os.WriteFile(path+".txn", journal, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenRegion(path); err == nil {
		t.Fatal("opening a region with an ambiguous full transaction record should fail")
	}
	if _, err := os.Stat(path + ".txn"); err != nil {
		t.Fatalf("invalid journal was removed instead of preserved: %v", err)
	}
}

func TestSaveChunkKeepsDirtyFlagWhenRegionCommitFails(t *testing.T) {
	sm, err := NewSaveManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sm.Close()

	cp := ChunkPos{X: 1, Z: 1}
	chunk := NewChunk(cp)
	chunk.SetBlock(17, 64, 17, 1)
	if err := sm.SaveChunk(chunk); err != nil {
		t.Fatalf("initial SaveChunk: %v", err)
	}
	if chunk.IsDirty {
		t.Fatal("successful save left chunk dirty")
	}
	chunk.SetBlock(17, 64, 17, 2)
	if !chunk.IsDirty {
		t.Fatal("block change did not mark chunk dirty")
	}

	region := sm.regions["r.0.0.mca"]
	region.file = &injectedRegionFile{
		regionFile:          region.file,
		shortWriteOffset:    int64(chunkIndex(cp.X, cp.Z) * 4),
		shortWriteRemaining: 1,
	}
	if err := sm.SaveChunk(chunk); err == nil {
		t.Fatal("expected region commit failure")
	}
	if !chunk.IsDirty {
		t.Fatal("failed save cleared chunk dirty flag")
	}

	saved, err := sm.LoadChunk(cp)
	if err != nil {
		t.Fatalf("load previously committed chunk after failed update: %v", err)
	}
	if got := saved.GetBlock(17, 64, 17); got != 1 {
		t.Fatalf("failed update changed persisted block to %d, want old value 1", got)
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
			X:          12.5,
			Y:          65.0,
			Z:          -18.3,
			Yaw:        45.0,
			Pitch:      10.0,
			Health:     18.5,
			Hunger:     16,
			Saturation: 3.25,
			Absorption: 4,
			Effects:    []PlayerStatusEffect{{ID: "minecraft:regeneration", DurationTicks: 80, Amplifier: 1}},
			GameMode:   0,
			Inventory: []SavedItemStack{
				{Slot: 0, ItemID: 270, Count: 1, Damage: 5},
				{Slot: 1, ItemID: 4, Count: 64, Damage: 0},
				{Slot: 9, ItemID: 265, Count: 12, Damage: 0},
				{Slot: PlayerOffhandInventorySlot, ItemID: 50, Count: 8, Damage: 0},
			},
			Armor: []SavedItemStack{
				{Slot: 0, ItemID: 313, Count: 1, Damage: 7},
				{Slot: 3, ItemID: 310, Count: 1, Damage: 2},
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
	if loaded.Player.Hunger != original.Player.Hunger || loaded.Player.Saturation != original.Player.Saturation || loaded.Player.Absorption != original.Player.Absorption {
		t.Errorf("Player food/absorption round trip mismatch: hunger=%d saturation=%v absorption=%v", loaded.Player.Hunger, loaded.Player.Saturation, loaded.Player.Absorption)
	}
	if len(loaded.Player.Effects) != 1 || loaded.Player.Effects[0] != original.Player.Effects[0] {
		t.Errorf("Player active effects round trip mismatch: %+v", loaded.Player.Effects)
	}
	if len(loaded.Player.Inventory) != len(original.Player.Inventory) {
		t.Fatalf("Player inventory length mismatch: got %d, want %d", len(loaded.Player.Inventory), len(original.Player.Inventory))
	}
	if loaded.Player.Inventory[2].ItemID != 265 || loaded.Player.Inventory[2].Count != 12 {
		t.Errorf("Player inventory item mismatch: %+v", loaded.Player.Inventory[2])
	}
	if got := loaded.Player.Inventory[3]; got.Slot != PlayerOffhandInventorySlot || got.ItemID != 50 || got.Count != 8 {
		t.Errorf("Player offhand round trip mismatch: %+v", got)
	}
	if len(loaded.Player.Armor) != 2 || loaded.Player.Armor[0].Slot != 0 || loaded.Player.Armor[0].ItemID != 313 || loaded.Player.Armor[0].Damage != 7 || loaded.Player.Armor[1].Slot != 3 || loaded.Player.Armor[1].ItemID != 310 {
		t.Errorf("Player armor round trip mismatch: %+v", loaded.Player.Armor)
	}
}
