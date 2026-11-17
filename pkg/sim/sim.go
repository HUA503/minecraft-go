package sim

import (
	"sync"
	"time"

	"github.com/HUA503/minecraft-go/pkg/world"
)

// PlayerEntity represents an active player within the simulation.
type PlayerEntity struct {
	ID       int32
	Username string
	UUID     [16]byte
	Position Vec3
	Velocity Vec3
	Yaw      float32
	Pitch    float32
	OnGround bool
}

// WorldEvent represents a simulation event to broadcast to connected clients.
type WorldEvent struct {
	Type     string // "block_update", "player_move", "player_join", "player_leave"
	PlayerID int32
	BlockPos world.BlockPos
	BlockID  int32
	Position Vec3
	Yaw      float32
	Pitch    float32
}

// Engine manages the authoritative 20 TPS game simulation.
type Engine struct {
	mu          sync.RWMutex
	World       map[world.ChunkPos]*world.Chunk
	Players     map[int32]*PlayerEntity
	nextPlayerID int32
	TickCount   int64
	Events      chan WorldEvent
	stopChan    chan struct{}
}

func NewEngine() *Engine {
	return &Engine{
		World:        make(map[world.ChunkPos]*world.Chunk),
		Players:      make(map[int32]*PlayerEntity),
		nextPlayerID: 1,
		Events:       make(chan WorldEvent, 1024),
		stopChan:     make(chan struct{}),
	}
}

func (e *Engine) LoadChunk(cp world.ChunkPos, c *world.Chunk) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.World[cp] = c
}

func (e *Engine) GetChunk(cp world.ChunkPos) *world.Chunk {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.World[cp]
}

func (e *Engine) GetBlock(x, y, z int32) int32 {
	cp := world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()
	e.mu.RLock()
	chunk, ok := e.World[cp]
	e.mu.RUnlock()
	if !ok || chunk == nil {
		return 0
	}
	return chunk.GetBlock(x, y, z)
}

func (e *Engine) SetBlock(x, y, z int32, state int32) {
	cp := world.BlockPos{X: x, Y: y, Z: z}.ToChunkPos()
	e.mu.Lock()
	chunk, ok := e.World[cp]
	if !ok || chunk == nil {
		chunk = world.NewChunk(cp)
		e.World[cp] = chunk
	}
	chunk.SetBlock(x, y, z, state)
	e.mu.Unlock()

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

func (e *Engine) AddPlayer(username string, uuid [16]byte, spawnPos Vec3) *PlayerEntity {
	e.mu.Lock()
	defer e.mu.Unlock()

	id := e.nextPlayerID
	e.nextPlayerID++

	p := &PlayerEntity{
		ID:       id,
		Username: username,
		UUID:     uuid,
		Position: spawnPos,
		OnGround: true,
	}
	e.Players[id] = p

	select {
	case e.Events <- WorldEvent{
		Type:     "player_join",
		PlayerID: id,
		Position: spawnPos,
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

func (e *Engine) MovePlayer(id int32, pos Vec3, yaw, pitch float32, onGround bool) {
	e.mu.Lock()
	p, ok := e.Players[id]
	if ok {
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

// Tick executes a single 50ms simulation step.
func (e *Engine) Tick() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.TickCount++

	// Apply gravity and drag for free-falling players
	for _, p := range e.Players {
		if !p.OnGround {
			p.Velocity.Y -= 0.08 // Minecraft standard gravity per tick
			p.Velocity = p.Velocity.Scale(0.98) // Air resistance
			p.Position = p.Position.Add(p.Velocity)

			// Simple ground check against block at feet
			footBlockY := int32(p.Position.Y)
			if footBlockY <= 64 { // Flat surface ground level test
				p.Position.Y = 64
				p.Velocity.Y = 0
				p.OnGround = true
			}
		}
	}
}

// StartTickLoop runs the 20 TPS loop in a background goroutine.
func (e *Engine) StartTickLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	go func() {
		for {
			select {
			case <-ticker.C:
				e.Tick()
			case <-e.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (e *Engine) Stop() {
	close(e.stopChan)
}
