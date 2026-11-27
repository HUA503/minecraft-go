package sim

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

// PlayerEntity represents an active player within the simulation.
type PlayerEntity struct {
	ID             int32
	Username       string
	UUID           [16]byte
	Position       Vec3
	SpawnPos       Vec3
	Velocity       Vec3
	Yaw            float32
	Pitch          float32
	OnGround       bool
	Health         float32 // 0.0 to 20.0
	Hunger         int     // 0 to 20
	Saturation     float32 // 0.0 to 20.0 (canonical vanilla saturation)
	Absorption     float32
	ActiveEffects  [16]world.PlayerStatusEffect
	EffectCount    int
	Exhaustion     float32 // 0.0 to 4.0
	RegenTimer     int     // Tick counter for natural regeneration
	StarveTimer    int     // Tick counter for starvation damage
	Level          int     // Experience level (e.g. 0, 1, 30)
	ExpProgress    float32 // Progress to next level: 0.0 to 1.0
	ExpTotal       int     // Total experience points collected in current level
	FallDistance   float32
	HurtTimer      int // Damage immunity cooldown (10 ticks / 0.5s in Minecraft Java Edition)
	IsAlive        bool
	ClientDriven   bool
	GameMode       int // 0: Survival, 1: Creative, 2: Adventure, 3: Spectator
	Inventory      *inventory.PlayerInventory
	ArmorDefense   int
	ArmorToughness float32
}

// AddExp adds canonical Minecraft experience points and levels up accordingly.
func (p *PlayerEntity) AddExp(amount int) {
	if amount <= 0 {
		return
	}
	p.ExpTotal += amount
	for {
		cap := p.ExpBarCap()
		if p.ExpTotal < cap {
			p.ExpProgress = float32(p.ExpTotal) / float32(cap)
			break
		}
		p.ExpTotal -= cap
		p.Level++
	}
}

// ExpBarCap returns the points required to advance from the current level to the next.
func (p *PlayerEntity) ExpBarCap() int {
	if p.Level >= 30 {
		return 112 + (p.Level-30)*9
	}
	if p.Level >= 15 {
		return 37 + (p.Level-15)*5
	}
	return 7 + p.Level*2
}

// EndCrystal represents an active healing crystal atop an obsidian tower.
type EndCrystal struct {
	ID         int32
	Position   Vec3
	IsAlive    bool
	BeamTarget Vec3
}

// EnderDragon represents the authoritative End dimension boss entity.
type EnderDragon struct {
	Position         Vec3
	Velocity         Vec3
	Yaw              float32
	Pitch            float32
	Health           float32 // 0.0 to 200.0 (100 hearts)
	MaxHealth        float32 // 200.0
	IsAlive          bool
	IsDying          bool
	DeathTicks       int
	Phase            int // 0: circling, 1: dive attack, 2: perched at exit portal
	TargetPos        Vec3
	HealingCrystalID int32
	WingAngle        float32
	DiveCooldown     int
}

// ItemDrop represents a 3D physical item entity in the world dropped from mined blocks or mobs.
type ItemDrop struct {
	ID          int32
	ItemID      int32
	Count       int
	Position    Vec3
	Velocity    Vec3
	Age         float64
	PickupDelay float64
}

// WorldEvent represents a simulation event to broadcast to connected clients.
type WorldEvent struct {
	Type     string // "block_update", "player_move", "player_join", "player_leave", "player_death", "player_respawn", "item_pickup", "dimension_change"
	PlayerID int32
	BlockPos world.BlockPos
	BlockID  int32
	Position Vec3
	Yaw      float32
	Pitch    float32
	Health   float32
}

// Engine manages the authoritative 20 TPS game simulation.
type Engine struct {
	mu                   sync.RWMutex
	saveMu               sync.Mutex
	mobMu                sync.Mutex
	World                map[world.ChunkPos]*world.Chunk
	DimensionWorlds      map[int]map[world.ChunkPos]*world.Chunk
	CurrentDimension     int // 0: Overworld, 1: Nether, 2: End
	Players              map[int32]*PlayerEntity
	clientPlayers        map[int32]PlayerEntity
	nextPlayerID         int32
	Mobs                 map[int32]*MobEntity
	nextMobID            int32
	ItemDrops            map[int32]*ItemDrop
	nextItemDropID       int32
	EnderDragon          *EnderDragon
	EndCrystals          map[int32]*EndCrystal
	nextCrystalID        int32
	TickCount            int64
	WorldTime            int64
	randomTickState      uint64
	Events               chan WorldEvent
	stopChan             chan struct{}
	loopMu               sync.Mutex
	tickDone             chan struct{}
	tickLoopStarted      bool
	stopped              bool
	stopOnce             sync.Once
	Storage              *world.SaveManager
	DimensionStorage     [3]*world.SaveManager
	Furnaces             map[world.BlockPos]*inventory.FurnaceBlockEntity
	buttonReleaseAt      map[world.BlockPos]int64
	pressurePlatePowered map[world.BlockPos]struct{}
	IsPaused             bool
	Difficulty           int // 0: Peaceful, 1: Easy, 2: Normal, 3: Hard
}

func NewEngine() *Engine {
	eng := &Engine{
		World:                make(map[world.ChunkPos]*world.Chunk),
		DimensionWorlds:      make(map[int]map[world.ChunkPos]*world.Chunk),
		CurrentDimension:     0,
		Players:              make(map[int32]*PlayerEntity),
		clientPlayers:        make(map[int32]PlayerEntity),
		nextPlayerID:         1,
		Mobs:                 make(map[int32]*MobEntity),
		nextMobID:            100,
		ItemDrops:            make(map[int32]*ItemDrop),
		nextItemDropID:       1,
		EndCrystals:          make(map[int32]*EndCrystal),
		nextCrystalID:        1,
		WorldTime:            6000, // Canonical Minecraft noon daylight
		randomTickState:      uint64(time.Now().UnixNano()) | 1,
		Difficulty:           2, // Default: Normal difficulty
		Events:               make(chan WorldEvent, 1024),
		stopChan:             make(chan struct{}),
		tickDone:             make(chan struct{}),
		Furnaces:             make(map[world.BlockPos]*inventory.FurnaceBlockEntity),
		buttonReleaseAt:      make(map[world.BlockPos]int64),
		pressurePlatePowered: make(map[world.BlockPos]struct{}),
	}
	eng.DimensionWorlds[0] = eng.World
	eng.DimensionWorlds[1] = make(map[world.ChunkPos]*world.Chunk)
	eng.DimensionWorlds[2] = make(map[world.ChunkPos]*world.Chunk)
	return eng
}

// GetWorldTime returns the authoritative 24000-tick Minecraft world time.
func (e *Engine) GetWorldTime() int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.WorldTime
}

// GetWorldClock returns the total tick count and the current day time atomically.
func (e *Engine) GetWorldClock() (ticks, dayTime int64) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.TickCount, e.WorldTime
}

// SetWorldTime sets the authoritative 24000-tick Minecraft world time.
func (e *Engine) SetWorldTime(t int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.WorldTime = t % 24000
	if e.WorldTime < 0 {
		e.WorldTime += 24000
	}
}

// SetWorldClock restores the total tick count and current day time from saved world metadata.
func (e *Engine) SetWorldClock(ticks, dayTime int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if ticks < 0 {
		ticks = 0
	}
	e.TickCount = ticks
	e.WorldTime = dayTime % 24000
	if e.WorldTime < 0 {
		e.WorldTime += 24000
	}
}

func (e *Engine) spawnItemDropLocked(itemID int32, count int, pos Vec3, vel Vec3, delay float64) *ItemDrop {
	id := e.nextItemDropID
	e.nextItemDropID++
	drop := &ItemDrop{
		ID:          id,
		ItemID:      itemID,
		Count:       count,
		Position:    pos,
		Velocity:    vel,
		Age:         0,
		PickupDelay: delay,
	}
	e.ItemDrops[id] = drop
	return drop
}

func (e *Engine) SpawnItemDrop(itemID int32, count int, pos Vec3, vel Vec3) *ItemDrop {
	return e.SpawnItemDropWithDelay(itemID, count, pos, vel, 0.3)
}

func (e *Engine) SpawnItemDropWithDelay(itemID int32, count int, pos Vec3, vel Vec3, delay float64) *ItemDrop {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.spawnItemDropLocked(itemID, count, pos, vel, delay)
}

func (e *Engine) GetItemDrops() []ItemDrop {
	e.mu.RLock()
	defer e.mu.RUnlock()
	drops := make([]ItemDrop, 0, len(e.ItemDrops))
	for _, d := range e.ItemDrops {
		drops = append(drops, *d)
	}
	return drops
}

// GetPlayerList returns value snapshots of active players.
func (e *Engine) GetPlayerList() []PlayerEntity {
	e.mu.RLock()
	defer e.mu.RUnlock()
	res := make([]PlayerEntity, 0, len(e.Players))
	for id, p := range e.Players {
		if p.ClientDriven {
			if snapshot, ok := e.clientPlayers[id]; ok {
				res = append(res, snapshot)
				continue
			}
		}
		res = append(res, *p)
	}
	return res
}

// SetPlayerClientDriven enables client-owned movement and creates an AI-facing state snapshot.
func (e *Engine) SetPlayerClientDriven(id int32, enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if p := e.Players[id]; p != nil {
		p.ClientDriven = enabled
		if enabled {
			if e.clientPlayers == nil {
				e.clientPlayers = make(map[int32]PlayerEntity)
			}
			snapshot := *p
			e.clientPlayers[id] = snapshot
		} else {
			delete(e.clientPlayers, id)
		}
	}
}

// UpdateClientPlayerSnapshot publishes client-owned movement state for mob AI.
func (e *Engine) UpdateClientPlayerSnapshot(id int32, pos Vec3, gameMode int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if snapshot, ok := e.clientPlayers[id]; ok {
		snapshot.Position = pos
		snapshot.GameMode = gameMode
		e.clientPlayers[id] = snapshot
	}
}

func (e *Engine) SpawnMob(mobType MobType, pos Vec3) *MobEntity {
	e.mu.Lock()
	defer e.mu.Unlock()
	id := e.nextMobID
	e.nextMobID++
	mob := NewMobEntity(id, mobType, pos)
	e.Mobs[id] = mob
	return mob
}

func (e *Engine) DamageMob(mobID int32, damage float32) (drops []inventory.ItemStack, killed bool) {
	e.mobMu.Lock()
	defer e.mobMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	mob, ok := e.Mobs[mobID]
	if !ok || !mob.IsAlive {
		return nil, false
	}
	if mob.HurtTime > 5 {
		return nil, false
	}
	mob.Health -= damage
	mob.HurtTime = 10
	mob.LastHurtTime = time.Now()
	if mob.Health <= 0 {
		mob.Health = 0
		mob.IsAlive = false
		drops = mob.DeathDrops()
		for _, stack := range drops {
			e.nextItemDropID++
			e.ItemDrops[e.nextItemDropID] = &ItemDrop{
				ID:          e.nextItemDropID,
				ItemID:      stack.ItemID,
				Count:       stack.Count,
				Position:    mob.Position.Add(Vec3{X: 0, Y: 0.5, Z: 0}),
				Velocity:    Vec3{X: 0, Y: 0.15, Z: 0},
				PickupDelay: 0.2,
			}
		}
		delete(e.Mobs, mobID)
		return drops, true
	}
	return nil, false
}

func (e *Engine) GetMobs() []MobEntity {
	e.mobMu.Lock()
	defer e.mobMu.Unlock()
	e.mu.RLock()
	defer e.mu.RUnlock()
	res := make([]MobEntity, 0, len(e.Mobs))
	for _, m := range e.Mobs {
		if m.IsAlive {
			res = append(res, *m)
		}
	}
	return res
}

func (e *Engine) AttackMobInFront(origin, look Vec3, reach float64, damage float32) (int32, bool, bool) {
	e.mobMu.Lock()
	defer e.mobMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. If in The End (dimension 2), prioritize End Crystals and Ender Dragon
	if e.CurrentDimension == 2 {
		for _, crystal := range e.EndCrystals {
			if !crystal.IsAlive {
				continue
			}
			toCrystal := crystal.Position.Sub(origin)
			dist := toCrystal.Length()
			if dist <= reach+1.5 {
				dir := toCrystal.Normalize()
				dot := dir.X*look.X + dir.Y*look.Y + dir.Z*look.Z
				if dot > 0.70 {
					e.attackEndCrystalLocked(crystal.ID)
					return crystal.ID, true, true
				}
			}
		}

		if e.EnderDragon != nil && e.EnderDragon.IsAlive && !e.EnderDragon.IsDying {
			toDragon := e.EnderDragon.Position.Sub(origin)
			dist := toDragon.Length()
			if dist <= reach+6.0 {
				dir := toDragon.Normalize()
				dot := dir.X*look.X + dir.Y*look.Y + dir.Z*look.Z
				if dot > 0.55 {
					killed := e.damageEnderDragonLocked(damage)
					return -999, killed, true
				}
			}
		}
	}

	var closestMob *MobEntity
	closestDist := reach

	for _, mob := range e.Mobs {
		if !mob.IsAlive {
			continue
		}
		// Target mob center/chest (~0.9m above feet) for authentic hit registration
		mobCenter := mob.Position.Add(Vec3{X: 0, Y: 0.9, Z: 0})
		toMob := mobCenter.Sub(origin)
		dist := toMob.Length()
		if dist > reach {
			continue
		}
		dir := toMob.Normalize()
		dot := dir.X*look.X + dir.Y*look.Y + dir.Z*look.Z
		if dot > 0.80 && dist < closestDist {
			closestDist = dist
			closestMob = mob
		}
	}

	if closestMob == nil {
		return 0, false, false
	}
	if closestMob.HurtTime > 5 {
		return closestMob.ID, false, false
	}

	closestMob.Health -= damage
	closestMob.HurtTime = 10
	closestMob.LastHurtTime = time.Now()
	knockback := look.Scale(0.25)
	knockback.Y = 0.12
	closestMob.Velocity = closestMob.Velocity.Add(knockback)

	killed := false
	if closestMob.Health <= 0 {
		closestMob.Health = 0
		closestMob.IsAlive = false
		killed = true
		drops := closestMob.DeathDrops()
		for _, stack := range drops {
			e.nextItemDropID++
			e.ItemDrops[e.nextItemDropID] = &ItemDrop{
				ID:          e.nextItemDropID,
				ItemID:      stack.ItemID,
				Count:       stack.Count,
				Position:    closestMob.Position.Add(Vec3{X: 0, Y: 0.5, Z: 0}),
				Velocity:    Vec3{X: 0, Y: 0.15, Z: 0},
				PickupDelay: 0.2,
			}
		}
		delete(e.Mobs, closestMob.ID)
	}

	return closestMob.ID, killed, true
}

// SetStorage configures an optional world SaveManager for persistence.
func (e *Engine) SetStorage(s *world.SaveManager) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	var nether, end *world.SaveManager
	if s != nil {
		var err error
		nether, err = s.GetDimensionStorage(1)
		if err != nil {
			return err
		}
		end, err = s.GetDimensionStorage(2)
		if err != nil {
			_ = nether.Close()
			return err
		}
	}
	e.Storage = s
	e.DimensionStorage[0] = s
	e.DimensionStorage[1] = nether
	e.DimensionStorage[2] = end
	return nil
}

// SaveAll flushes all dirty loaded chunks across all dimensions to storage.
func (e *Engine) SaveAll() error {
	e.saveMu.Lock()
	defer e.saveMu.Unlock()
	type pendingChunk struct {
		dimension int
		position  world.ChunkPos
		storage   *world.SaveManager
		live      *world.Chunk
	}
	var pending []pendingChunk
	e.mu.RLock()
	for dim, chunks := range e.DimensionWorlds {
		storage := e.DimensionStorage[dim]
		if storage == nil && dim == 0 {
			storage = e.Storage
		}
		if storage == nil {
			continue
		}
		for pos, chunk := range chunks {
			if chunk != nil && chunk.IsDirty {
				pending = append(pending, pendingChunk{dimension: dim, position: pos, storage: storage, live: chunk})
			}
		}
	}
	e.mu.RUnlock()
	var firstErr error
	for _, item := range pending {
		e.mu.RLock()
		if item.dimension < 0 || item.dimension >= len(e.DimensionWorlds) ||
			e.DimensionWorlds[item.dimension][item.position] != item.live || !item.live.IsDirty {
			e.mu.RUnlock()
			continue
		}
		revision := item.live.Revision
		snapshot := item.live.Clone()
		e.mu.RUnlock()
		if err := item.storage.SaveChunk(snapshot); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		e.mu.Lock()
		if item.dimension >= 0 && item.dimension < len(e.DimensionWorlds) &&
			e.DimensionWorlds[item.dimension][item.position] == item.live && item.live.Revision == revision {
			item.live.IsDirty = false
		}
		e.mu.Unlock()
	}
	return firstErr
}

// CloseStorage closes all dimension region files after SaveAll has completed.
func (e *Engine) CloseStorage() error {
	e.saveMu.Lock()
	defer e.saveMu.Unlock()
	e.mu.Lock()
	storages := make([]*world.SaveManager, 0, len(e.DimensionStorage)+1)
	seen := make(map[*world.SaveManager]struct{}, len(e.DimensionStorage)+1)
	for i, storage := range e.DimensionStorage {
		if storage == nil {
			continue
		}
		if _, ok := seen[storage]; !ok {
			seen[storage] = struct{}{}
			storages = append(storages, storage)
		}
		e.DimensionStorage[i] = nil
	}
	if e.Storage != nil {
		if _, ok := seen[e.Storage]; !ok {
			storages = append(storages, e.Storage)
		}
		e.Storage = nil
	}
	e.mu.Unlock()

	var firstErr error
	for _, storage := range storages {
		if err := storage.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (e *Engine) LoadChunk(cp world.ChunkPos, c *world.Chunk) {
	e.LoadChunkInDimension(e.GetDimension(), cp, c)
}

// LoadChunkInDimension loads persistent data when present and installs the
// supplied fallback only when the region slot is genuinely empty.
func (e *Engine) LoadChunkInDimension(dim int, cp world.ChunkPos, c *world.Chunk) {
	if c == nil {
		return
	}
	if c.Pos != cp {
		log.Printf("[World] Refusing to install chunk %v at mismatched position %v", c.Pos, cp)
		return
	}
	loaded, err := e.GetChunkInDimensionWithError(dim, cp)
	if err != nil {
		log.Printf("[World] Refusing to replace unreadable chunk %v in dimension %d: %v", cp, dim, err)
		return
	}
	if loaded != nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if dim < 0 || dim >= len(e.DimensionWorlds) {
		return
	}
	chunks := e.DimensionWorlds[dim]
	if chunks == nil {
		chunks = make(map[world.ChunkPos]*world.Chunk)
		e.DimensionWorlds[dim] = chunks
	}
	if chunks[cp] == nil {
		chunks[cp] = c
	}
}

// LoadGeneratedChunk installs a chunk generated after the caller has already
// checked persistent storage. It avoids a second synchronous disk lookup while
// preserving a chunk another goroutine may have installed in the meantime.
func (e *Engine) LoadGeneratedChunk(cp world.ChunkPos, c *world.Chunk) {
	e.LoadGeneratedChunkInDimension(e.GetDimension(), cp, c)
}

func (e *Engine) LoadGeneratedChunkInDimension(dim int, cp world.ChunkPos, c *world.Chunk) {
	if c == nil {
		return
	}
	if c.Pos != cp {
		log.Printf("[World] Refusing to install generated chunk %v at mismatched position %v", c.Pos, cp)
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if dim < 0 || dim >= len(e.DimensionWorlds) {
		return
	}
	chunks := e.DimensionWorlds[dim]
	if chunks == nil {
		chunks = make(map[world.ChunkPos]*world.Chunk)
		e.DimensionWorlds[dim] = chunks
	}
	if existing := chunks[cp]; existing == nil {
		chunks[cp] = c
	} else if existing != c {
		for pos, state := range existing.ModifiedBlocks {
			c.SetBlock(pos.X, pos.Y, pos.Z, state)
		}
		chunks[cp] = c
	}
}

func (e *Engine) GetChunk(cp world.ChunkPos) *world.Chunk {
	return e.GetChunkInDimension(e.GetDimension(), cp)
}

func (e *Engine) GetChunkInDimension(dim int, cp world.ChunkPos) *world.Chunk {
	chunk, _ := e.GetChunkInDimensionWithError(dim, cp)
	return chunk
}

// GetChunkInDimensionWithError distinguishes an absent region slot from an
// unreadable chunk, so callers never mistake corruption for fresh terrain.
func (e *Engine) GetChunkInDimensionWithError(dim int, cp world.ChunkPos) (*world.Chunk, error) {
	e.mu.RLock()
	if dim < 0 || dim >= len(e.DimensionWorlds) {
		e.mu.RUnlock()
		return nil, nil
	}
	chunks := e.DimensionWorlds[dim]
	var chunk *world.Chunk
	if chunks != nil {
		chunk = chunks[cp]
	}
	storage := e.DimensionStorage[dim]
	if storage == nil && dim == 0 {
		storage = e.Storage
	}
	e.mu.RUnlock()
	if chunk != nil || storage == nil {
		return chunk, nil
	}

	// Region-file reads can be slow. Never hold the global simulation lock while
	// doing disk I/O; install the loaded chunk only after rechecking the map.
	loaded, err := storage.LoadChunk(cp)
	if errors.Is(err, world.ErrChunkNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if loaded == nil {
		return nil, nil
	}
	if loaded.Pos != cp {
		return nil, fmt.Errorf("storage returned chunk %v for requested position %v", loaded.Pos, cp)
	}
	e.mu.Lock()
	if dim >= 0 && dim < len(e.DimensionWorlds) {
		chunks = e.DimensionWorlds[dim]
		if chunks == nil {
			chunks = make(map[world.ChunkPos]*world.Chunk)
			e.DimensionWorlds[dim] = chunks
		}
		if existing := chunks[cp]; existing != nil {
			chunk = existing
		} else {
			chunks[cp] = loaded
			chunk = loaded
		}
	}
	e.mu.Unlock()
	return chunk, nil
}

func (e *Engine) getBlockLocked(x, y, z int32) int32 {
	if !world.ValidBlockCoordinates(x, y, z) {
		return 0
	}
	cp := world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()
	chunk := e.World[cp]
	if chunk == nil {
		return 0
	}
	return chunk.GetBlock(x, y, z)
}

func (e *Engine) GetBlock(x, y, z int32) int32 {
	if !world.ValidBlockCoordinates(x, y, z) {
		return 0
	}
	dimension := e.GetDimension()
	chunk := e.GetChunkInDimension(dimension, world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos())
	if chunk == nil {
		return 0
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.CurrentDimension != dimension {
		return 0
	}
	return chunk.GetBlock(x, y, z)
}

// GetBlockIfLoaded reads only the active in-memory world map. Rendering uses it
// to avoid lazy disk I/O while building meshes for chunk boundaries.
func (e *Engine) GetBlockIfLoaded(x, y, z int32) int32 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !world.ValidBlockCoordinates(x, y, z) {
		return 0
	}
	cp := world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()
	chunk := e.World[cp]
	if chunk == nil {
		return 0
	}
	return chunk.GetBlock(x, y, z)
}

func (e *Engine) setBlockLocked(x, y, z int32, state int32) {
	if !world.ValidBlockCoordinates(x, y, z) || state < 0 {
		return
	}
	cp := world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()
	chunk := e.World[cp]
	storage := e.DimensionStorage[e.CurrentDimension]
	if storage == nil && e.CurrentDimension == 0 {
		storage = e.Storage
	}
	if chunk == nil && storage != nil {
		// Tick-owned block changes must never synthesize a blank chunk over a
		// missing or damaged saved chunk. User-facing SetBlock loads outside
		// this lock before reaching the mutation path.
		return
	}
	if chunk == nil {
		chunk = world.NewChunk(cp)
		e.World[cp] = chunk
	}
	chunk.SetBlock(x, y, z, state)

	// Broadcast block update event
	select {
	case e.Events <- WorldEvent{
		Type:     "block_update",
		BlockPos: world.BlockPos{X: x, Y: y, Z: z},
		BlockID:  state,
	}:
	default:
	}
}

func (e *Engine) SetBlock(x, y, z int32, state int32) {
	_ = e.SetBlockChecked(x, y, z, state)
}

// SetBlockChecked writes to a loaded or genuinely absent chunk and reports
// whether the mutation was applied. An unreadable stored chunk is left intact.
func (e *Engine) SetBlockChecked(x, y, z int32, state int32) bool {
	if !world.ValidBlockCoordinates(x, y, z) || state < 0 {
		return false
	}
	dimension := e.GetDimension()
	cp := world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()
	loaded, err := e.GetChunkInDimensionWithError(dimension, cp)
	if err != nil {
		log.Printf("[World] Refusing block update in unreadable chunk %v (dimension %d): %v", cp, dimension, err)
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.CurrentDimension != dimension || dimension < 0 || dimension >= len(e.DimensionWorlds) {
		return false
	}
	if e.World[cp] == nil {
		chunk := loaded
		if chunk == nil {
			chunk = world.NewChunk(cp)
		}
		e.World[cp] = chunk
	}
	e.setBlockLocked(x, y, z, state)
	return e.World[cp] != nil && e.World[cp].GetBlock(x, y, z) == state
}

// UnloadChunk purges a distant chunk from active simulation, saving it to disk if dirty.
func (e *Engine) UnloadChunk(pos world.ChunkPos) error {
	return e.UnloadChunkInDimension(e.GetDimension(), pos)
}

func (e *Engine) UnloadChunkInDimension(dim int, pos world.ChunkPos) error {
	e.saveMu.Lock()
	defer e.saveMu.Unlock()
	e.mu.Lock()
	if dim < 0 || dim >= len(e.DimensionWorlds) {
		e.mu.Unlock()
		return nil
	}
	chunks := e.DimensionWorlds[dim]
	if chunk, exists := chunks[pos]; exists {
		storage := e.DimensionStorage[dim]
		if storage == nil && dim == 0 {
			storage = e.Storage
		}
		if chunk != nil && chunk.IsDirty {
			if storage == nil {
				e.mu.Unlock()
				return nil
			}
			revision := chunk.Revision
			snapshot := chunk.Clone()
			e.mu.Unlock()
			if err := storage.SaveChunk(snapshot); err != nil {
				return err
			}
			e.mu.Lock()
			if chunks[pos] != chunk || chunk.Revision != revision {
				e.mu.Unlock()
				return fmt.Errorf("chunk %v changed while being saved; keeping it loaded", pos)
			}
			chunk.IsDirty = false
		}
		delete(chunks, pos)
	}
	e.mu.Unlock()
	return nil
}

// BreakBlockWithTool processes player mining using held tool, enforcing authentic harvest requirements and tool wear.
func (e *Engine) BreakBlockWithTool(playerID int32, pos world.BlockPos, toolID int32) (int32, bool) {
	origBlock := e.GetBlock(pos.X, pos.Y, pos.Z)
	if origBlock == 0 || origBlock == 7 { // Air or Bedrock cannot be mined
		return 0, false
	}

	if !e.SetBlockChecked(pos.X, pos.Y, pos.Z, 0) {
		return 0, false
	}
	if carpet := e.removeUnsupportedCarpetAbove(pos); carpet != 0 {
		e.mu.Lock()
		e.spawnItemDropLocked(carpet, 1, Vec3{X: float64(pos.X) + 0.5, Y: float64(pos.Y) + 1.1, Z: float64(pos.Z) + 0.5}, Vec3{Y: 0.18}, 0.2)
		e.mu.Unlock()
	}
	if flower := e.removeUnsupportedFlowerAbove(pos); flower != 0 {
		e.mu.Lock()
		e.spawnItemDropLocked(flower, 1, Vec3{X: float64(pos.X) + 0.5, Y: float64(pos.Y) + 1.1, Z: float64(pos.Z) + 0.5}, Vec3{Y: 0.18}, 0.2)
		e.mu.Unlock()
	}
	e.removeWoodDoorOtherHalf(pos, origBlock)
	if origBlock == 61 || origBlock == 62 {
		e.mu.Lock()
		if f, ok := e.Furnaces[pos]; ok && f != nil {
			dropPos := Vec3{X: float64(pos.X) + 0.5, Y: float64(pos.Y) + 0.5, Z: float64(pos.Z) + 0.5}
			if !f.InputSlot.IsEmpty() {
				e.spawnItemDropLocked(f.InputSlot.ItemID, f.InputSlot.Count, dropPos, Vec3{Y: 0.1}, 0.2)
			}
			if !f.FuelSlot.IsEmpty() {
				e.spawnItemDropLocked(f.FuelSlot.ItemID, f.FuelSlot.Count, dropPos, Vec3{Y: 0.1}, 0.2)
			}
			if !f.OutputSlot.IsEmpty() {
				e.spawnItemDropLocked(f.OutputSlot.ItemID, f.OutputSlot.Count, dropPos, Vec3{Y: 0.1}, 0.2)
			}
			delete(e.Furnaces, pos)
		}
		e.mu.Unlock()
	}

	canHarvest := inventory.CanHarvestBlock(toolID, origBlock)
	var dropItem int32
	if inventory.IsExtendedLeaves(origBlock) {
		if toolID == inventory.ItemShears {
			canHarvest = true
			dropItem = origBlock
		} else {
			e.mu.Lock()
			canHarvest = e.nextRandomTickValueLocked()%20 == 0
			e.mu.Unlock()
			dropItem = poplarSaplingID
		}
	}

	if canHarvest {
		// Determine drop item
		if !inventory.IsExtendedLeaves(origBlock) {
			dropItem = inventory.BlockItemForState(origBlock)
		}
		switch origBlock {
		case 1, 4: // Stone, Cobblestone -> Cobblestone
			dropItem = inventory.ItemCobblestone
		case 2, 3: // Grass, Dirt -> Dirt
			dropItem = inventory.ItemDirt
		case 5: // Oak Planks
			dropItem = inventory.ItemOakPlanks
		case 17: // Oak Log
			dropItem = inventory.ItemOakLog
		case 16: // Coal Ore -> Coal
			dropItem = inventory.ItemCoal
		case 14: // Gold Ore -> Raw Gold
			dropItem = inventory.ItemRawGold
		case 15: // Iron Ore -> Raw Iron
			dropItem = inventory.ItemRawIron
		case 56: // Diamond Ore -> Diamond
			dropItem = inventory.ItemDiamond
		case 73: // Redstone Ore -> Redstone Dust
			dropItem = 331
		case 74: // Lapis Ore -> Lapis Lazuli
			dropItem = inventory.ItemLapisLazuli
		case 129: // Emerald Ore -> Emerald
			dropItem = inventory.ItemEmerald
		case 58: // Crafting Table
			dropItem = inventory.ItemCraftingTable
		case 61, 62: // Furnace, Lit Furnace -> drops Furnace (61)
			dropItem = inventory.ItemFurnace
		case 31: // Tall Grass -> Wheat Seeds
			dropItem = 295
		case 37, 38, 45, 49, 50, 87, 89, 121, 150:
			dropItem = origBlock
		}
		if definition, ok := inventory.ExtendedBlockByID(inventory.BlockItemForState(origBlock)); ok {
			if definition.DropsNothing {
				dropItem = inventory.ItemAir
			} else if definition.DropRuntimeID != 0 {
				dropItem = definition.DropRuntimeID
			}
		}

		if dropItem != inventory.ItemAir {
			dropPos := Vec3{X: float64(pos.X) + 0.5, Y: float64(pos.Y) + 0.5, Z: float64(pos.Z) + 0.5}
			dropCount := e.rollBlockDropCount(origBlock)
			if _, candleCount, _, isCandle := inventory.CandleStateProperties(origBlock); isCandle {
				dropCount = candleCount
			}
			e.mu.Lock()
			e.nextItemDropID++
			e.ItemDrops[e.nextItemDropID] = &ItemDrop{
				ID:          e.nextItemDropID,
				ItemID:      dropItem,
				Count:       dropCount,
				Position:    dropPos,
				Velocity:    Vec3{X: 0, Y: 0.18, Z: 0},
				PickupDelay: 0.4,
			}
			e.mu.Unlock()
		}
	}

	// Apply tool durability loss if player used a tool
	e.mu.Lock()
	if p, ok := e.Players[playerID]; ok && p.Inventory != nil {
		slot := p.Inventory.SelectedHotbar
		if slot >= 0 && slot < len(p.Inventory.Slots) {
			item := &p.Inventory.Slots[slot]
			if !item.IsEmpty() {
				inventory.ApplyToolWear(item)
			}
		}
	}
	e.mu.Unlock()

	return dropItem, canHarvest
}

// BreakBlock processes player mining, updates block to Air, and adds dropped item to inventory.
func (e *Engine) BreakBlock(playerID int32, pos world.BlockPos) int32 {
	origBlock := e.GetBlock(pos.X, pos.Y, pos.Z)
	if origBlock == 0 || origBlock == 7 {
		return 0
	}

	if !e.SetBlockChecked(pos.X, pos.Y, pos.Z, 0) {
		return 0
	}
	e.removeWoodDoorOtherHalf(pos, origBlock)
	if carpet := e.removeUnsupportedCarpetAbove(pos); carpet != 0 {
		e.mu.Lock()
		if player, ok := e.Players[playerID]; ok && player.Inventory != nil {
			player.Inventory.AddItem(carpet, 1)
		}
		e.mu.Unlock()
	}
	if flower := e.removeUnsupportedFlowerAbove(pos); flower != 0 {
		e.mu.Lock()
		if player, ok := e.Players[playerID]; ok && player.Inventory != nil {
			player.Inventory.AddItem(flower, 1)
		}
		e.mu.Unlock()
	}
	// Java 26.3 cushions are destroyed when their supporting block is broken.
	// Remove the unsupported block and return its item to the player as well.
	if pos.Y < world.MaxWorldY {
		above := world.BlockPos{X: pos.X, Y: pos.Y + 1, Z: pos.Z}
		unsupported := e.GetBlock(above.X, above.Y, above.Z)
		if inventory.IsCushionBlock(unsupported) && !IsSolidBlock(e.GetBlock(pos.X, pos.Y, pos.Z)) {
			if e.SetBlockChecked(above.X, above.Y, above.Z, 0) {
				e.mu.Lock()
				if player, ok := e.Players[playerID]; ok && player.Inventory != nil {
					player.Inventory.AddItem(unsupported, 1)
				}
				e.mu.Unlock()
			}
		}
	}
	if origBlock == 61 || origBlock == 62 {
		e.mu.Lock()
		if f, ok := e.Furnaces[pos]; ok && f != nil {
			dropPos := Vec3{X: float64(pos.X) + 0.5, Y: float64(pos.Y) + 0.5, Z: float64(pos.Z) + 0.5}
			if !f.InputSlot.IsEmpty() {
				e.spawnItemDropLocked(f.InputSlot.ItemID, f.InputSlot.Count, dropPos, Vec3{Y: 0.1}, 0.2)
			}
			if !f.FuelSlot.IsEmpty() {
				e.spawnItemDropLocked(f.FuelSlot.ItemID, f.FuelSlot.Count, dropPos, Vec3{Y: 0.1}, 0.2)
			}
			if !f.OutputSlot.IsEmpty() {
				e.spawnItemDropLocked(f.OutputSlot.ItemID, f.OutputSlot.Count, dropPos, Vec3{Y: 0.1}, 0.2)
			}
			delete(e.Furnaces, pos)
		}
		e.mu.Unlock()
	}

	// Determine drop item
	dropItem := inventory.BlockItemForState(origBlock)
	if inventory.IsExtendedLeaves(origBlock) {
		e.mu.Lock()
		dropsSapling := e.nextRandomTickValueLocked()%20 == 0
		e.mu.Unlock()
		if dropsSapling {
			dropItem = poplarSaplingID
		} else {
			dropItem = inventory.ItemAir
		}
	}
	switch origBlock {
	case 1: // Stone -> Cobblestone
		dropItem = inventory.ItemCobblestone
	case 2: // Grass -> Dirt
		dropItem = inventory.ItemDirt
	case 16: // Coal Ore -> Coal
		dropItem = inventory.ItemCoal
	case 14: // Gold Ore -> Raw Gold
		dropItem = inventory.ItemRawGold
	case 15: // Iron Ore -> Raw Iron
		dropItem = inventory.ItemRawIron
	case 56: // Diamond Ore -> Diamond
		dropItem = inventory.ItemDiamond
	case 73: // Redstone Ore -> Redstone Dust
		dropItem = 331
	case 74: // Lapis Ore -> Lapis Lazuli
		dropItem = inventory.ItemLapisLazuli
	case 129: // Emerald Ore -> Emerald
		dropItem = inventory.ItemEmerald
	}
	if definition, ok := inventory.ExtendedBlockByID(inventory.BlockItemForState(origBlock)); ok {
		if definition.DropsNothing {
			dropItem = inventory.ItemAir
		} else if definition.DropRuntimeID != 0 {
			dropItem = definition.DropRuntimeID
		}
		if definition.RequiresHarvestTool {
			toolID := inventory.ItemAir
			e.mu.RLock()
			if player, ok := e.Players[playerID]; ok && player.Inventory != nil {
				toolID = player.Inventory.GetSelectedItem().ItemID
			}
			e.mu.RUnlock()
			if !inventory.CanHarvestBlock(toolID, origBlock) {
				dropItem = inventory.ItemAir
			}
		}
	}
	switch origBlock {
	case 14, 15, 16, 56, 73, 74, 129:
		toolID := inventory.ItemAir
		e.mu.RLock()
		if player, ok := e.Players[playerID]; ok && player.Inventory != nil {
			toolID = player.Inventory.GetSelectedItem().ItemID
		}
		e.mu.RUnlock()
		if !inventory.CanHarvestBlock(toolID, origBlock) {
			dropItem = inventory.ItemAir
		}
	}

	dropCount := 0
	if dropItem != inventory.ItemAir {
		dropCount = e.rollBlockDropCount(origBlock)
		if _, candleCount, _, isCandle := inventory.CandleStateProperties(origBlock); isCandle {
			dropCount = candleCount
		}
	}
	e.mu.Lock()
	if p, ok := e.Players[playerID]; ok && p.Inventory != nil && dropItem != inventory.ItemAir {
		p.Inventory.AddItem(dropItem, dropCount)
	}
	e.mu.Unlock()

	return dropItem
}

func (e *Engine) rollBlockDropCount(blockID int32) int {
	minCount, maxCount := inventory.BlockDropCountRange(blockID)
	if maxCount <= minCount {
		return minCount
	}
	e.mu.Lock()
	random := e.nextRandomTickValueLocked()
	e.mu.Unlock()
	return minCount + int(random%uint64(maxCount-minCount+1))
}

// PlaceBlock places a block if it does not intersect the player's bounding box.
func (e *Engine) PlaceBlock(playerID int32, pos world.BlockPos, blockID int32) bool {
	if pressureState, ok := inventory.PressurePlatePlacementState(blockID); ok {
		blockID = pressureState
	}
	if buttonState, ok := inventory.ButtonPlacementState(blockID, 0, 1, 0); ok {
		blockID = buttonState
	}
	existing := e.GetBlock(pos.X, pos.Y, pos.Z)
	if itemID, facing, _, hinge, upper, isDoor := inventory.WoodDoorStateProperties(blockID); isDoor {
		if upper || pos.Y <= 0 || !inventory.IsReplaceableBlock(existing) || !IsSolidBlock(e.GetBlock(pos.X, pos.Y-1, pos.Z)) {
			return false
		}
		upperPos := world.BlockPos{X: pos.X, Y: pos.Y + 1, Z: pos.Z}
		if !world.ValidBlockCoordinates(upperPos.X, upperPos.Y, upperPos.Z) || !inventory.IsReplaceableBlock(e.GetBlock(upperPos.X, upperPos.Y, upperPos.Z)) {
			return false
		}
		lowerState := inventory.WoodDoorBlockState(itemID, facing, false, hinge, false)
		upperState := inventory.WoodDoorBlockState(itemID, facing, false, hinge, true)
		if e.placementIntersectsPlayer(doorPosState{pos: pos, state: lowerState}) || e.placementIntersectsPlayer(doorPosState{pos: upperPos, state: upperState}) {
			return false
		}
		if !e.SetBlockChecked(pos.X, pos.Y, pos.Z, lowerState) {
			return false
		}
		if !e.SetBlockChecked(upperPos.X, upperPos.Y, upperPos.Z, upperState) {
			_ = e.SetBlockChecked(pos.X, pos.Y, pos.Z, 0)
			return false
		}
		e.consumePlacedBlockItem(playerID, itemID)
		return true
	}
	if inventory.IsWoodDoorItem(blockID) {
		return false
	}
	if inventory.IsCandleItem(blockID) {
		if existingItem, count, lit, ok := inventory.CandleStateProperties(existing); ok && existingItem == blockID && count < 4 {
			state := inventory.CandleBlockState(existingItem, count+1, lit)
			if !e.SetBlockChecked(pos.X, pos.Y, pos.Z, state) {
				return false
			}
			e.consumePlacedBlockItem(playerID, blockID)
			return true
		}
		blockID = inventory.CandleBlockState(blockID, 1, false)
	}
	if inventory.IsPressurePlateState(blockID) {
		if pos.Y <= 0 || !IsSolidBlock(e.GetBlock(pos.X, pos.Y-1, pos.Z)) {
			return false
		}
	}
	if _, _, _, isButton := inventory.ButtonStateProperties(blockID); isButton {
		sx, sy, sz, _ := inventory.ButtonSupportOffset(blockID)
		if !world.ValidBlockCoordinates(pos.X+sx, pos.Y+sy, pos.Z+sz) || !IsSolidBlock(e.GetBlock(pos.X+sx, pos.Y+sy, pos.Z+sz)) {
			return false
		}
	}
	if !inventory.IsReplaceableBlock(existing) {
		existingItem, existingIsSlab := inventory.SlabItemForBlockState(existing)
		placedItem, placedIsSlab := inventory.SlabItemForBlockState(blockID)
		if !inventory.IsDoubleSlabState(blockID) || !existingIsSlab || !placedIsSlab || existingItem != placedItem || inventory.IsDoubleSlabState(existing) {
			return false
		}
		if !e.SetBlockChecked(pos.X, pos.Y, pos.Z, blockID) {
			return false
		}
		e.consumePlacedBlockItem(playerID, placedItem)
		return true
	}
	below := e.GetBlock(pos.X, pos.Y-1, pos.Z)
	if inventory.IsSmallFlower(blockID) && (pos.Y <= world.MinWorldY || !inventory.IsFlowerSoilBlock(below)) {
		return false // Small flowers require a vanilla flower-supporting soil block.
	}
	if (blockID == 10100 || inventory.IsCushionBlock(blockID) || inventory.IsCandleBlock(blockID) || inventory.IsCarpetBlock(blockID)) && (pos.Y <= world.MinWorldY || !IsSolidBlock(below) || (inventory.IsCarpetBlock(blockID) && inventory.IsCarpetBlock(below))) {
		return false // Red shrubs, cushions, candles, and carpets need a solid block beneath them.
	}

	// For solid blocks, prevent placing inside living players
	if IsSolidBlock(blockID) {
		blockBoxes := BlockCollisionAABBsAt(blockID, pos.X, pos.Y, pos.Z, e.GetBlock)

		e.mu.RLock()
		for _, p := range e.Players {
			if p.IsAlive {
				playerBox := NewPlayerAABB(p.Position)
				for _, blockBox := range blockBoxes {
					if PlacementIntersects(playerBox, blockBox) {
						e.mu.RUnlock()
						return false // cannot place inside player
					}
				}
			}
		}
		e.mu.RUnlock()
	}

	if !e.SetBlockChecked(pos.X, pos.Y, pos.Z, blockID) {
		return false
	}

	// Decrement the inventory item represented by this block state (creative
	// mode does not consume items).
	e.consumePlacedBlockItem(playerID, inventory.BlockItemForState(blockID))

	if blockID == inventory.ItemFurnace {
		e.mu.Lock()
		if e.Furnaces == nil {
			e.Furnaces = make(map[world.BlockPos]*inventory.FurnaceBlockEntity)
		}
		e.Furnaces[pos] = inventory.NewFurnaceBlockEntity()
		e.mu.Unlock()
	}
	return true
}

func (e *Engine) removeUnsupportedCarpetAbove(pos world.BlockPos) int32 {
	if pos.Y >= world.MaxWorldY {
		return 0
	}
	above := world.BlockPos{X: pos.X, Y: pos.Y + 1, Z: pos.Z}
	carpet := e.GetBlock(above.X, above.Y, above.Z)
	if !inventory.IsCarpetBlock(carpet) || IsSolidBlock(e.GetBlock(pos.X, pos.Y, pos.Z)) {
		return 0
	}
	if e.SetBlockChecked(above.X, above.Y, above.Z, 0) {
		return carpet
	}
	return 0
}

func (e *Engine) removeUnsupportedFlowerAbove(pos world.BlockPos) int32 {
	if pos.Y >= world.MaxWorldY {
		return 0
	}
	above := world.BlockPos{X: pos.X, Y: pos.Y + 1, Z: pos.Z}
	flower := e.GetBlock(above.X, above.Y, above.Z)
	if !inventory.IsSmallFlower(flower) || inventory.IsFlowerSoilBlock(e.GetBlock(pos.X, pos.Y, pos.Z)) {
		return 0
	}
	if e.SetBlockChecked(above.X, above.Y, above.Z, 0) {
		return flower
	}
	return 0
}

type doorPosState struct {
	pos   world.BlockPos
	state int32
}

func (e *Engine) placementIntersectsPlayer(candidate doorPosState) bool {
	if !IsSolidBlock(candidate.state) {
		return false
	}
	boxes := BlockCollisionAABBsAt(candidate.state, candidate.pos.X, candidate.pos.Y, candidate.pos.Z, e.GetBlock)
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, player := range e.Players {
		if !player.IsAlive {
			continue
		}
		playerBox := NewPlayerAABB(player.Position)
		for _, box := range boxes {
			if PlacementIntersects(playerBox, box) {
				return true
			}
		}
	}
	return false
}

// SetCandleLit changes the lit state of a placed candle cluster.
func (e *Engine) SetCandleLit(pos world.BlockPos, lit bool) bool {
	itemID, count, currentLit, ok := inventory.CandleStateProperties(e.GetBlock(pos.X, pos.Y, pos.Z))
	if !ok || currentLit == lit {
		return false
	}
	return e.SetBlockChecked(pos.X, pos.Y, pos.Z, inventory.CandleBlockState(itemID, count, lit))
}

func (e *Engine) SetFenceGateOpen(pos world.BlockPos, open bool) bool {
	itemID, facing, currentOpen, inWall, ok := inventory.FenceGateStateProperties(e.GetBlock(pos.X, pos.Y, pos.Z))
	if !ok || currentOpen == open {
		return false
	}
	return e.SetBlockChecked(pos.X, pos.Y, pos.Z, inventory.FenceGateBlockState(itemID, facing, open, inWall))
}

func (e *Engine) SetWoodDoorOpen(pos world.BlockPos, open bool) bool {
	state := e.GetBlock(pos.X, pos.Y, pos.Z)
	itemID, facing, currentOpen, hinge, upper, ok := inventory.WoodDoorStateProperties(state)
	if !ok || currentOpen == open {
		return false
	}
	lowerPos, upperPos := pos, pos
	if upper {
		lowerPos.Y--
	} else {
		upperPos.Y++
	}
	lowerState := e.GetBlock(lowerPos.X, lowerPos.Y, lowerPos.Z)
	upperState := e.GetBlock(upperPos.X, upperPos.Y, upperPos.Z)
	if lowerID, lowerFacing, _, lowerHinge, lowerUpper, valid := inventory.WoodDoorStateProperties(lowerState); !valid || lowerID != itemID || lowerFacing != facing || lowerHinge != hinge || lowerUpper {
		return false
	}
	if upperID, upperFacing, _, upperHinge, upperHalf, valid := inventory.WoodDoorStateProperties(upperState); !valid || upperID != itemID || upperFacing != facing || upperHinge != hinge || !upperHalf {
		return false
	}
	if !e.SetBlockChecked(lowerPos.X, lowerPos.Y, lowerPos.Z, inventory.WoodDoorBlockState(itemID, facing, open, hinge, false)) {
		return false
	}
	if !e.SetBlockChecked(upperPos.X, upperPos.Y, upperPos.Z, inventory.WoodDoorBlockState(itemID, facing, open, hinge, true)) {
		_ = e.SetBlockChecked(lowerPos.X, lowerPos.Y, lowerPos.Z, lowerState)
		return false
	}
	return true
}

func (e *Engine) removeWoodDoorOtherHalf(pos world.BlockPos, brokenState int32) {
	itemID, facing, _, hinge, upper, ok := inventory.WoodDoorStateProperties(brokenState)
	if !ok {
		return
	}
	otherPos := pos
	if upper {
		otherPos.Y--
	} else {
		otherPos.Y++
	}
	otherState := e.GetBlock(otherPos.X, otherPos.Y, otherPos.Z)
	otherItem, otherFacing, _, otherHinge, otherUpper, valid := inventory.WoodDoorStateProperties(otherState)
	if valid && otherItem == itemID && otherFacing == facing && otherHinge == hinge && otherUpper != upper {
		_ = e.SetBlockChecked(otherPos.X, otherPos.Y, otherPos.Z, 0)
	}
}

func (e *Engine) SetWoodTrapdoorOpen(pos world.BlockPos, open bool) bool {
	itemID, facing, currentOpen, top, ok := inventory.WoodTrapdoorStateProperties(e.GetBlock(pos.X, pos.Y, pos.Z))
	if !ok || currentOpen == open {
		return false
	}
	return e.SetBlockChecked(pos.X, pos.Y, pos.Z, inventory.WoodTrapdoorBlockState(itemID, facing, open, top))
}

// SetIronTrapdoorPowered applies the redstone-controlled open state. Iron
// trapdoors are intentionally not toggled by the player's use action.
func (e *Engine) SetIronTrapdoorPowered(pos world.BlockPos, powered bool) bool {
	facing, open, top, ok := inventory.IronTrapdoorStateProperties(e.GetBlock(pos.X, pos.Y, pos.Z))
	if !ok || open == powered {
		return false
	}
	return e.SetBlockChecked(pos.X, pos.Y, pos.Z, inventory.IronTrapdoorBlockState(facing, powered, top))
}

func (e *Engine) consumePlacedBlockItem(playerID int32, itemID int32) {
	e.mu.Lock()
	if p, ok := e.Players[playerID]; ok && p.Inventory != nil {
		if p.GameMode != 1 {
			sel := p.Inventory.GetSelectedItem()
			if sel.ItemID == itemID && sel.Count > 0 {
				_ = p.Inventory.RemoveItem(p.Inventory.SelectedHotbar, 1)
			}
		}
	}
	e.mu.Unlock()
}

// UseBoneMealOnRedShrub grows one red shrub in a random adjacent loaded,
// supported position. Bone meal is consumed only after the block update works.
func (e *Engine) UseBoneMealOnRedShrub(playerID int32, pos world.BlockPos) (world.BlockPos, bool) {
	if !world.ValidBlockCoordinates(pos.X, pos.Y, pos.Z) {
		return world.BlockPos{}, false
	}
	e.mu.Lock()
	player, ok := e.Players[playerID]
	if !ok || player == nil || player.Inventory == nil || player.Inventory.GetSelectedItem().ItemID != inventory.ItemBoneMeal ||
		e.CurrentDimension != 0 || e.getBlockLocked(pos.X, pos.Y, pos.Z) != 10100 {
		e.mu.Unlock()
		return world.BlockPos{}, false
	}
	start := int(e.nextRandomTickValueLocked() % 4)
	directions := [4][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	var grown world.BlockPos
	found := false
	for i := 0; i < len(directions); i++ {
		direction := directions[(start+i)%len(directions)]
		candidate := world.BlockPos{X: pos.X + direction[0], Y: pos.Y, Z: pos.Z + direction[1]}
		if !world.ValidBlockCoordinates(candidate.X, candidate.Y, candidate.Z) || candidate.Y <= world.MinWorldY {
			continue
		}
		below := world.BlockPos{X: candidate.X, Y: candidate.Y - 1, Z: candidate.Z}
		if e.World[candidate.ToChunkPos()] == nil || e.World[below.ToChunkPos()] == nil ||
			e.getBlockLocked(candidate.X, candidate.Y, candidate.Z) != 0 || !IsSolidBlock(e.getBlockLocked(below.X, below.Y, below.Z)) {
			continue
		}
		grown, found = candidate, true
		break
	}
	creative := player.GameMode == 1
	e.mu.Unlock()
	if !found || !e.SetBlockChecked(grown.X, grown.Y, grown.Z, 10100) {
		return world.BlockPos{}, false
	}
	if !creative {
		e.mu.Lock()
		if current, exists := e.Players[playerID]; exists && current != nil && current.Inventory != nil {
			slot := current.Inventory.SelectedHotbar
			if slot >= 0 && slot < len(current.Inventory.Slots) {
				stack := &current.Inventory.Slots[slot]
				if stack.ItemID == inventory.ItemBoneMeal && stack.Count > 0 {
					stack.Count--
					if stack.Count == 0 {
						*stack = inventory.ItemStack{}
					}
				}
			}
		}
		e.mu.Unlock()
	}
	return grown, true
}

func (e *Engine) AddPlayer(username string, uuid [16]byte, spawnPos Vec3) *PlayerEntity {
	e.mu.Lock()
	defer e.mu.Unlock()

	id := e.nextPlayerID
	e.nextPlayerID++

	p := &PlayerEntity{
		ID:         id,
		Username:   username,
		UUID:       uuid,
		Position:   spawnPos,
		SpawnPos:   spawnPos,
		OnGround:   true,
		Health:     20.0,
		Hunger:     20,
		Saturation: 5.0,
		IsAlive:    true,
		Inventory:  inventory.NewPlayerInventory(),
	}
	e.Players[id] = p

	select {
	case e.Events <- WorldEvent{
		Type:     "player_join",
		PlayerID: id,
		Position: spawnPos,
		Health:   p.Health,
	}:
	default:
	}

	return p
}

func (e *Engine) RemovePlayer(id int32) {
	e.mu.Lock()
	delete(e.Players, id)
	e.mu.Unlock()

	select {
	case e.Events <- WorldEvent{
		Type:     "player_leave",
		PlayerID: id,
	}:
	default:
	}
}

// SetPlayerArmorProtection updates the simulation snapshot used by damage
// calculations. The caller supplies a value so the engine never reads mutable
// inventory slots while another thread is drawing or editing the GUI.
func (e *Engine) SetPlayerArmorProtection(id int32, protection inventory.ArmorProtection) {
	e.mu.Lock()
	if player, ok := e.Players[id]; ok {
		player.ArmorDefense = max(0, min(20, protection.Defense))
		player.ArmorToughness = float32(math.Max(0, math.Min(20, float64(protection.Toughness))))
	}
	e.mu.Unlock()
}

func armorMitigatesDamage(reason string) bool {
	switch reason {
	case "fall", "starvation", "void", "command", "fatal", "magic", "poison", "wither", "drowning", "suffocation":
		return false
	default:
		return true
	}
}

func mitigatedDamage(amount float32, defense int, toughness float32) float32 {
	if amount <= 0 || defense <= 0 {
		return amount
	}
	effective := float32(defense) - amount/(2+toughness/4)
	minimum := float32(defense) * 0.2
	if effective < minimum {
		effective = minimum
	}
	if effective > 20 {
		effective = 20
	}
	return amount * (1 - effective/25)
}

func (e *Engine) applyPlayerDamageLocked(player *PlayerEntity, amount float32, reason string) bool {
	if player == nil || !player.IsAlive || amount <= 0 || math.IsNaN(float64(amount)) || math.IsInf(float64(amount), 0) {
		return false
	}
	if player.GameMode == 1 && reason != "void" && reason != "command" {
		return false
	}
	if player.HurtTimer > 0 && reason != "void" && reason != "command" && reason != "fatal" {
		return false
	}
	if armorMitigatesDamage(reason) {
		amount = mitigatedDamage(amount, player.ArmorDefense, player.ArmorToughness)
	}
	if reason != "void" && reason != "command" && reason != "fatal" {
		if index := activeEffectIndex(player, inventory.EffectResistance); index >= 0 {
			resistance := float32(min(1.0, 0.2*float64(player.ActiveEffects[index].Amplifier+1)))
			amount *= 1 - resistance
		}
		if player.Absorption > 0 {
			absorbed := min(player.Absorption, amount)
			player.Absorption -= absorbed
			amount -= absorbed
		}
		if amount <= 0 {
			player.HurtTimer = 10
			return true
		}
	}
	player.Health -= amount
	player.HurtTimer = 10
	if player.Health > 0 {
		return true
	}
	player.Health = 0
	player.IsAlive = false
	if snapshot, ok := e.clientPlayers[player.ID]; ok {
		snapshot.IsAlive = false
		e.clientPlayers[player.ID] = snapshot
	}
	select {
	case e.Events <- WorldEvent{Type: "player_death", PlayerID: player.ID}:
	default:
	}
	return true
}

func (e *Engine) DamagePlayer(id int32, amount float32, reason string) {
	e.mu.Lock()
	e.applyPlayerDamageLocked(e.Players[id], amount, reason)
	e.mu.Unlock()
}

func (e *Engine) RespawnPlayer(id int32, spawnPos Vec3) {
	e.mu.Lock()
	p, ok := e.Players[id]
	if ok {
		p.Health = 20.0
		p.Hunger = 20
		p.Saturation = 5.0
		p.Exhaustion = 0
		p.RegenTimer = 0
		p.StarveTimer = 0
		p.IsAlive = true
		p.Position = spawnPos
		p.SpawnPos = spawnPos
		p.Velocity = Vec3{}
		p.FallDistance = 0
		p.OnGround = true
		p.Absorption = 0
		p.ActiveEffects = [16]world.PlayerStatusEffect{}
		p.EffectCount = 0
		if snapshot, ok := e.clientPlayers[id]; ok {
			snapshot.Position = spawnPos
			snapshot.IsAlive = true
			e.clientPlayers[id] = snapshot
		}
	}
	e.mu.Unlock()

	if ok {
		select {
		case e.Events <- WorldEvent{
			Type:     "player_respawn",
			PlayerID: id,
			Position: spawnPos,
			Health:   20.0,
		}:
		default:
		}
	}
}

func (e *Engine) MovePlayer(id int32, pos Vec3, yaw, pitch float32, onGround bool) {
	e.mu.Lock()
	p, ok := e.Players[id]
	if ok && p.IsAlive {
		// Track fall distance only for server-simulated players (client-driven handles its own)
		if !p.ClientDriven {
			if !onGround && pos.Y < p.Position.Y {
				p.FallDistance += float32(p.Position.Y - pos.Y)
			} else if onGround {
				if p.FallDistance > 3.0 {
					damage := p.FallDistance - 3.0
					e.mu.Unlock()
					e.DamagePlayer(id, damage, "fall")
					e.mu.Lock()
				}
				p.FallDistance = 0
			}
		}

		p.Position = pos
		p.Yaw = yaw
		p.Pitch = pitch
		p.OnGround = onGround
	}
	e.mu.Unlock()

	if ok {
		select {
		case e.Events <- WorldEvent{
			Type:     "player_move",
			PlayerID: id,
			Position: pos,
			Yaw:      yaw,
			Pitch:    pitch,
		}:
		default:
		}
	}
}

// MovePlayerRelative handles movement inputs with authentic AABB voxel collision resolution.
// Prevents any wall or floor clipping, and supports smooth sliding along solid voxel faces.
func (e *Engine) MovePlayerRelative(id int32, delta Vec3, yaw, pitch float32) (Vec3, bool, bool) {
	e.mu.Lock()
	p, ok := e.Players[id]
	if !ok || !p.IsAlive {
		if ok && !p.IsAlive {
			// Auto respawn player at safe spawn location
			p.Health = 20.0
			p.Hunger = 20
			p.IsAlive = true
			respawnPos := p.SpawnPos
			if respawnPos.X == 0 && respawnPos.Y == 0 && respawnPos.Z == 0 {
				respawnPos = Vec3{X: 8.0, Y: 66.0, Z: 8.0}
			}
			p.Position = respawnPos
			p.Velocity = Vec3{}
			p.FallDistance = 0
			p.OnGround = true
			p.ClientDriven = true
			e.mu.Unlock()
			return p.Position, true, false
		}
		e.mu.Unlock()
		return Vec3{X: 8.0, Y: 66.0, Z: 8.0}, false, false
	}
	p.ClientDriven = true
	oldPos := p.Position
	e.mu.Unlock()

	resolvedPos, ground, hitCeiling := ResolveMovementCollisionForBlocks(oldPos, delta, e.GetBlock)

	// Check if player is in liquid (water 8/9 or lava 10/11) before acquiring lock
	bx := int32(math.Floor(resolvedPos.X))
	by := int32(math.Floor(resolvedPos.Y))
	bz := int32(math.Floor(resolvedPos.Z))
	bFeet := e.GetBlock(bx, by, bz)
	inLiquid := bFeet == 8 || bFeet == 9 || bFeet == 10 || bFeet == 11

	e.mu.Lock()
	if p, ok := e.Players[id]; ok && p.IsAlive {
		if inLiquid {
			p.FallDistance = 0
		} else if ground {
			p.Velocity.Y = 0
			if p.GameMode != 1 && p.FallDistance > 3.0 {
				fallDmg := p.FallDistance - 3.0
				e.mu.Unlock()
				e.DamagePlayer(id, fallDmg, "fall")
				e.mu.Lock()
			}
			p.FallDistance = 0
		} else if hitCeiling && p.Velocity.Y > 0 {
			p.Velocity.Y = 0
		} else if resolvedPos.Y < oldPos.Y {
			p.FallDistance += float32(oldPos.Y - resolvedPos.Y)
		} else if resolvedPos.Y >= oldPos.Y {
			p.FallDistance = 0
		}
		p.Position = resolvedPos
		p.Yaw = yaw
		p.Pitch = pitch
		p.OnGround = ground

		// Vanilla movement exhaustion
		horizDist := math.Sqrt(delta.X*delta.X + delta.Z*delta.Z)
		if horizDist > 0 {
			p.Exhaustion += float32(horizDist * 0.01)
		}
	}
	e.mu.Unlock()

	select {
	case e.Events <- WorldEvent{
		Type:     "player_move",
		PlayerID: id,
		Position: resolvedPos,
		Yaw:      yaw,
		Pitch:    pitch,
	}:
	default:
	}

	return resolvedPos, ground, hitCeiling
}

// SetPaused toggles singleplayer simulation pause state.
func (e *Engine) SetPaused(paused bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.IsPaused = paused
}

// Tick executes a single 50ms simulation step.
func (e *Engine) Tick() {
	e.mobMu.Lock()
	defer e.mobMu.Unlock()
	e.mu.Lock()
	if e.IsPaused {
		e.mu.Unlock()
		return
	}
	e.TickCount++
	e.WorldTime = (e.WorldTime + 1) % 24000
	for pos, releaseTick := range e.buttonReleaseAt {
		if e.TickCount < releaseTick {
			continue
		}
		if itemID, facing, powered, ok := inventory.ButtonStateProperties(e.getBlockLocked(pos.X, pos.Y, pos.Z)); ok && powered {
			e.setBlockLocked(pos.X, pos.Y, pos.Z, inventory.ButtonBlockState(itemID, facing, false))
		}
		delete(e.buttonReleaseAt, pos)
	}

	getBlock := func(x, y, z int32) int32 {
		cp := world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()
		chunk, ok := e.World[cp]
		if !ok || chunk == nil {
			return 0
		}
		return chunk.GetBlock(x, y, z)
	}
	isSolid := func(x, y, z int32) bool {
		return IsSolidBlock(getBlock(x, y, z))
	}

	// Apply gravity and drag for free-falling players (skip client-driven players to avoid jitter/double gravity)
	for _, p := range e.Players {
		if !p.IsAlive || p.OnGround || p.ClientDriven {
			continue
		}
		p.Velocity.Y -= 0.08                // Minecraft standard gravity per tick
		p.Velocity = p.Velocity.Scale(0.98) // Air resistance
		delta := Vec3{X: p.Velocity.X, Y: p.Velocity.Y, Z: p.Velocity.Z}
		resPos, ground, hitCeiling := ResolveMovementCollisionForBlocks(p.Position, delta, getBlock)
		if ground {
			p.Velocity.Y = 0
			if p.FallDistance > 3.0 {
				fallDmg := p.FallDistance - 3.0
				p.Health -= fallDmg
				if p.Health <= 0 {
					p.Health = 0
					p.IsAlive = false
				}
			}
			p.FallDistance = 0
		} else if hitCeiling && p.Velocity.Y > 0 {
			p.Velocity.Y = 0
		} else if resPos.Y < p.Position.Y {
			p.FallDistance += float32(p.Position.Y - resPos.Y)
		}
		p.Position = resPos
		p.OnGround = ground
	}
	e.updatePressurePlatesLocked()

	// Update physical floating item drops
	for id, drop := range e.ItemDrops {
		drop.Age += 0.05
		if drop.PickupDelay > 0 {
			drop.PickupDelay -= 0.05
		}
		// Item physics: gravity & gentle air resistance
		drop.Velocity.Y -= 0.04
		drop.Position.X += drop.Velocity.X
		drop.Position.Y += drop.Velocity.Y
		drop.Position.Z += drop.Velocity.Z

		// Ground rest collision
		bx := int32(math.Floor(drop.Position.X))
		by := int32(math.Floor(drop.Position.Y))
		bz := int32(math.Floor(drop.Position.Z))
		if isSolid(bx, by, bz) {
			drop.Position.Y = float64(by + 1)
			drop.Velocity = Vec3{}
		}

		// Proximity magnetic pull & pickup by any player within reach (2.5 blocks)
		if drop.PickupDelay <= 0 {
			for _, p := range e.Players {
				if !p.IsAlive {
					continue
				}
				dist := p.Position.DistanceTo(drop.Position)
				if dist < 2.5 {
					// Magnetic acceleration towards player
					dx := p.Position.X - drop.Position.X
					dy := (p.Position.Y + 0.6) - drop.Position.Y
					dz := p.Position.Z - drop.Position.Z
					drop.Position.X += dx * 0.2
					drop.Position.Y += dy * 0.2
					drop.Position.Z += dz * 0.2
				}
				if dist < 1.8 {
					rem := p.Inventory.AddItem(drop.ItemID, drop.Count)
					pickedUp := drop.Count - rem
					if pickedUp > 0 {
						select {
						case e.Events <- WorldEvent{
							Type:     "item_pickup",
							PlayerID: p.ID,
							BlockID:  drop.ItemID,
						}:
						default:
						}
					}
					if rem <= 0 {
						delete(e.ItemDrops, id)
					} else {
						drop.Count = rem
					}
					break
				}
			}
		}

		// Despawn ancient item drops after 5 minutes
		if drop.Age > 300.0 {
			delete(e.ItemDrops, id)
		}
	}

	// Natural Health Regeneration & Starvation per canonical Minecraft Java rules
	for _, p := range e.Players {
		if !p.IsAlive {
			continue
		}
		e.tickPlayerStatusEffectsLocked(p)

		// In Creative mode (GameMode == 1), health, hunger, saturation remain pristine
		if p.HurtTimer > 0 {
			p.HurtTimer--
		}

		if p.GameMode == 1 {
			p.Health = 20.0
			p.Hunger = 20
			p.Saturation = 5.0
			p.Exhaustion = 0
			continue
		}

		// Peaceful difficulty (0): hunger never drains and health continuously recovers
		if e.Difficulty == 0 {
			p.Hunger = 20
			p.Saturation = 5.0
			p.Exhaustion = 0
			if p.Health < 20.0 {
				p.Health = float32(math.Min(20.0, float64(p.Health+1.0)))
			}
			continue
		}

		// Drain exhaustion: each 4.0 exhaustion consumes 1.0 saturation, or 1 hunger
		for p.Exhaustion >= 4.0 {
			p.Exhaustion -= 4.0
			if p.Saturation > 0 {
				p.Saturation = float32(math.Max(0, float64(p.Saturation-1.0)))
			} else if p.Hunger > 0 {
				p.Hunger--
			}
		}

		// Fast natural regeneration: when Hunger == 20 && Saturation > 0
		if p.Hunger == 20 && p.Saturation > 0 && p.Health < 20.0 {
			p.RegenTimer++
			if p.RegenTimer >= 10 { // Every 0.5s (10 ticks)
				p.Health = float32(math.Min(20.0, float64(p.Health+1.0)))
				p.Exhaustion += 6.0 // Drains saturation quickly
				p.RegenTimer = 0
			}
		} else if p.Hunger >= 18 && p.Health < 20.0 {
			// Normal natural regeneration: when Hunger >= 18
			p.RegenTimer++
			if p.RegenTimer >= 80 { // Every 4.0s (80 ticks)
				p.Health = float32(math.Min(20.0, float64(p.Health+1.0)))
				p.Exhaustion += 6.0
				p.RegenTimer = 0
			}
		} else {
			p.RegenTimer = 0
		}

		// Starvation: when Hunger == 0
		if p.Hunger == 0 {
			p.StarveTimer++
			if p.StarveTimer >= 80 { // Every 4.0s (80 ticks)
				// Easy stops at 10 HP, Normal stops at 1 HP, Hard kills at 0 HP
				canStarve := false
				if e.Difficulty == 1 && p.Health > 10.0 {
					canStarve = true
				} else if e.Difficulty == 2 && p.Health > 1.0 {
					canStarve = true
				} else if e.Difficulty >= 3 {
					canStarve = true
				}
				if canStarve {
					p.Health -= 1.0
					if p.Health <= 0 {
						p.Health = 0
						p.IsAlive = false
					}
				}
				p.StarveTimer = 0
			}
		} else {
			p.StarveTimer = 0
		}
	}

	// Tick active Furnaces
	for pos, f := range e.Furnaces {
		if f != nil {
			changed := f.Tick()
			if changed {
				curBlock := e.getBlockLocked(pos.X, pos.Y, pos.Z)
				if f.IsBurning() && curBlock == 61 {
					e.setBlockLocked(pos.X, pos.Y, pos.Z, 62) // Lit furnace
				} else if !f.IsBurning() && curBlock == 62 {
					e.setBlockLocked(pos.X, pos.Y, pos.Z, 61) // Unlit furnace
				}
			}
		}
	}

	// Random block ticks are applied only to already loaded chunks near players.
	e.tickRandomBlocksLocked()

	// Periodic natural mob spawning & despawning every 60 ticks (3 seconds)
	if e.TickCount%60 == 0 && len(e.Players) > 0 {
		e.tickNaturalMobSpawningLocked()
	}

	// Tick Ender Dragon Boss when in End dimension
	if e.CurrentDimension == 2 && e.EnderDragon != nil {
		e.tickEnderDragon()
	}

	var activeMobs []*MobEntity
	for _, m := range e.Mobs {
		if m.IsAlive {
			activeMobs = append(activeMobs, m)
		}
	}
	e.mu.Unlock()

	// Tick mob AI and behaviors outside the mutex
	for _, m := range activeMobs {
		m.TickMob(e)
	}
}

func (e *Engine) updatePressurePlatesLocked() {
	if e.pressurePlatePowered == nil {
		e.pressurePlatePowered = make(map[world.BlockPos]struct{})
	}
	entityCounts := make(map[world.BlockPos]int)
	plateItems := make(map[world.BlockPos]int32)
	countEntity := func(position Vec3, radius float64, itemEntity bool) {
		baseY := int32(math.Floor(position.Y))
		minY, maxY := baseY, baseY
		if itemEntity {
			minY = baseY - 1
		}
		baseX, baseZ := int32(math.Floor(position.X)), int32(math.Floor(position.Z))
		for y := minY; y <= maxY; y++ {
			for x := baseX - 1; x <= baseX+1; x++ {
				for z := baseZ - 1; z <= baseZ+1; z++ {
					pos := world.BlockPos{X: x, Y: y, Z: z}
					state := e.getBlockLocked(x, y, z)
					itemID, _, ok := inventory.PressurePlateStateProperties(state)
					if !ok || (itemEntity && !inventory.IsWeightedPressurePlate(itemID) && !inventory.IsWoodenPressurePlate(itemID)) ||
						position.X+radius <= float64(x)+.125 || position.X-radius >= float64(x)+.875 ||
						position.Z+radius <= float64(z)+.125 || position.Z-radius >= float64(z)+.875 {
						continue
					}
					if itemEntity {
						if position.Y < float64(y)+.5 || position.Y > float64(y)+1.3 {
							continue
						}
					} else if position.Y < float64(y)+.05 || position.Y > float64(y)+.35 {
						continue
					}
					entityCounts[pos]++
					plateItems[pos] = itemID
				}
			}
		}
	}
	for _, player := range e.Players {
		if player != nil && player.IsAlive {
			countEntity(player.Position, .3, false)
		}
	}
	for _, mob := range e.Mobs {
		if mob != nil && mob.IsAlive {
			countEntity(mob.Position, .35, false)
		}
	}
	for _, drop := range e.ItemDrops {
		if drop != nil {
			countEntity(drop.Position, .125, true)
		}
	}
	for pos := range e.pressurePlatePowered {
		if _, exists := entityCounts[pos]; !exists {
			plateItems[pos], _, _ = inventory.PressurePlateStateProperties(e.getBlockLocked(pos.X, pos.Y, pos.Z))
		}
	}
	for pos, itemID := range plateItems {
		count := entityCounts[pos]
		signal := inventory.WeightedPressurePlateSignal(itemID, count)
		if !inventory.IsWeightedPressurePlate(itemID) && count > 0 {
			signal = 15
		}
		state := int32(0)
		if inventory.IsWeightedPressurePlate(itemID) {
			state = inventory.WeightedPressurePlateBlockState(itemID, signal)
		} else {
			state = inventory.PressurePlateBlockState(itemID, signal > 0)
		}
		if current := e.getBlockLocked(pos.X, pos.Y, pos.Z); current != state {
			e.setBlockLocked(pos.X, pos.Y, pos.Z, state)
		}
		if signal > 0 {
			e.pressurePlatePowered[pos] = struct{}{}
		} else {
			delete(e.pressurePlatePowered, pos)
		}
	}
}

// PressButton powers a button for ten game ticks and schedules its reset.
func (e *Engine) PressButton(pos world.BlockPos) bool {
	state := e.GetBlock(pos.X, pos.Y, pos.Z)
	itemID, facing, powered, ok := inventory.ButtonStateProperties(state)
	if !ok || powered {
		return false
	}
	if !e.SetBlockChecked(pos.X, pos.Y, pos.Z, inventory.ButtonBlockState(itemID, facing, true)) {
		return false
	}
	e.mu.Lock()
	if e.buttonReleaseAt == nil {
		e.buttonReleaseAt = make(map[world.BlockPos]int64)
	}
	e.buttonReleaseAt[pos] = e.TickCount + 10
	e.mu.Unlock()
	return true
}

// StartTickLoop runs the 20 TPS loop in a background goroutine.
func (e *Engine) StartTickLoop() {
	e.loopMu.Lock()
	if e.stopped || e.tickLoopStarted {
		e.loopMu.Unlock()
		return
	}
	e.tickLoopStarted = true
	ticker := time.NewTicker(50 * time.Millisecond)
	go func() {
		defer ticker.Stop()
		defer close(e.tickDone)
		defer func() {
			if r := recover(); r != nil {
				stacks := make([]byte, 1<<20)
				n := runtime.Stack(stacks, true)
				message := fmt.Sprintf("SIMULATION TICK LOOP PANIC: %v\n\nGoroutine stacks:\n%s", r, stacks[:n])
				log.Printf("[Simulation] %s", message)
				appendSimulationCrashLog(message)
			}
		}()
		for {
			select {
			case <-ticker.C:
				e.Tick()
			case <-e.stopChan:
				return
			}
		}
	}()
	e.loopMu.Unlock()
}

// TickLoopDone closes when the background simulation loop exits, whether it
// was stopped normally or terminated after a panic. Servers can observe this
// to avoid continuing to accept clients with a dead simulation.
func (e *Engine) TickLoopDone() <-chan struct{} {
	return e.tickDone
}

var simulationCrashLogMu sync.Mutex

func appendSimulationCrashLog(message string) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("[Simulation] Cannot locate user config directory for crash log: %v", err)
		return
	}
	logDir := filepath.Join(configDir, "minecraft-go")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("[Simulation] Cannot create crash log directory %q: %v", logDir, err)
		return
	}
	path := filepath.Join(logDir, "crash.log")
	entry := fmt.Sprintf("\n===== %s =====\n%s\n", time.Now().Format(time.RFC3339Nano), message)
	simulationCrashLogMu.Lock()
	defer simulationCrashLogMu.Unlock()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[Simulation] Cannot open crash log %q: %v", path, err)
		return
	}
	if _, err := file.WriteString(entry); err != nil {
		log.Printf("[Simulation] Cannot append crash log %q: %v", path, err)
	}
	if err := file.Close(); err != nil {
		log.Printf("[Simulation] Cannot close crash log %q: %v", path, err)
	}
}

func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		close(e.stopChan)
		e.loopMu.Lock()
		e.stopped = true
		started := e.tickLoopStarted
		e.loopMu.Unlock()
		if started {
			<-e.tickDone
		}
	})
}

// GetOrCreateFurnace retrieves or creates a FurnaceBlockEntity at pos.
func (e *Engine) GetOrCreateFurnace(pos world.BlockPos) *inventory.FurnaceBlockEntity {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.Furnaces == nil {
		e.Furnaces = make(map[world.BlockPos]*inventory.FurnaceBlockEntity)
	}
	f, ok := e.Furnaces[pos]
	if !ok || f == nil {
		f = inventory.NewFurnaceBlockEntity()
		e.Furnaces[pos] = f
	}
	return f
}

func (e *Engine) tickNaturalMobSpawningLocked() {
	// 1. Despawn distant mobs (> 128 blocks from all alive players)
	for id, m := range e.Mobs {
		if !m.IsAlive {
			delete(e.Mobs, id)
			continue
		}
		tooFar := true
		for _, p := range e.Players {
			if p.IsAlive && m.Position.DistanceTo(p.Position) < 128.0 {
				tooFar = false
				break
			}
		}
		if tooFar {
			delete(e.Mobs, id)
		}
	}

	// 2. Count current mobs in dimension
	hostileCount := 0
	passiveCount := 0
	for _, m := range e.Mobs {
		if !m.IsAlive {
			continue
		}
		switch m.Type {
		case MobZombie, MobSkeleton, MobCreeper, MobSpider, MobEnderman:
			hostileCount++
		case MobCow, MobPig, MobSheep, MobChicken:
			passiveCount++
		}
	}

	// Pick a player to spawn around
	var targetPlayer *PlayerEntity
	for _, p := range e.Players {
		if p.IsAlive {
			targetPlayer = p
			break
		}
	}
	if targetPlayer == nil {
		return
	}

	pPos := targetPlayer.Position
	randAngle := float64(e.TickCount%360) * math.Pi / 180.0
	randDist := 28.0 + float64((e.TickCount*17)%20)
	sx := pPos.X + randDist*math.Cos(randAngle)
	sz := pPos.Z + randDist*math.Sin(randAngle)
	wx := int32(math.Floor(sx))
	wz := int32(math.Floor(sz))

	// Find top surface block
	var surfY float64 = -999
	for y := int32(110); y >= -50; y-- {
		b := e.getBlockLocked(wx, y, wz)
		above := e.getBlockLocked(wx, y+1, wz)
		above2 := e.getBlockLocked(wx, y+2, wz)
		if IsSolidBlock(b) && (above == 0 || above == 31 || above == 37 || above == 38) && above2 == 0 {
			surfY = float64(y + 1)
			break
		}
	}
	if surfY < -60 {
		return
	}

	spawnPos := Vec3{X: sx, Y: surfY, Z: sz}
	daytime := e.WorldTime % 24000
	isNight := daytime >= 13000 && daytime <= 23000

	if e.CurrentDimension == 1 { // Nether
		if hostileCount < 20 {
			mobType := MobSkeleton
			if (e.TickCount % 3) == 0 {
				mobType = MobEnderman
			}
			e.spawnMobLocked(mobType, spawnPos)
		}
	} else if e.CurrentDimension == 2 { // The End
		if hostileCount < 25 {
			e.spawnMobLocked(MobEnderman, spawnPos)
		}
	} else { // Overworld
		if isNight && hostileCount < 25 {
			mobTypes := []MobType{MobZombie, MobSkeleton, MobSpider, MobCreeper, MobEnderman}
			choice := mobTypes[(int(e.TickCount) % len(mobTypes))]
			e.spawnMobLocked(choice, spawnPos)
		} else if !isNight && passiveCount < 20 {
			ground := e.getBlockLocked(wx, int32(surfY-1), wz)
			if ground == 2 { // Grass block
				animals := []MobType{MobCow, MobPig, MobSheep, MobChicken}
				choice := animals[(int(e.TickCount) % len(animals))]
				e.spawnMobLocked(choice, spawnPos)
			}
		}
	}
}

func (e *Engine) spawnMobLocked(mobType MobType, pos Vec3) *MobEntity {
	id := e.nextMobID
	e.nextMobID++
	m := NewMobEntity(id, mobType, pos)
	e.Mobs[id] = m
	return m
}

// ConsumeFood eats one unit of food from player's inventory, restoring hunger and saturation points.
func (e *Engine) ConsumeFood(playerID int32, slotIdx int) (bool, int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	p, ok := e.Players[playerID]
	if !ok || !p.IsAlive || p.Inventory == nil {
		return false, 0
	}
	if slotIdx < 0 || slotIdx >= len(p.Inventory.Slots) {
		return false, 0
	}
	slot := &p.Inventory.Slots[slotIdx]
	if slot.IsEmpty() {
		return false, 0
	}
	food, isFood := inventory.FoodRegistry[slot.ItemID]
	if !isFood {
		return false, 0
	}
	consumedItemID := slot.ItemID
	if p.Hunger >= 20 && !food.AlwaysEdible {
		return false, 0 // Full stomach
	}
	e.consumeFoodEffectsLocked(p, food)

	p.Hunger += food.HungerRestored
	if p.Hunger > 20 {
		p.Hunger = 20
	}
	p.Saturation += food.Saturation
	if p.Saturation > float32(p.Hunger) {
		p.Saturation = float32(p.Hunger)
	}

	slot.Count--
	if slot.Count <= 0 {
		*slot = inventory.ItemStack{}
	}
	if food.RemainderItemID != 0 {
		if slot.IsEmpty() {
			*slot = inventory.ItemStack{ItemID: food.RemainderItemID, Count: 1}
		} else if remainder := p.Inventory.AddItem(food.RemainderItemID, 1); remainder > 0 {
			e.spawnItemDropLocked(food.RemainderItemID, remainder, p.Position, Vec3{}, 0.3)
		}
	}

	select {
	case e.Events <- WorldEvent{
		Type:     "player_eat",
		PlayerID: playerID,
		BlockID:  consumedItemID,
	}:
	default:
	}

	return true, p.Hunger
}

// SwitchDimension transitions the active simulation to another dimension (0: Overworld, 1: Nether, 2: The End).
func (e *Engine) SwitchDimension(dim int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if dim < 0 || dim > 2 || dim == e.CurrentDimension {
		return
	}
	e.DimensionWorlds[e.CurrentDimension] = e.World
	e.CurrentDimension = dim
	if e.DimensionWorlds[dim] == nil {
		e.DimensionWorlds[dim] = make(map[world.ChunkPos]*world.Chunk)
	}
	e.World = e.DimensionWorlds[dim]
	e.Storage = e.DimensionStorage[dim]

	// If entering The End for the first time, spawn Ender Dragon and End Crystals
	if dim == 2 && e.EnderDragon == nil {
		e.spawnEndBossLocked()
	}

	select {
	case e.Events <- WorldEvent{
		Type:    "dimension_change",
		BlockID: int32(dim),
	}:
	default:
	}
}

// GetDimension returns current active dimension index (0: Overworld, 1: Nether, 2: End).
func (e *Engine) GetDimension() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.CurrentDimension
}

func (e *Engine) spawnEndBossLocked() {
	e.EnderDragon = &EnderDragon{
		Position:  Vec3{X: 0, Y: 75.0, Z: 0},
		Health:    200.0,
		MaxHealth: 200.0,
		IsAlive:   true,
		Phase:     0,
		TargetPos: Vec3{X: 0, Y: 75.0, Z: 0},
	}

	// 8 End Crystals atop obsidian towers
	pillarAngles := []float64{0, 45, 90, 135, 180, 225, 270, 315}
	pillarHeights := []int32{78, 86, 92, 82, 98, 84, 90, 80}

	for idx, angDeg := range pillarAngles {
		rad := angDeg * math.Pi / 180.0
		px := math.Round(42.0 * math.Cos(rad))
		pz := math.Round(42.0 * math.Sin(rad))
		topY := float64(pillarHeights[idx] + 2)

		cid := e.nextCrystalID
		e.nextCrystalID++
		e.EndCrystals[cid] = &EndCrystal{
			ID:       cid,
			Position: Vec3{X: px, Y: topY, Z: pz},
			IsAlive:  true,
		}
	}
}

// SpawnEndBoss publicly initializes the boss encounter in The End.
func (e *Engine) SpawnEndBoss() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spawnEndBossLocked()
}

func (e *Engine) attackEndCrystalLocked(crystalID int32) bool {
	c, ok := e.EndCrystals[crystalID]
	if !ok || !c.IsAlive {
		return false
	}
	c.IsAlive = false

	// If dragon is tethered to this crystal, deal 10.0 HP damage to dragon!
	if e.EnderDragon != nil && e.EnderDragon.IsAlive && e.EnderDragon.HealingCrystalID == crystalID {
		e.EnderDragon.Health -= 10.0
		e.EnderDragon.HealingCrystalID = 0
		if e.EnderDragon.Health <= 0 {
			e.EnderDragon.Health = 0
			e.EnderDragon.IsDying = true
		}
	}

	// Detonate in 4.0 block radius
	for _, p := range e.Players {
		if p.IsAlive && p.Position.DistanceTo(c.Position) < 6.0 {
			e.applyPlayerDamageLocked(p, 8.0, "explosion")
		}
	}
	return true
}

// AttackEndCrystal detonates an active End Crystal when attacked.
func (e *Engine) AttackEndCrystal(crystalID int32) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.attackEndCrystalLocked(crystalID)
}

// SetDragonHealingCrystal sets the active healing tether crystal ID.
func (e *Engine) SetDragonHealingCrystal(crystalID int32) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.EnderDragon != nil {
		e.EnderDragon.HealingCrystalID = crystalID
	}
}

func (e *Engine) damageEnderDragonLocked(damage float32) bool {
	if e.EnderDragon == nil || !e.EnderDragon.IsAlive || e.EnderDragon.IsDying {
		return false
	}
	e.EnderDragon.Health -= damage
	if e.EnderDragon.Health <= 0 {
		e.EnderDragon.Health = 0
		e.EnderDragon.IsDying = true
		return true
	}
	return false
}

// DamageEnderDragon inflicts direct damage on the boss.
func (e *Engine) DamageEnderDragon(damage float32) (killed bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.damageEnderDragonLocked(damage)
}

// GetEnderDragon returns snapshot of current Ender Dragon state.
func (e *Engine) GetEnderDragon() *EnderDragon {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.EnderDragon == nil {
		return nil
	}
	cp := *e.EnderDragon
	return &cp
}

// GetEndCrystals returns snapshot of active End Crystals.
func (e *Engine) GetEndCrystals() []EndCrystal {
	e.mu.RLock()
	defer e.mu.RUnlock()
	res := make([]EndCrystal, 0, len(e.EndCrystals))
	for _, c := range e.EndCrystals {
		res = append(res, *c)
	}
	return res
}

func (e *Engine) tickEnderDragon() {
	dragon := e.EnderDragon
	if dragon == nil {
		return
	}

	if dragon.IsDying {
		dragon.DeathTicks++
		if dragon.DeathTicks >= 100 {
			dragon.IsAlive = false
			dragon.IsDying = false
			// Spawn massive XP/celebration drops without deadlocking
			e.spawnItemDropLocked(264, 64, Vec3{X: 0, Y: 68, Z: 0}, Vec3{Y: 0.2}, 0.3)
		}
		return
	}

	if !dragon.IsAlive {
		return
	}

	// Flap wings
	dragon.WingAngle += 0.2
	if dragon.WingAngle > math.Pi*2 {
		dragon.WingAngle -= math.Pi * 2
	}

	// Healing tether: find closest alive End Crystal within 45 blocks
	dragon.HealingCrystalID = 0
	for _, c := range e.EndCrystals {
		if c.IsAlive && dragon.Position.DistanceTo(c.Position) < 45.0 {
			dragon.HealingCrystalID = c.ID
			dragon.Health = float32(math.Min(200.0, float64(dragon.Health+0.2))) // Regenerate
			break
		}
	}

	// Dragon AI Flight: Smooth 3D circular flight circling (0, 75, 0)
	flightAngle := float64(e.TickCount%1200) * (2.0 * math.Pi / 300.0)
	targetRadius := 40.0
	targetHeight := 75.0 + math.Sin(flightAngle*2.0)*6.0

	// Dive attack mode
	if dragon.DiveCooldown > 0 {
		dragon.DiveCooldown--
	} else if len(e.Players) > 0 {
		// Pick closest player to swoop down
		for _, p := range e.Players {
			if p.IsAlive {
				targetRadius = 8.0
				targetHeight = p.Position.Y + 2.0
				dist := dragon.Position.DistanceTo(p.Position)
				if dist < 4.0 {
					e.applyPlayerDamageLocked(p, 6.0, "ender_dragon")
					dragon.DiveCooldown = 300 // 15s between dives
				}
				break
			}
		}
	}

	destX := targetRadius * math.Cos(flightAngle)
	destZ := targetRadius * math.Sin(flightAngle)
	destPos := Vec3{X: destX, Y: targetHeight, Z: destZ}

	toDest := destPos.Sub(dragon.Position)
	distToDest := toDest.Length()
	if distToDest > 0.1 {
		dir := toDest.Scale(1.0 / distToDest)
		speed := 0.35
		dragon.Velocity = dir.Scale(speed)
		dragon.Position = dragon.Position.Add(dragon.Velocity)
		dragon.Yaw = float32(math.Atan2(-dir.X, dir.Z) * 180.0 / math.Pi)
		dragon.Pitch = float32(-math.Asin(dir.Y) * 180.0 / math.Pi)
	}
}
