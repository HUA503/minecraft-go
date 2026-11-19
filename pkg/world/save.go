package world

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/HUA503/minecraft-go/pkg/nbt"
)

// LevelData holds top-level world metadata.
type LevelData struct {
	Seed      int64
	Time      int64
	DayTime   int64
	SpawnX    int32
	SpawnY    int32
	SpawnZ    int32
	LevelName string
}

// SaveManager coordinates persistence for chunks and world level data in Anvil MCA format.
type SaveManager struct {
	worldDir  string
	regionDir string
	regions   map[string]*Region
	mu        sync.Mutex
}

func NewSaveManager(worldDir string) (*SaveManager, error) {
	regionDir := filepath.Join(worldDir, "region")
	if err := os.MkdirAll(regionDir, 0755); err != nil {
		return nil, err
	}
	return &SaveManager{
		worldDir:  worldDir,
		regionDir: regionDir,
		regions:   make(map[string]*Region),
	}, nil
}

// getRegion returns or opens the Region file for the given chunk coordinates.
func (sm *SaveManager) getRegion(chunkX, chunkZ int32) (*Region, error) {
	regX := int32(math.Floor(float64(chunkX) / 32.0))
	regZ := int32(math.Floor(float64(chunkZ) / 32.0))
	key := fmt.Sprintf("r.%d.%d.mca", regX, regZ)

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if reg, exists := sm.regions[key]; exists {
		return reg, nil
	}

	path := filepath.Join(sm.regionDir, key)
	reg, err := OpenRegion(path)
	if err != nil {
		return nil, err
	}
	sm.regions[key] = reg
	return reg, nil
}

// SaveChunk writes a dirty chunk to its corresponding region file.
func (sm *SaveManager) SaveChunk(chunk *Chunk) error {
	if chunk == nil {
		return nil
	}
	reg, err := sm.getRegion(chunk.Pos.X, chunk.Pos.Z)
	if err != nil {
		return err
	}
	if err := reg.WriteChunkData(chunk.Pos.X, chunk.Pos.Z, chunk.ToNBT()); err != nil {
		return err
	}
	chunk.IsDirty = false
	return nil
}

// LoadChunk reads a chunk from its region file if it exists.
func (sm *SaveManager) LoadChunk(cp ChunkPos) (*Chunk, error) {
	reg, err := sm.getRegion(cp.X, cp.Z)
	if err != nil {
		return nil, err
	}
	tag, err := reg.ReadChunkData(cp.X, cp.Z)
	if err != nil {
		return nil, err
	}
	return ChunkFromNBT(tag)
}

// SaveLevel writes level.dat with world seed, spawn point, and time.
func (sm *SaveManager) SaveLevel(data *LevelData) error {
	if data == nil {
		return nil
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()

	root := nbt.NewCompound()
	dataTag := nbt.NewCompound()
	dataTag.Put("DataVersion", &nbt.IntTag{Value: 5023})
	dataTag.Put("RandomSeed", &nbt.LongTag{Value: data.Seed})
	dataTag.Put("Time", &nbt.LongTag{Value: data.Time})
	dataTag.Put("DayTime", &nbt.LongTag{Value: data.DayTime})
	dataTag.Put("SpawnX", &nbt.IntTag{Value: data.SpawnX})
	dataTag.Put("SpawnY", &nbt.IntTag{Value: data.SpawnY})
	dataTag.Put("SpawnZ", &nbt.IntTag{Value: data.SpawnZ})
	dataTag.Put("LevelName", &nbt.StringTag{Value: data.LevelName})
	root.Put("Data", dataTag)

	datPath := filepath.Join(sm.worldDir, "level.dat")
	f, err := os.Create(datPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return nbt.WriteGzipped(f, "", root)
}

// LoadLevel reads level.dat if present.
func (sm *SaveManager) LoadLevel() (*LevelData, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	datPath := filepath.Join(sm.worldDir, "level.dat")
	f, err := os.Open(datPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, root, err := nbt.ReadGzipped(f)
	if err != nil {
		return nil, err
	}
	dataTag, ok := root.Get("Data").(*nbt.CompoundTag)
	if !ok || dataTag == nil {
		return nil, fmt.Errorf("invalid level.dat: missing Data compound")
	}

	return &LevelData{
		Seed:      dataTag.GetLong("RandomSeed"),
		Time:      dataTag.GetLong("Time"),
		DayTime:   dataTag.GetLong("DayTime"),
		SpawnX:    dataTag.GetInt("SpawnX"),
		SpawnY:    dataTag.GetInt("SpawnY"),
		SpawnZ:    dataTag.GetInt("SpawnZ"),
		LevelName: dataTag.GetString("LevelName"),
	}, nil
}

// Close closes all open region files.
func (sm *SaveManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var firstErr error
	for _, reg := range sm.regions {
		if err := reg.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	sm.regions = make(map[string]*Region)
	return firstErr
}
