package world

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/HUA503/minecraft-go/pkg/nbt"
)

// SavedItemStack holds an item stack for persistent storage.
type SavedItemStack struct {
	Slot   int
	ItemID int32
	Count  int
	Damage int
}

// PlayerSaveData holds persistent player information matching vanilla Minecraft level.dat.
type PlayerSaveData struct {
	X, Y, Z    float64
	Yaw, Pitch float32
	Health     float32
	Hunger     int
	GameMode   int // 0: Survival, 1: Creative
	Inventory  []SavedItemStack
}

// LevelData holds top-level world metadata.
type LevelData struct {
	Seed      int64
	Time      int64
	DayTime   int64
	SpawnX    int32
	SpawnY    int32
	SpawnZ    int32
	LevelName string
	GameMode  int // 0: Survival, 1: Creative
	Player    *PlayerSaveData
}

// SaveManager coordinates persistence for chunks and world level data in Anvil MCA format.
type SaveManager struct {
	worldDir  string
	regionDir string
	regions   map[string]*Region
	mu        sync.Mutex
}

func NewSaveManager(worldDir string) (*SaveManager, error) {
	return NewDimensionSaveManager(worldDir, 0)
}

// NewDimensionSaveManager creates a SaveManager rooted in the canonical Minecraft dimension subfolder:
// dim 0: <worldDir>/region
// dim 1: <worldDir>/DIM-1/region (Nether)
// dim 2: <worldDir>/DIM1/region  (The End)
func NewDimensionSaveManager(worldDir string, dim int) (*SaveManager, error) {
	var regionDir string
	switch dim {
	case 1:
		regionDir = filepath.Join(worldDir, "DIM-1", "region")
	case 2:
		regionDir = filepath.Join(worldDir, "DIM1", "region")
	default:
		regionDir = filepath.Join(worldDir, "region")
	}
	if err := os.MkdirAll(regionDir, 0755); err != nil {
		return nil, err
	}
	return &SaveManager{
		worldDir:  worldDir,
		regionDir: regionDir,
		regions:   make(map[string]*Region),
	}, nil
}

// GetDimensionStorage returns a SaveManager targeting the specified dimension.
func (sm *SaveManager) GetDimensionStorage(dim int) *SaveManager {
	if dim == 0 && sm.regionDir == filepath.Join(sm.worldDir, "region") {
		return sm
	}
	dsm, err := NewDimensionSaveManager(sm.worldDir, dim)
	if err != nil {
		return sm
	}
	return dsm
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
	dataTag.Put("GameType", &nbt.IntTag{Value: int32(data.GameMode)})

	if data.Player != nil {
		pTag := nbt.NewCompound()
		posList := &nbt.ListTag{ElementType: nbt.TagDouble}
		posList.Elements = []nbt.Tag{
			&nbt.DoubleTag{Value: data.Player.X},
			&nbt.DoubleTag{Value: data.Player.Y},
			&nbt.DoubleTag{Value: data.Player.Z},
		}
		pTag.Put("Pos", posList)

		rotList := &nbt.ListTag{ElementType: nbt.TagFloat}
		rotList.Elements = []nbt.Tag{
			&nbt.FloatTag{Value: data.Player.Yaw},
			&nbt.FloatTag{Value: data.Player.Pitch},
		}
		pTag.Put("Rotation", rotList)
		pTag.Put("Health", &nbt.FloatTag{Value: data.Player.Health})
		pTag.Put("foodLevel", &nbt.IntTag{Value: int32(data.Player.Hunger)})
		pTag.Put("playerGameType", &nbt.IntTag{Value: int32(data.Player.GameMode)})

		invList := &nbt.ListTag{ElementType: nbt.TagCompound}
		for _, it := range data.Player.Inventory {
			if it.Count > 0 && it.ItemID > 0 {
				itemTag := nbt.NewCompound()
				itemTag.Put("Slot", &nbt.ByteTag{Value: int8(it.Slot)})
				itemTag.Put("id", &nbt.IntTag{Value: it.ItemID})
				itemTag.Put("Count", &nbt.ByteTag{Value: int8(it.Count)})
				itemTag.Put("Damage", &nbt.ShortTag{Value: int16(it.Damage)})
				invList.Elements = append(invList.Elements, itemTag)
			}
		}
		pTag.Put("Inventory", invList)
		dataTag.Put("Player", pTag)
	}

	root.Put("Data", dataTag)

	datPath := filepath.Join(sm.worldDir, "level.dat")
	f, err := os.CreateTemp(sm.worldDir, "level.dat.tmp-*")
	if err != nil {
		return err
	}
	tmpPath := f.Name()
	defer os.Remove(tmpPath)
	if err := nbt.WriteGzipped(f, "", root); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, datPath); err != nil {
		return fmt.Errorf("replace level.dat: %w", err)
	}
	return nil
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

	res := &LevelData{
		Seed:      dataTag.GetLong("RandomSeed"),
		Time:      dataTag.GetLong("Time"),
		DayTime:   dataTag.GetLong("DayTime"),
		SpawnX:    dataTag.GetInt("SpawnX"),
		SpawnY:    dataTag.GetInt("SpawnY"),
		SpawnZ:    dataTag.GetInt("SpawnZ"),
		LevelName: dataTag.GetString("LevelName"),
		GameMode:  int(dataTag.GetInt("GameType")),
	}

	if pTag, ok := dataTag.Get("Player").(*nbt.CompoundTag); ok && pTag != nil {
		pd := &PlayerSaveData{
			Health:   20.0,
			Hunger:   20,
			GameMode: int(pTag.GetInt("playerGameType")),
		}
		if hTag, hasH := pTag.Get("Health").(*nbt.FloatTag); hasH {
			pd.Health = hTag.Value
		}
		if fTag, hasF := pTag.Get("foodLevel").(*nbt.IntTag); hasF {
			pd.Hunger = int(fTag.Value)
		}

		if posList, ok := pTag.Get("Pos").(*nbt.ListTag); ok && len(posList.Elements) >= 3 {
			if d0, ok := posList.Elements[0].(*nbt.DoubleTag); ok {
				pd.X = d0.Value
			}
			if d1, ok := posList.Elements[1].(*nbt.DoubleTag); ok {
				pd.Y = d1.Value
			}
			if d2, ok := posList.Elements[2].(*nbt.DoubleTag); ok {
				pd.Z = d2.Value
			}
		}

		if rotList, ok := pTag.Get("Rotation").(*nbt.ListTag); ok && len(rotList.Elements) >= 2 {
			if r0, ok := rotList.Elements[0].(*nbt.FloatTag); ok {
				pd.Yaw = r0.Value
			}
			if r1, ok := rotList.Elements[1].(*nbt.FloatTag); ok {
				pd.Pitch = r1.Value
			}
		}

		if invList, ok := pTag.Get("Inventory").(*nbt.ListTag); ok {
			for _, itemRaw := range invList.Elements {
				if itemComp, ok := itemRaw.(*nbt.CompoundTag); ok {
					slot := int(itemComp.GetByte("Slot"))
					id := itemComp.GetInt("id")
					cnt := int(itemComp.GetByte("Count"))
					dmg := int(itemComp.GetShort("Damage"))
					pd.Inventory = append(pd.Inventory, SavedItemStack{
						Slot:   slot,
						ItemID: id,
						Count:  cnt,
						Damage: dmg,
					})
				}
			}
		}
		res.Player = pd
	}

	return res, nil
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
