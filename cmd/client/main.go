Warning: truncated output (original token count: 48749)
Total output lines: 6066

package main

import (
	"bufio"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/protocol"
	"github.com/HUA503/minecraft-go/pkg/render"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/sound"
	"github.com/HUA503/minecraft-go/pkg/world"
	"github.com/HUA503/minecraft-go/pkg/world/gen"
)

type RemotePlayer struct {
	Pos   sim.Vec3
	Yaw   float32
	Pitch float32
}

type sectionMeshJob struct {
	key        [3]int32
	generation uint64
	revision   uint64
	engine     *sim.Engine
}

type chunkStreamJob struct {
	pos        world.ChunkPos
	generation uint64
	dimension  int
	engine     *sim.Engine
	generator  *gen.Generator
	unload     bool
}

type chunkStreamKey struct {
	dimension int
	pos       world.ChunkPos
}

type chunkStreamFailure struct {
	generation uint64
	retryAfter time.Time
}

type ClientScreenState int

var crashLogMu sync.Mutex

const (
	ScreenTitle ClientScreenState = iota
	ScreenWorldSelect
	ScreenNewWorld
	ScreenPlaying
)

type Client struct {
	wndRef                   *render.Window
	screenState              ClientScreenState
	saveMgr                  *world.SaveManager
	currentWorldDir          string
	currentWorldName         string
	availableWorlds          []render.WorldInfo
	worldScanResult          chan []render.WorldInfo
	worldScanPending         bool
	selectedWorldIdx         int
	newWorldName             string
	newWorldMode             int // 0: Survival, 1: Creative
	newWorldSeedText         string
	newWorldActiveBox        int // 0: Name, 1: Seed
	titleSpBtn               [4]int
	titleMpBtn               [4]int
	titleQuitBtn             [4]int
	wsPlayBtn                [4]int
	wsNewBtn                 [4]int
	wsBackBtn                [4]int
	wsRects                  [][4]int
	nwCreateBtn              [4]int
	nwCancelBtn              [4]int
	nwModeBtn                [4]int
	nwNameBox                [4]int
	nwSeedBox                [4]int
	cursorItem               inventory.ItemStack
	craftGrid                [9]inventory.ItemStack
	craftResult              inventory.ItemStack
	isCraftingTableOpen      bool
	isFurnaceOpen            bool
	currentFurnacePos        world.BlockPos
	serverAddr               string
	username                 string
	uuid                     [16]byte
	conn                     net.Conn
	networkDone              chan struct{}
	networkSendQueue         chan protocol.Packet
	networkSendDone          chan struct{}
	networkSendStop          chan struct{}
	networkSendStopOnce      sync.Once
	networkDisconnected      atomic.Bool
	lastNetworkPositionSend  time.Time
	state                    protocol.ConnectionState
	framebuf                 *render.Framebuffer
	renderPanicCount         int
	quitSaving               atomic.Bool
	quitSaveFailed           atomic.Bool
	quitSaveDone             chan struct{}
	worldStarting            atomic.Bool
	worldStartFailed         atomic.Bool
	worldStartCapturePending atomic.Bool
	worldStartDone           chan struct{}
	camera                   *render.Camera
	engine                   *sim.Engine
	tickAccumulator          float64
	player                   *sim.PlayerEntity
	craftingMgr              *inventory.CraftingManager
	isLocal                  bool
	meshCache                map[[3]int32]*render.Mesh
	meshGeneration           uint64
	meshRevisions            map[[3]int32]uint64
	meshMu                   sync.RWMutex
	meshJobsMu               sync.Mutex
	meshJobsPending          map[[3]int32]sectionMeshJob
	meshJobsRunning          map[[3]int32]sectionMeshJob
	meshJobsWake             chan struct{}
	meshWorkerOnce           sync.Once
	autosaveWG               sync.WaitGroup
	autosaveRunning          atomic.Bool
	streamJobsMu             sync.Mutex
	streamJobsPending        map[chunkStreamKey]chunkStreamJob
	streamJobsRunning        map[chunkStreamKey]chunkStreamJob
	streamJobsFailed         map[chunkStreamKey]chunkStreamFailure
	streamJobsPrunedAt       time.Time
	streamJobsWake           chan struct{}
	streamJobsIdle           chan struct{}
	streamJobsActive         int
	streamWorkerOnce         sync.Once
	remoteMu                 sync.RWMutex
	remotePlayers            map[string]*RemotePlayer
	lastRemotePos            sim.Vec3
	targetedPos              world.BlockPos
	targetedFace             world.BlockPos
	hasTarget                bool
	walkCycle                float64
	isMoving                 bool
	isSprinting              bool
	isSneaking               bool
	wasSpaceDown             bool
	genOverworld             *gen.Generator
	genNether                *gen.Generator
	genEnd                   *gen.Generator
	currentDim               gen.Dimension
	portalStandTime          float64
	loadedChunks             map[world.ChunkPos]bool
	lastChunkX               int32
	lastChunkZ               int32
	lastChunkStreamTime      time.Time
	// Authentic Minecraft Mining State
	isMining            bool
	breakProgress       float64 // 0.0 to 1.0
	miningPos           world.BlockPos
	hasMiningBlock      bool
	swingProgress       float64 // 0.0 to 1.0
	isSwinging          bool
	lastHitSoundTime    time.Time
	placeCooldown       float64
	particles           []render.Particle
	showInventoryGUI    bool
	showAdvancementsGUI bool
	showPauseMenu       bool
	backBtnRect         [4]int
	quitBtnRect         [4]int
	isEating            bool
	eatTimer            float64
	lastMunchSound      time.Time
	showDebugF3         bool
	lastStepTime        time.Time
	fpsCounter          int
	currentFPS          int
	fpsTimer            time.Time
	lastFrameDuration   float64
	isChatOpen          bool
	chatInput           string
	chatMessages        []render.ChatMessage
	advancements        map[string]bool
	isCreative          bool
	isFlying            bool
	perspectiveMode     int // 0: 1st person, 1: 3rd person back, 2: 3rd person front
	lastPlayerHealth    float32
	hurtFlashTimer      float64
	currentEyeHeight    float64
	targetFov           float64
	equipProgress       float64
	lastSelectedSlot    int
	respawnBtnRect      [4]int
	deathTitleBtnRect   [4]int
}

func buildSectionMeshSafely(engine *sim.Engine, key [3]int32) (mesh *render.Mesh) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("[Client] section mesh build failed for section %v: %v\n%s", key, recovered, debug.Stack())
			// Keep a failed section empty so one malformed chunk cannot crash or
			// repeatedly stall the render loop while the rest of the world loads.
			mesh = &render.Mesh{}
		}
	}()
	return render.BuildSectionMesh(func(x, y, z int32) int32 {
		return engine.GetBlockIfLoaded(x, y, z)
	}, key[0]*16, key[1]*16, key[2]*16)
}

func meshTriangle(mesh *render.Mesh, index int) (render.Vertex, render.Vertex, render.Vertex, bool) {
	if mesh == nil || index < 0 || index+2 >= len(mesh.Indices) {
		return render.Vertex{}, render.Vertex{}, render.Vertex{}, false
	}
	i0, i1, i2 := mesh.Indices[index], mesh.Indices[index+1], mesh.Indices[index+2]
	if uint64(i0) >= uint64(len(mesh.Vertices)) || uint64(i1) >= uint64(len(mesh.Vertices)) || uint64(i2) >= uint64(len(mesh.Vertices)) {
		return render.Vertex{}, render.Vertex{}, render.Vertex{}, false
	}
	return mesh.Vertices[int(i0)], mesh.Vertices[int(i1)], mesh.Vertices[int(i2)], true
}

// findSafeSpawnPoint discovers an authentic, randomized safe surface spawn location in the procedural world.
// Evaluates terrain elevation across random coordinates within the spawn region, ensuring the player
// is placed firmly on a solid walkable surface (grass, dirt, sand, stone) strictly above sea level (>= 62),
// completely clear of water or lava hazards, and with unobstructed headroom.
func findSafeSpawnPoint(generator *gen.Generator, engine *sim.Engine) sim.Vec3 {
	// Search nearby chunks in expanding rings. This keeps startup work local and
	// ensures each candidate chunk is generated or loaded at most once.
	for radius := int32(0); radius <= 3; radius++ {
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				if radius > 0 && max(absInt32(dx), absInt32(dz)) != radius {
					continue
				}
				cp := world.ChunkPos{X: dx, Z: dz}
				dimension := int(generator.Dimension)
				chunk, err := engine.GetChunkInDimensionWithError(dimension, cp)
				if err != nil {
					log.Printf("[Client] Skipping unreadable spawn chunk %v in dimension %d: %v", cp, dimension, err)
					continue
				}
				if chunk == nil {
					chunk = generateChunkSafely(generator, cp)
					if chunk == nil {
						continue
					}
					engine.LoadGeneratedChunkInDimension(dimension, cp, chunk)
				}
				for localX := int32(0); localX < world.ChunkWidth; localX++ {
					for localZ := int32(0); localZ < world.ChunkWidth; localZ++ {
						wx := cp.X*world.ChunkWidth + localX
						wz := cp.Z*world.ChunkWidth + localZ
						for y := int32(110); y >= 62; y-- {
							b := chunk.GetBlock(wx, y, wz)
							if b == gen.BlockGrass || b == gen.BlockSand || b == gen.BlockDirt || b == gen.BlockStone {
								above1 := chunk.GetBlock(wx, y+1, wz)
								above2 := chunk.GetBlock(wx, y+2, wz)
								if !sim.IsSolidBlock(above1) && !sim.IsSolidBlock(above2) &&
									above1 != 8 && above1 != 9 && above1 != 10 && above1 != 11 &&
									above2 != 8 && above2 != 9 && above2 != 10 && above2 != 11 {
									return sim.Vec3{X: float64(wx) + 0.5, Y: float64(y + 1), Z: float64(wz) + 0.5}
								}
								break
							}
							if b == gen.BlockWater || b == gen.BlockFlowingWater {
								break
							}
						}
					}
				}
			}
		}
	}
	return sim.Vec3{X: 8.5, Y: 66.0, Z: 8.5}
}

func absInt32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func handleSlotLeftClick(cursor *inventory.ItemStack, slot *inventory.ItemStack) {
	inventory.HandleSlotLeftClick(cursor, slot)
}

func handleSlotRightClick(cursor *inventory.ItemStack, slot *inventory.ItemStack) {
	inventory.HandleSlotRightClick(cursor, slot)
}

func (c *Client) updateCraftResult() {
	if c.craftingMgr == nil {
		return
	}
	if c.isCraftingTableOpen {
		grid := make([]int32, 9)
		for i := 0; i < 9; i++ {
			if !c.craftGrid[i].IsEmpty() {
				grid[i] = c.craftGrid[i].ItemID
			}
		}
		out, ok := c.craftingMgr.MatchCrafting(grid, 3)
		if ok {
			c.craftResult = out
		} else {
			c.craftResult = inventory.ItemStack{}
		}
	} else {
		grid := make([]int32, 4)
		for i := 0; i < 4; i++ {
			if !c.craftGrid[i].IsEmpty() {
				grid[i] = c.craftGrid[i].ItemID
			}
		}
		out, ok := c.craftingMgr.MatchCrafting(grid, 2)
		if ok {
			c.craftResult = out
		} else {
			c.craftResult = inventory.ItemStack{}
		}
	}
}

func (c *Client) CloseCurrentGUI() {
	if c.player == nil || c.player.Inventory == nil {
		c.showInventoryGUI = false
		c.isCraftingTableOpen = false
		c.isFurnaceOpen = false
		if c.wndRef != nil {
			c.wndRef.SetMouseCaptured(true)
		}
		return
	}

	for i := range c.craftGrid {
		if !c.craftGrid[i].IsEmpty() {
			rem := c.player.Inventory.AddItem(c.craftGrid[i].ItemID, c.craftGrid[i].Count)
			if rem > 0 && c.engine != nil {
				c.engine.SpawnItemDrop(c.craftGrid[i].ItemID, rem, c.player.Position.Add(sim.Vec3{X: 0, Y: 1.0, Z: 0}), sim.Vec3{})
			}
			c.craftGrid[i] = inventory.ItemStack{}
		}
	}
	c.craftResult = inventory.ItemStack{}

	if !c.cursorItem.IsEmpty() {
		rem := c.player.Inventory.AddItem(c.cursorItem.ItemID, c.cursorItem.Count)
		if rem > 0 && c.engine != nil {
			c.engine.SpawnItemDrop(c.cursorItem.ItemID, rem, c.player.Position.Add(sim.Vec3{X: 0, Y: 1.0, Z: 0}), sim.Vec3{})
		}
		c.cursorItem = inventory.ItemStack{}
	}

	c.showInventoryGUI = false
	c.isCraftingTableOpen = false
	c.isFurnaceOpen = false
	if c.wndRef != nil {
		c.wndRef.SetMouseCaptured(true)
	}
}

func (c *Client) scanSavedWorlds() {
	c.availableWorlds = nil
	if c.worldScanPending {
		return
	}
	result := make(chan []render.WorldInfo, 1)
	c.worldScanResult = result
	c.worldScanPending = true
	go func() {
		result <- loadSavedWorlds()
	}()
}

func (c *Client) pollSavedWorldScan() {
	if !c.worldScanPending || c.worldScanResult == nil {
		return
	}
	select {
	case worlds := <-c.worldScanResult:
		c.availableWorlds = worlds
		c.worldScanResult = nil
		c.worldScanPending = false
		if len(worlds) == 0 {
			c.selectedWorldIdx = -1
		} else if c.selectedWorldIdx < 0 || c.selectedWorldIdx >= len(worlds) {
			c.selectedWorldIdx = 0
		}
	default:
	}
}

func loadSavedWorlds() []render.WorldInfo {
	var worlds []render.WorldInfo
	savesDir := "saves"
	_ = os.MkdirAll(savesDir, 0755)
	entries, err := os.ReadDir(savesDir)
	if err != nil {
		return worlds
	}
	for _, entry := range entries {
		if entry.IsDir() {
			dirPath := filepath.Join(savesDir, entry.Name())
			datPath := filepath.Join(dirPath, "level.dat")
			if fi, err := os.Stat(datPath); err == nil {
				wInfo := render.WorldInfo{
					DirName:   entry.Name(),
					LevelName: entry.Name(),
					LastTime:  fi.ModTime(),
				}
				sm, err := world.NewSaveManager(dirPath)
				if err == nil {
					lvl, err := sm.LoadLevel()
					if err == nil && lvl != nil {
						if lvl.LevelName != "" {
							wInfo.LevelName = lvl.LevelName
						}
						wInfo.GameMode = lvl.GameMode
					}
					_ = sm.Close()
				}
				worlds = append(worlds, wInfo)
			}
		}
	}
	return worlds
}

func validSavedPlayerPosition(pos sim.Vec3) bool {
	return !math.IsNaN(pos.X) && !math.IsInf(pos.X, 0) && math.Abs(pos.X) <= 29_999_984 &&
		!math.IsNaN(pos.Y) && !math.IsInf(pos.Y, 0) && pos.Y >= -2048 && pos.Y <= 2048 &&
		!math.IsNaN(pos.Z) && !math.IsInf(pos.Z, 0) && math.Abs(pos.Z) <= 29_999_984
}

func savedWorldSpawn(player *sim.PlayerEntity) sim.Vec3 {
	validSpawn := func(pos sim.Vec3) bool {
		return validSavedPlayerPosition(pos) && pos.Y >= float64(world.MinWorldY) && pos.Y <= float64(world.MaxWorldY) &&
			(pos.X != 0 || pos.Y != 0 || pos.Z != 0)
	}
	if player != nil && validSpawn(player.SpawnPos) {
		return player.SpawnPos
	}
	if player != nil && validSpawn(player.Position) {
		return player.Position
	}
	return sim.Vec3{X: 8.5, Y: 66, Z: 8.5}
}

func validRemotePlayerPosition(pos protocol.PlayerPositionPacket) bool {
	return !math.IsNaN(pos.X) && !math.IsInf(pos.X, 0) && math.Abs(pos.X) <= world.WorldBorderLimit &&
		!math.IsNaN(pos.Y) && !math.IsInf(pos.Y, 0) && pos.Y >= float64(world.MinWorldY-64) && pos.Y <= float64(world.MaxWorldY+64) &&
		!math.IsNaN(pos.Z) && !math.IsInf(pos.Z, 0) && math.Abs(pos.Z) <= world.WorldBorderLimit &&
		!math.IsNaN(float64(pos.Yaw)) && !math.IsInf(float64(pos.Yaw), 0) &&
		!math.IsNaN(float64(pos.Pitch)) && !math.IsInf(float64(pos.Pitch), 0) && pos.Pitch >= -90 && pos.Pitch <= 90
}

func restoreSavedInventory(target *inventory.PlayerInventory, saved []world.SavedItemStack) {
	if target == nil {
		return
	}
	for _, item := range saved {
		if item.Slot < 0 || item.Slot >= len(target.Slots) || item.ItemID <= 0 || item.Count <= 0 {
			continue
		}
		if !target.Slots[item.Slot].IsEmpty() {
			continue
		}
		count := item.Count
		if max := inventory.GetMaxStackSize(item.ItemID); count > max {
			count = max
		}
		damage := item.Damage
		if damage < 0 {
			damage = 0
		}
		if props, ok := inventory.GetToolProperties(item.ItemID); ok && props.MaxDurability > 0 && damage >= props.MaxDurability {
			continue
		}
		target.Slots[item.Slot] = inventory.ItemStack{ItemID: item.ItemID, Count: count, Damage: damage}
	}
}

func createUniqueWorldDirectory(dirName string) (string, error) {
	name := filepath.Base(filepath.Clean(dirName))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "", fmt.Errorf("invalid world directory name %q", dirName)
	}
	root := "saves"
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", err
	}
	for suffix := 0; suffix < 1000; suffix++ {
		candidate := name
		if suffix > 0 {
			candidate = fmt.Sprintf("%s_%d", name, suffix)
		}
		path := filepath.Join(root, candidate)
		if err := os.Mkdir(path, 0755); err == nil {
			return path, nil
		} else if !os.IsExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("could not allocate a unique directory for world %q", dirName)
}

func (c *Client) StartSingleplayerWorld(worldDirName string, isNew bool, newSeed int64, newLevelName string, newGameMode int) {
	if newGameMode != 1 {
		newGameMode = 0
	}
	worldPath := filepath.Join("saves", worldDirName)
	if isNew {
		var err error
		worldPath, err = createUniqueWorldDirectory(worldDirName)
		if err != nil {
			log.Printf("[Client] Failed to create world directory: %v", err)
			return
		}
	} else if info, err := os.Stat(worldPath); err != nil || !info.IsDir() {
		if err == nil {
			err = fmt.Errorf("path is not a directory")
		}
		log.Printf("[Client] Failed to open world directory %q: %v", worldPath, err)
		return
	}

	sm, err := world.NewSaveManager(worldPath)
	if err != nil {
		log.Printf("[Client] Failed to open save manager: %v\n", err)
		c.saveMgr = nil
		return
	}
	c.saveMgr = sm
	c.currentWorldDir = worldPath
	c.currentWorldName = newLevelName
	if c.currentWorldName == "" {
		c.currentWorldName = worldDirName
	}

	c.engine = sim.NewEngine()
	c.advancements = make(map[string]bool)
	if sm != nil {
		if err := c.engine.SetStorage(sm); err != nil {
			log.Printf("[Client] Failed to initialize dimension storage: %v", err)
			_ = sm.Close()
			c.saveMgr = nil
			c.engine = nil
			c.player = nil
			c.currentWorldDir = ""
			c.currentWorldName = ""
			return
		}
	}

	var seed int64
	var spawnPos sim.Vec3
	var savedPlayer *world.PlayerSaveData

	if isNew {
		if newSeed == 0 {
			seed = time.Now().UnixNano()
		} else {
			seed = newSeed
		}
		c.genOverworld = gen.NewGenerator(seed, gen.DimensionOverworld)
		c.genNether = gen.NewGenerator(seed, gen.DimensionNether)
		c.genEnd = gen.NewGenerator(seed, gen.DimensionEnd)
		c.currentDim = gen.DimensionOverworld

		spawnPos = findSafeSpawnPoint(c.genOverworld, c.engine)
		c.player = c.engine.AddPlayer(c.username, c.uuid, spawnPos)
		c.engine.SetPlayerClientDriven(c.player.ID, true)
		c.player.GameMode = newGameMode
		c.UnlockAdvancement("minecraft:story/root", "进入世界")
		c.isCreative = (newGameMode == 1)
		if c.isCreative {
			c.isFlying = true
		}

		// Initial level.dat save
		lvlData := &world.LevelData{
			Seed:      seed,
			DayTime:   c.engine.GetWorldTime(),
			SpawnX:    int32(spawnPos.X),
			SpawnY:    int32(spawnPos.Y),
			SpawnZ:    int32(spawnPos.Z),
			LevelName: c.currentWorldName,
			GameMode:  newGameMode,
		}
		if sm != nil {
			if err := sm.SaveLevel(lvlData); err != nil {
				log.Printf("[Client] Failed to save initial level data: %v", err)
				if closeErr := c.engine.CloseStorage(); closeErr != nil {
					log.Printf("[Client] Failed to close storage after initial save failure: %v", closeErr)
				}
				c.saveMgr = nil
				c.engine = nil
				c.player = nil
				c.currentWorldDir = ""
				c.currentWorldName = ""
				return
			}
		}

		// Spawn authentic natural herds and flocks around meadow
		spawnMobOnSurface := func(mobType sim.MobType, ox, oz float64) {
			mx := spawnPos.X + ox
			mz := spawnPos.Z + oz
			wx := int32(math.Floor(mx))
			wz := int32(math.Floor(mz))
			cp := world.BlockPos{X: wx, Y: 0, Z: wz}.ToChunkPos()
			chk, loadErr := c.engine.GetChunkInDimensionWithError(int(gen.DimensionOverworld), cp)
			if loadErr != nil {
				log.Printf("[Client] Skipping mob spawn near unreadable chunk %v: %v", cp, loadErr)
				return
			}
			if chk == nil {
				chk = generateChunkSafely(c.genOverworld, cp)
				if chk == nil {
					return
				}
				c.engine.LoadGeneratedChunkInDimension(int(gen.DimensionOverworld), cp, chk)
			}
			surfY := spawnPos.Y
			for y := int32(110); y >= 55; y-- {
				b := chk.GetBlock(wx, y, wz)
				if sim.IsSolidBlock(b) {
					surfY = float64(y + 1)
					break
				}
			}
			c.engine.SpawnMob(mobType, sim.Vec3{X: mx, Y: surfY, Z: mz})
		}

		spawnMobOnSurface(sim.MobCow, -6.5, 5.0)
		spawnMobOnSurface(sim.MobCow, -8.0, 7.0)
		spawnMobOnSurface(sim.MobPig, 6.0, 4.5)
		spawnMobOnSurface(sim.MobPig, 8.0, 6.0)
		spawnMobOnSurface(sim.MobSheep, -12.0, -8.0)
		spawnMobOnSurface(sim.MobSheep, -10.5, -10.0)
		spawnMobOnSurface(sim.MobChicken, 9.0, -7.0)
		spawnMobOnSurface(sim.MobChicken, 11.0, -8.5)
		// Hostile mobs (Zombies, Skeletons, Creepers) spawn naturally in dark caves or night per vanilla rules
	} else {
		lvlData, err := sm.LoadLevel()
		if err == nil && lvlData != nil {
			if lvlData.Time != 0 || lvlData.DayTime != 0 {
				c.engine.SetWorldClock(lvlData.Time, lvlData.DayTime)
			} else {
				// Older saves omitted the clock fields; preserve the established noon default.
				c.engine.SetWorldClock(0, 6000)
			}
			seed = lvlData.Seed
			newGameMode = 0
			if lvlData.GameMode == 1 {
				newGameMode = 1
			}
			if lvlData.LevelName != "" {
				c.currentWorldName = lvlData.LevelName
			}
			savedPlayer = lvlData.Player
			spawnPos = sim.Vec3{X: float64(lvlData.SpawnX), Y: float64(lvlData.SpawnY), Z: float64(lvlData.SpawnZ)}
		} else {
			log.Printf("[Client] Refusing to load world %q because level.dat could not be read: %v", worldDirName, err)
			if closeErr := sm.Close(); closeErr != nil {
				log.Printf("[Client] Failed to close save manager after load failure: %v", closeErr)
			}
			c.saveMgr = nil
			c.engine = nil
			return
		}

		c.genOverworld = gen.NewGenerator(seed, gen.DimensionOverworld)
		c.genNether = gen.NewGenerator(seed, gen.DimensionNether)
		c.genEnd = gen.NewGenerator(seed, gen.DimensionEnd)
		c.currentDim = gen.DimensionOverworld
		if (spawnPos.X == 0 && spawnPos.Y == 0 && spawnPos.Z == 0) || spawnPos.Y < float64(world.MinWorldY) || spawnPos.Y > float64(world.MaxWorldY) || math.Abs(spawnPos.X) > 29_999_984 || math.Abs(spawnPos.Z) > 29_999_984 {
			spawnPos = findSafeSpawnPoint(c.genOverworld, c.engine)
		}

		posToSpawn := spawnPos
		if savedPlayer != nil {
			savedPos := sim.Vec3{X: savedPlayer.X, Y: savedPlayer.Y, Z: savedPlayer.Z}
			if validSavedPlayerPosition(savedPos) && (savedPos.X != 0 || savedPos.Y != 0 || savedPos.Z != 0) {
				posToSpawn = savedPos
			}
		}
		c.player = c.engine.AddPlayer(c.username, c.uuid, posToSpawn)
		c.engine.SetPlayerClientDriven(c.player.ID, true)
		c.player.GameMode = newGameMode
		c.isCreative = (newGameMode == 1)
		if c.isCreative {
			c.isFlying = true
		}

		if savedPlayer != nil {
			for _, id := range savedPlayer.Advancements {
				c.advancements[id] = true
			}
			if math.IsNaN(float64(savedPlayer.Health)) || math.IsInf(float64(savedPlayer.Health), 0) {
				c.player.Health = 20
			} else {
				c.player.Health = float32(math.Max(0, math.Min(20, float64(savedPlayer.Health))))
			}
			c.player.Hunger = max(0, min(20, savedPlayer.Hunger))
			if !math.IsNaN(float64(savedPlayer.Yaw)) && !math.IsInf(float64(savedPlayer.Yaw), 0) {
				c.camera.Yaw = float64(savedPlayer.Yaw)
			}
			if !math.IsNaN(float64(savedPlayer.Pitch)) && !math.IsInf(float64(savedPlayer.Pitch), 0) {
				c.camera.Pitch = math.Max(-90, math.Min(90, float64(savedPlayer.Pitch)))
			}
			restoreSavedInventory(c.player.Inventory, savedPlayer.Inventory)
		}
		c.UnlockAdvancement("minecraft:story/root", "进入世界")
	}

	c.publishPlayerSnapshot()
	c.camera.X = c.player.Position.X
	c.camera.Y = c.player.Position.Y + 1.62
	c.camera.Z = c.player.Position.Z

	c.resetMeshCache(true)

	// Prime only the nearby chunks here; the normal simulation loop streams the
	// remaining chunks progressively so world creation does not block the window callback.
	c.EnsureChunksAroundPlayer(c.player.Position.X, c.player.Position.Z, 3)

	c.screenState = ScreenPlaying
}

func (c *Client) beginSingleplayerWorldStart(worldDirName string, isNew bool, newSeed int64, newLevelName string, newGameMode int) {
	if c.quitSaving.Load() || !c.worldStarting.CompareAndSwap(false, true) {
		return
	}
	c.worldStartFailed.Store(false)
	c.worldStartCapturePending.Store(false)
	done := make(chan struct{})
	c.worldStartDone = done
	go func() {
		defer close(done)
		defer func() {
			if recovered := recover(); recovered != nil {
				msg := fmt.Sprintf("CLIENT WORLD START PANIC: %v\n\nStack:\n%s", recovered, debug.Stack())
				log.Print(msg)
				writeCrashLog(msg)
				c.worldStartFailed.Store(true)
				func() {
					defer func() {
						if cleanupPanic := recover(); cleanupPanic != nil {
							log.Printf("[Client] Cleanup after world-start panic also failed: %v", cleanupPanic)
						}
					}()
					if c.engine != nil {
						if err := c.SaveAndQuitToTitle(); err != nil {
							log.Printf("[Client] Cleanup after world-start panic had save errors: %v", err)
						}
					}
				}()
			}
			c.worldStarting.Store(false)
		}()
		c.StartSingleplayerWorld(worldDirName, isNew, newSeed, newLevelName, newGameMode)
		if c.screenState == ScreenPlaying {
			c.worldStartCapturePending.Store(true)
		} else {
			c.worldStartFailed.Store(true)
		}
	}()
}

// RespawnLocalPlayer resurrects the local player at a safe spawn location, resets vital stats, and reloads surrounding chunks.
func (c *Client) RespawnLocalPlayer() {
	if c.player == nil || c.engine == nil {
		return
	}
	spawnPos := savedWorldSpawn(c.player)
	c.engine.RespawnPlayer(c.player.ID, spawnPos)
	c.player.Health = 20.0
	c.player.Hunger = 20
	c.player.Saturation = 5.0
	c.player.Exhaustion = 0
	c.player.IsAlive = true
	c.player.Position = spawnPos
	c.player.Velocity = sim.Vec3{}
	c.player.FallDistance = 0
	c.player.OnGround = true

	c.camera.X = spawnPos.X
	c.camera.Y = spawnPos.Y + 1.62
	c.camera.Z = spawnPos.Z
	c.hurtFlashTimer = 0

	c.resetMeshCache(false)

	c.EnsureChunksAroundPlayer(spawnPos.X, spawnPos.Z, 3)
}

func (c *Client) SaveAndQuitToTitle() error {
	var saveErrors []error
	// Do not close region files while a periodic save still uses them.
	c.autosaveWG.Wait()
	if c.conn != nil {
		c.stopNetworkSender()
		if c.networkDone != nil {
			<-c.networkDone
		}
		if c.networkSendDone != nil {
			<-c.networkSendDone
		}
	}
	if c.engine != nil {
		// Invalidate and drain in-flight chunk reads/generation/saves before the
		// final world flush, or a late worker could install data after SaveAll.
		c.resetMeshCache(true)
		c.waitForChunkStreamIdle()
		c.engine.Stop()
		if err := c.engine.SaveAll(); err != nil {
			log.Printf("[Client] Failed to save one or more chunks while leaving the world: %v", err)
			saveErrors = append(saveErrors, fmt.Errorf("save chunks: %w", err))
		}

		if c.saveMgr != nil && c.player != nil {
			savedInv := make([]world.SavedItemStack, 0, 36)
			for slotIdx, s := range c.player.Inventory.Slots {
				if !s.IsEmpty() {
					savedInv = append(savedInv, world.SavedItemStack{
						Slot:   slotIdx,
						ItemID: s.ItemID,
						Count:  s.Count,
						Damage: s.Damage,
					})
				}
			}
			gm := 0
			if c.isCreative {
				gm = 1
			}
			seed := int64(1)
			if c.genOverworld != nil {
				seed = c.genOverworld.Seed
			}
			saveHealth := c.player.Health
			saveX, saveY, saveZ := c.player.Position.X, c.player.Position.Y, c.player.Position.Z
			if saveHealth <= 0 {
				saveHealth = 20.0
				sp := savedWorldSpawn(c.player)
				saveX, saveY, saveZ = sp.X, sp.Y, sp.Z
			}
			spawnPos := savedWorldSpawn(c.player)
			ticks, dayTime := c.engine.GetWorldClock()
			lvl := &world.LevelData{
				Seed:      seed,
				Time:      ticks,
				DayTime:   dayTime,
				SpawnX:    int32(spawnPos.X),
				SpawnY:    int32(spawnPos.Y),
				SpawnZ:    int32(spawnPos.Z),
				LevelName: c.currentWorldName,
				GameMode:  gm,
				Player: &world.PlayerSaveData{
					X:            saveX,
					Y:            saveY,
					Z:            saveZ,
					Yaw:          float32(c.camera.Yaw),
					Pitch:        float32(c.camera.Pitch),
					Health:       saveHealth,
					Hunger:       c.player.Hunger,
					GameMode:     gm,
					Inventory:    savedInv,
					Advancements: c.advancementIDs(),
				},
			}
			if err := c.saveMgr.SaveLevel(lvl); err != nil {
				log.Printf("[Client] Failed to save level data while leaving the world: %v", err)
				saveErrors = append(saveErrors, fmt.Errorf("save level data: %w", err))
			}
			c.saveMgr = nil
		}
		if err := c.engine.CloseStorage(); err != nil {
			log.Printf("[Client] Failed to close world save files: %v", err)
			saveErrors = append(saveErrors, fmt.Errorf("close world storage: %w", err))
		}

		c.engine = nil
		c.player = nil
	} else if c.saveMgr != nil {
		if err := c.saveMgr.Close(); err != nil {
			log.Printf("[Client] Failed to close partially initialized save manager: %v", err)
			saveErrors = append(saveErrors, fmt.Errorf("close partial save manager: %w", err))
		}
		c.saveMgr = nil
	}

	c.resetMeshCache(true)

	c.showPauseMenu = false
	c.showInventoryGUI = false
	c.isCraftingTableOpen = false
	c.isFurnaceOpen = false
	c.screenState = ScreenTitle
	if len(saveErrors) > 0 {
		saveErr := errors.Join(saveErrors...)
		msg := fmt.Sprintf("WORLD SAVE FAILED: %v", saveErr)
		writeCrashLog(msg)
		c.quitSaveFailed.Store(true)
		return saveErr
	}
	c.quitSaveFailed.Store(false)
	return nil
}

// beginSaveAndQuitToTitle keeps the native message loop responsive while the
// current world is flushed to disk. The main loop stops touching world state
// until quitSaving is cleared after SaveAndQuitToTitle finishes.
func (c *Client) beginSaveAndQuitToTitle(wnd *render.Window) {
	if !c.quitSaving.CompareAndSwap(false, true) {
		return
	}
	c.quitSaveFailed.Store(false)
	if c.engine != nil {
		c.engine.SetPaused(true)
	}
	if wnd != nil {
		wnd.SetMouseCaptured(false)
	}
	done := make(chan struct{})
	c.quitSaveDone = done
	go func() {
		defer close(done)
		defer func() {
			if recovered := recover(); recovered != nil {
				msg := fmt.Sprintf("CLIENT SAVE-AND-QUIT PANIC: %v\n\nStack:\n%s", recovered, debug.Stack())
				log.Print(msg)
				writeCrashLog(msg)
				c.quitSaveFailed.Store(true)
			}
		}()
		if err := c.SaveAndQuitToTitle(); err != nil {
			log.Printf("[Client] World save finished with errors: %v", err)
		}
		c.quitSaving.Store(false)
	}()
}

// snapshotLevelData copies main-thread-owned player state before a background
// autosave starts. Chunk data is snapshotted separately by Engine.SaveAll.
func (c *Client) snapshotLevelData() *world.LevelData {
	if c.engine == nil || c.player == nil || c.camera == nil {
		return nil
	}
	gm := 0
	if c.isCreative {
		gm = 1
	}
	seed := int64(1)
	if c.genOverworld != nil {
		seed = c.genOverworld.Seed
	}
	inventory := make([]world.SavedItemStack, 0, len(c.player.Inventory.Slots))
	for slot, stack := range c.player.Inventory.Slots {
		if stack.IsEmpty() {
			continue
		}
		inventory = append(inventory, world.SavedItemStack{Slot: slot, ItemID: stack.ItemID, Count: stack.Count, Damage: stack.Damage})
	}
	ticks, dayTime := c.engine.GetWorldClock()
	position := c.player.Position
	spawnPos := savedWorldSpawn(c.player)
	return &world.LevelData{
		Seed: seed, Time: ticks, DayTime: dayTime,
		SpawnX: int32(spawnPos.X), SpawnY: int32(spawnPos.Y), SpawnZ: int32(spawnPos.Z),
		LevelName: c.currentWorldName, GameMode: gm,
		Player: &world.PlayerSaveData{
			X: position.X, Y: position.Y, Z: position.Z,
			Yaw: float32(c.camera.Yaw), Pitch: float32(c.camera.Pitch),
			Health: c.player.Health, Hunger: c.player.Hunger, GameMode: gm,
			Inventory: inventory, Advancements: c.advancementIDs(),
		},
	}
}

func NewClient(serverAddr, username string, isSingleplayer bool, width, height int) *Client {
	var u [16]byte
	rand.Read(u[:])

	cli := &Client{
		serverAddr:        serverAddr,
		username:          username,
		uuid:              u,
		state:             protocol.StateHandshake,
		framebuf:          render.NewFramebuffer(width, height),
		camera:            render.NewCamera(8.0, 66.62, 8.0),
		craftingMgr:       inventory.NewCraftingManager(),
		isLocal:           isSingleplayer,
		meshCache:         make(map[[3]int32]*render.Mesh),
		meshRevisions:     make(map[[3]int32]uint64),
		meshJobsPending:   make(map[[3]int32]sectionMeshJob),
		meshJobsRunning:   make(map[[3]int32]sectionMeshJob),
		meshJobsWake:      make(chan struct{}, 1),
		streamJobsPending: make(map[chunkStreamKey]chunkStreamJob),
		streamJobsRunning: make(map[chunkStreamKey]chunkStreamJob),
		streamJobsWake:    make(chan struct{}, 1),
		streamJobsIdle:    make(chan struct{}, 1),
		loadedChunks:      make(map[world.ChunkPos]bool),
		remotePlayers:     make(map[string]*RemotePlayer),
		lastChunkX:        -999,
		lastChunkZ:        -999,
		currentEyeHeight:  1.62,
		targetFov:         70.0,
		equipProgress:     1.0,
		lastSelectedSlot:  1,
		screenState:       ScreenTitle,
		newWorldName:      "新世界",
		newWorldMode:      0,
	}

	if isSingleplayer && serverAddr == "" {
		cli.scanSavedWorlds()
	} else if serverAddr != "" {
		cli.screenState = ScreenPlaying
	}

	return cli
}

func (c *Client) getGenerator() *gen.Generator {
	switch c.currentDim {
	case gen.DimensionNether:
		return c.genNether
	case gen.DimensionEnd:
		return c.genEnd
	default:
		return c.genOverworld
	}
}

// SwitchDimension transitions client world and chunks to target dimension.
func (c *Client) SwitchDimension(dim gen.Dimension) {
	if c.engine == nil || c.currentDim == dim {
		return
	}
	oldDimension := int(c.currentDim)
	engine := c.engine
	oldChunks := c.switchDimensionAndTakeChunks(engine, int(dim))
	c.meshMu.RLock()
	streamGeneration := c.meshGeneration
	c.meshMu.RUnlock()
	unloadJobs := make([]chunkStreamJob, 0, len(oldChunks))
	for _, cp := range oldChunks {
		unloadJobs = append(unloadJobs, chunkStreamJob{pos: cp, generation: streamGeneration, dimension: oldDimension, engine: engine, unload: true})
	}
	c.enqueueChunkStreamJobs(unloadJobs)

	var newPos sim.Vec3
	switch dim {
	case gen.DimensionNether:
		c.UnlockAdvancement("minecraft:nether/root", "进入下界")
		newPos = sim.Vec3{X: 0.5, Y: 66.0, Z: 0.5}
		c.AddChatMessage("[Dimension] Entered the Nether. Beware the lava seas!")
	case gen.DimensionEnd:
		c.UnlockAdvancement("minecraft:story/enter_the_end", "进入末地")
		newPos = sim.Vec3{X: 0.5, Y: 62.0, Z: 0.5}
		c.AddChatMessage("[Dimension] Entered The End. Slay the Ender Dragon to open the exit portal!")
	default:
		newPos = c.player.SpawnPos
		if newPos.X == 0 && newPos.Y == 0 && newPos.Z == 0 {
			newPos = sim.Vec3{X: 8.5, Y: 66.0, Z: 8.5}
		}
		c.AddChatMessage("[Dimension] Returned to the Overworld.")
	}

	c.player.Position = newPos
	c.player.Velocity = sim.Vec3{}
	c.publishPlayerSnapshot()
	c.camera.X = newPos.X
	c.camera.Y = newPos.Y + 1.62
	c.camera.Z = newPos.Z

	// Let the regular loop progressively stream the rest of the destination dimension.
	c.EnsureChunksAroundPlayer(newPos.X, newPos.Z, 3)
}

func (c *Client) switchDimensionAndTakeChunks(engine *sim.Engine, dimension int) []world.ChunkPos {
	c.meshMu.Lock()
	defer c.meshMu.Unlock()
	chunks := make([]world.ChunkPos, 0, len(c.loadedChunks))
	for pos := range c.loadedChunks {
		chunks = append(chunks, pos)
	}
	engine.SwitchDimension(dimension)
	c.currentDim = gen.Dimension(dimension)
	c.meshGeneration++
	c.meshCache = make(map[[3]int32]*render.Mesh)
	c.meshRevisions = make(map[[3]int32]uint64)
	c.loadedChunks = make(map[world.ChunkPos]bool)
	return chunks
}

func (c *Client) chunkStreamPending(pos world.ChunkPos, dimension int, generation uint64) bool {
	c.streamJobsMu.Lock()
	defer c.streamJobsMu.Unlock()
	now := time.Now()
	if c.streamJobsPrunedAt.IsZero() || now.Sub(c.streamJobsPrunedAt) >= 10*time.Second {
		for key, failure := range c.streamJobsFailed {
			if failure.generation != generation || !now.Before(failure.retryAfter) {
				delete(c.streamJobsFailed, key)
			}
		}
		c.streamJobsPrunedAt = now
	}
	key := chunkStreamKey{dimension: dimension, pos: pos}
	if failure, exists := c.streamJobsFailed[key]; exists {
		if failure.generation == generation && now.Before(failure.retryAfter) {
			return true
		}
		delete(c.streamJobsFailed, key)
	}
	if job, exists := c.streamJobsPending[key]; exists && job.generation == generation {
		return true
	}
	job, exists := c.streamJobsRunning[key]
	return exists && job.generation == generation
}

func (c *Client) enqueueChunkStreamJobs(jobs []chunkStreamJob) {
	if len(jobs) == 0 {
		return
	}
	c.streamWorkerOnce.Do(func() {
		if c.streamJobsPending == nil {
			c.streamJobsPending = make(map[chunkStreamKey]chunkStreamJob)
		}
		if c.streamJobsRunning == nil {
			c.streamJobsRunning = make(map[chunkStreamKey]chunkStreamJob)
		}
		if c.streamJobsWake == nil {
			c.streamJobsWake = make(chan struct{}, 1)
		}
		if c.streamJobsIdle == nil {
			c.streamJobsIdle = make(chan struct{}, 1)
		}
		go c.chunkStreamLoop()
	})
	c.streamJobsMu.Lock()
	loadCount := 0
	for _, pending := range c.streamJobsPending {
		if !pending.unload {
			loadCount++
		}
	}
	for _, job := range jobs {
		key := chunkStreamKey{dimension: job.dimension, pos: job.pos}
		if existing, ok := c.streamJobsPending[key]; !ok && !job.unload {
			if loadCount >= 24 {
				continue
			}
			loadCount++
		} else if ok && existing.unload && !job.unload {
			if loadCount >= 24 {
				continue
			}
			loadCount++
		} else if ok && !existing.unload && job.unload {
			loadCount--
		}
		c.streamJobsPending[key] = job
	}
	c.streamJobsMu.Unlock()
	select {
	case c.streamJobsWake <- struct{}{}:
	default:
	}
}

func (c *Client) chunkStreamLoop() {
	for range c.streamJobsWake {
		for {
			c.streamJobsMu.Lock()
			batch := make([]chunkStreamJob, 0, len(c.streamJobsPending))
			for key, job := range c.streamJobsPending {
				batch = append(batch, job)
				c.streamJobsRunning[key] = job
				delete(c.streamJobsPending, key)
			}
			c.streamJobsActive += len(batch)
			c.streamJobsMu.Unlock()
			if len(batch) == 0 {
				break
			}
			sort.Slice(batch, func(i, j int) bool {
				if batch[i].unload != batch[j].unload {
					return !batch[i].unload
				}
				if batch[i].pos.X != batch[j].pos.X {
					return batch[i].pos.X < batch[j].pos.X
				}
				return batch[i].pos.Z < batch[j].pos.Z
			})
			for _, job := range batch {
				c.processChunkStreamJob(job)
				c.streamJobsMu.Lock()
				c.streamJobsActive--
				key := chunkStreamKey{dimension: job.dimension, pos: job.pos}
				if running, exists := c.streamJobsRunning[key]; exists && running.generation == job.generation && running.unload == job.unload {
					delete(c.streamJobsRunning, key)
				}
				if c.streamJobsActive == 0 && len(c.streamJobsPending) == 0 {
					select {
					case c.streamJobsIdle <- struct{}{}:
					default:
					}
				}
				c.streamJobsMu.Unlock()
			}
		}
	}
}

func (c *Client) processChunkStreamJob(job chunkStreamJob) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if !job.unload {
				c.suppressChunkStreamRetry(job)
			}
			stack := debug.Stack()
			msg := fmt.Sprintf("CLIENT CHUNK STREAM PANIC at %v (dimension %d): %v\n\nStack:\n%s", job.pos, job.dimension, recovered, stack)
			log.Print(msg)
			writeCrashLog(msg)
		}
	}()
	if job.engine == nil || (!job.unload && !c.meshGenerationIs(job.generation)) {
		return
	}
	if job.unload {
		if err := job.engine.UnloadChunkInDimension(job.dimension, job.pos); err != nil {
			log.Printf("[Client] Failed to save chunk %v while unloading; retaining it in memory: %v", job.pos, err)
			return
		}
		c.meshMu.Lock()
		if c.meshGeneration == job.generation && job.engine.GetDimension() == job.dimension {
			delete(c.loadedChunks, job.pos)
			if c.meshRevisions == nil {
				c.meshRevisions = make(map[[3]int32]uint64)
			}
			for sy := int32(world.MinSectionY); sy <= world.MaxSectionY; sy++ {
				key := [3]int32{job.pos.X, sy, job.pos.Z}
				c.meshRevisions[key]++
				delete(c.meshCache, key)
				for _, neighbor := range [][3]int32{{job.pos.X - 1, sy, job.pos.Z}, {job.pos.X + 1, sy, job.pos.Z}, {job.pos.X, sy, job.pos.Z - 1}, {job.pos.X, sy, job.pos.Z + 1}} {
					c.meshRevisions[neighbor]++
					delete(c.meshCache, neighbor)
				}
			}
		}
		c.meshMu.Unlock()
		c.pruneMeshRevisions()
		return
	}

	chunk, loadErr := job.engine.GetChunkInDimensionWithError(job.dimension, job.pos)
	if loadErr != nil {
		c.suppressChunkStreamRetry(job)
		log.Printf("[Client] Refusing to regenerate unreadable chunk %v in dimension %d: %v", job.pos, job.dimension, loadErr)
		return
	}
	if !c.meshGenerationIs(job.generation) {
		if job.engine.GetDimension() != job.dimension {
			if err := job.engine.UnloadChunkInDimension(job.dimension, job.pos); err != nil {
				log.Printf("[Client] Failed to save stale chunk %v in dimension %d: %v", job.pos, job.dimension, err)
			}
		}
		return
	}
	if chunk == nil {
		if job.generator == nil {
			return
		}
		chunk = generateChunkSafely(job.generator, job.pos)
		if chunk == nil {
			c.suppressChunkStreamRetry(job)
			return
		}
		if !c.meshGenerationIs(job.generation) {
			return
		}
		job.engine.LoadGeneratedChunkInDimension(job.dimension, job.pos, chunk)
	}
	c.streamJobsMu.Lock()
	delete(c.streamJobsFailed, chunkStreamKey{dimension: job.dimension, pos: job.pos})
	c.streamJobsMu.Unlock()
	c.meshMu.Lock()
	if c.meshGeneration == job.generation && job.engine.GetDimension() == job.dimension {
		if c.loadedChunks == nil {
			c.loadedChunks = make(map[world.ChunkPos]bool)
		}
		c.loadedChunks[job.pos] = true
		for sy := int32(world.MinSectionY); sy <= world.MaxSectionY; sy++ {
			key := [3]int32{job.pos.X, sy, job.pos.Z}
			c.meshRevisions[key]++
			delete(c.meshCache, key)
			for _, neighbor := range [][3]int32{{job.pos.X - 1, sy, job.pos.Z}, {job.pos.X + 1, sy, job.pos.Z}, {job.pos.X, sy, job.pos.Z - 1}, {job.pos.X, sy, job.pos.Z + 1}} {
				c.meshRevisions[neighbor]++
				delete(c.meshCache, neighbor)
			}
		}
	}
	c.meshMu.Unlock()
}

func (c *Client) suppressChunkStreamRetry(job chunkStreamJob) {
	c.streamJobsMu.Lock()
	if c.streamJobsFailed == nil {
		c.streamJobsFailed = make(map[chunkStreamKey]chunkStreamFailure)
	}
	c.streamJobsFailed[chunkStreamKey{dimension: job.dimension, pos: job.pos}] = chunkStreamFailure{
		generation: job.generation,
		retryAfter: time.Now().Add(30 * time.Second),
	}
	c.streamJobsMu.Unlock()
}

func (c *Client) waitForChunkStreamIdle() {
	for {
		c.streamJobsMu.Lock()
		idle := c.streamJobsActive == 0 && len(c.streamJobsPending) == 0
		c.streamJobsMu.Unlock()
		if idle {
			return
		}
		select {
		case <-c.streamJobsIdle:
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func generateChunkSafely(generator *gen.Generator, pos world.ChunkPos) (chunk *world.Chunk) {
	defer func() {
		if recovered := recover(); recovered != nil {
			stack := debug.Stack()
			msg := fmt.Sprintf("CLIENT CHUNK GENERATION PANIC at %v: %v\n\nStack:\n%s", pos, recovered, stack)
			log.Print(msg)
			writeCrashLog(msg)
			chunk = nil
		}
	}()
	return generator.GenerateChunk(pos)
}

func (c *Client) meshGenerationIs(generation uint64) bool {
	c.meshMu.RLock()
	defer c.meshMu.RUnlock()
	return c.meshGeneration == generation
}

func (c *Client) meshJobPending(key [3]int32, generation, revision uint64) bool {
	c.meshJobsMu.Lock()
	defer c.meshJobsMu.Unlock()
	if job, exists := c.meshJobsPending[key]; exists && job.generation == generation && job.revision == revision {
		return true
	}
	job, exists := c.meshJobsRunning[key]
	return exists && job.generation == generation && job.revision == revision
}

// EnsureChunksAroundPlayer schedules world streaming and progressively meshes sections around the player.
func (c *Client) EnsureChunksAroundPlayer(playerX, playerZ float64, radius int32) {
	generator := c.getGenerator()
	if generator == nil || c.engine == nil {
		return
	}
	pcx := int32(math.Floor(playerX / 16.0))
	pcz := int32(math.Floor(playerZ / 16.0))
	engine := c.engine
	dimension := engine.GetDimension()

	// Pass 0: Prune distant meshes and chunks beyond the active streaming radius.
	c.meshMu.Lock()
	for key := range c.meshCache {
		diffX := key[0] - pcx
		if diffX < 0 {
			diffX = -diffX
		}
		diffZ := key[2] - pcz
		if diffZ < 0 {
			diffZ = -diffZ
		}
		if diffX > radius+1 || diffZ > radius+1 {
			delete(c.meshCache, key)
		}
	}
	var distantChunks []world.ChunkPos
	for cp := range c.loadedChunks {
		diffX := cp.X - pcx
		if diffX < 0 {
			diffX = -diffX
		}
		diffZ := cp.Z - pcz
		if diffZ < 0 {
			diffZ = -diffZ
		}
		if diffX > radius+2 || diffZ > radius+2 {
			distantChunks = append(distantChunks, cp)
		}
	}
	c.meshMu.Unlock()
	streamJobs := make([]chunkStreamJob, 0, len(distantChunks)+2)
	c.meshMu.RLock()
	generation := c.meshGeneration
	c.meshMu.RUnlock()
	for _, cp := range distantChunks {
		streamJobs = append(streamJobs, chunkStreamJob{pos: cp, generation: generation, dimension: dimension, engine: engine, unload: true})
	}

	// Pass 1: Queue at most two missing chunks per call; disk reads, generation,
	// and save-on-unload all run away from the window/event thread.
	queuedCount := 0
	for dx := -radius - 1; dx <= radius+1; dx++ {
		for dz := -radius - 1; dz <= radius+1; dz++ {
			cp := world.ChunkPos{X: pcx + dx, Z: pcz + dz}
			c.meshMu.Lock()
			alreadyLoaded := c.loadedChunks[cp]
			c.meshMu.Unlock()
			if !alreadyLoaded {
				if c.chunkStreamPending(cp, dimension, generation) {
					continue
				}
				streamJobs = append(streamJobs, chunkStreamJob{pos: cp, generation: generation, dimension: dimension, engine: engine, generator: generator})
				queuedCount++
				if queuedCount >= 2 {
					break
				}
			}
		}
		if queuedCount >= 2 {
			break
		}
	}
	c.enqueueChunkStreamJobs(streamJobs)

	// Pass 2: Progressive section meshing (build at most 2 sections per frame)
	// Prioritize sections closest to player camera (covering Y=-64 up to Y=256)
	camSY := int32(math.Floor(c.camera.Y / 16.0))
	c.meshMu.RLock()
	unmeshed := make([]sectionMeshJob, 0, 2)
	orderedSY := sectionBuildOrder(camSY)

scanMeshes:
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			cp := world.ChunkPos{X: pcx + dx, Z: pcz + dz}
			for _, sy := range orderedSY {
				key := [3]int32{cp.X, sy, cp.Z}
				if c.meshCache[key] == nil {
					job := sectionMeshJob{
						key:        key,
						generation: c.meshGeneration,
						revision:   c.meshRevisions[key],
						engine:     engine,
					}
					if !c.meshJobPending(key, job.generation, job.revision) {
						unmeshed = append(unmeshed, job)
						if len(unmeshed) == cap(unmeshed) {
							break scanMeshes
						}
					}
				}
			}
		}
	}
	c.meshMu.RUnlock()

	if len(unmeshed) == 0 {
		return
	}

	c.enqueueMeshJobs(unmeshed)
}

func sectionBuildOrder(cameraSectionY int32) []int32 {
	const priorityRadius = 4
	ordered := make([]int32, 0, world.NumSections)
	seen := make(map[int32]struct{}, world.NumSections)
	appendSection := func(sectionY int32) {
		if sectionY < world.MinSectionY || sectionY > world.MaxSectionY {
			return
		}
		if _, exists := seen[sectionY]; exists {
			return
		}
		seen[sectionY] = struct{}{}
		ordered = append(ordered, sectionY)
	}

	appendSection(cameraSectionY)
	for distance := int32(1); distance <= priorityRadius; distance++ {
		appendSection(cameraSectionY + distance)
		appendSection(cameraSectionY - distance)
	}
	for sectionY := int32(world.MinSectionY); sectionY <= world.MaxSectionY; sectionY++ {
		appendSection(sectionY)
	}
	return ordered
}

// InvalidateBlockMesh purges cached section mesh containing pos and affected neighbor boundary sections.
func (c *Client) InvalidateBlockMesh(pos world.BlockPos) {
	c.meshMu.Lock()
	defer c.meshMu.Unlock()
	if c.meshRevisions == nil {
		c.meshRevisions = make(map[[3]int32]uint64)
	}
	for _, key := range affectedMeshSections(pos) {
		c.meshRevisions[key]++
		delete(c.meshCache, key)
	}
}

// RebuildAffectedMeshes queues the affected sections for background rebuild.
func (c *Client) RebuildAffectedMeshes(pos world.BlockPos) {
	if c.engine == nil {
		return
	}
	engine := c.engine
	keys := affectedMeshSections(pos)
	jobs := make([]sectionMeshJob, len(keys))
	c.meshMu.Lock()
	if c.meshRevisions == nil {
		c.meshRevisions = make(map[[3]int32]uint64)
	}
	for i, key := range keys {
		c.meshRevisions[key]++
		jobs[i] = sectionMeshJob{key: key, generation: c.meshGeneration, revision: c.meshRevisions[key], engine: engine}
	}
	c.meshMu.Unlock()
	c.enqueueMeshJobs(jobs)
}

func (c *Client) enqueueMeshJobs(jobs []sectionMeshJob) {
	if len(jobs) == 0 {
		return
	}
	c.meshWorkerOnce.Do(func() {
		if c.meshJobsPending == nil {
			c.meshJobsPending = make(map[[3]int32]sectionMeshJob)
		}
		if c.meshJobsRunning == nil {
			c.meshJobsRunning = make(map[[3]int32]sectionMeshJob)
		}
		if c.meshJobsWake == nil {
			c.meshJobsWake = make(chan struct{}, 1)
		}
		go c.meshUpdateLoop()
	})
	c.meshJobsMu.Lock()
	for _, job := range jobs {
		if _, exists := c.meshJobsPending[job.key]; !exists && len(c.meshJobsPending) >= 128 {
			continue
		}
		c.meshJobsPending[job.key] = job
	}
	c.meshJobsMu.Unlock()
	select {
	case c.meshJobsWake <- struct{}{}:
	default:
	}
}

func (c *Client) meshUpdateLoop() {
	for range c.meshJobsWake {
		for {
			c.meshJobsMu.Lock()
			batch := make([]sectionMeshJob, 0, len(c.meshJobsPending))
			for key, job := range c.meshJobsPending {
				batch = append(batch, job)
				c.meshJobsRunning[key] = job
				delete(c.meshJobsPending, key)
			}
			c.meshJobsMu.Unlock()
			if len(batch) == 0 {
				break
			}
			sort.Slice(batch, func(i, j int) bool {
				if batch[i].key[1] != batch[j].key[1] {
					return batch[i].key[1] < batch[j].key[1]
				}
				if batch[i].key[2] != batch[j].key[2] {
					return batch[i].key[2] < batch[j].key[2]
				}
				return batch[i].key[0] < batch[j].key[0]
			})
			for _, job := range batch {
				c.processMeshJob(job)
			}
		}
	}
}

func (c *Client) processMeshJob(job sectionMeshJob) {
	defer func() {
		if recovered := recover(); recovered != nil {
			stack := debug.Stack()
			msg := fmt.Sprintf("CLIENT MESH WORKER PANIC at section %v: %v\n\nStack:\n%s", job.key, recovered, stack)
			log.Print(msg)
			writeCrashLog(msg)
		}
		c.meshJobsMu.Lock()
		if running, exists := c.meshJobsRunning[job.key]; exists && running.generation == job.generation && running.revision == job.revision {
			delete(c.meshJobsRunning, job.key)
		}
		c.meshJobsMu.Unlock()
		c.pruneMeshRevisions()
	}()
	c.meshMu.RLock()
	current := c.meshGeneration == job.generation && c.meshRevisions[job.key] == job.revision
	c.meshMu.RUnlock()
	if !current || job.engine == nil {
		return
	}
	mesh := buildSectionMeshSafely(job.engine, job.key)
	c.meshMu.Lock()
	if c.meshGeneration == job.generation && c.meshRevisions[job.key] == job.revision {
		c.meshCache[job.key] = mesh
	}
	c.meshMu.Unlock()
}

// pruneMeshRevisions keeps section version metadata bounded to loaded chunks
// and jobs that still need an invalidation token.
func (c *Client) pruneMeshRevisions() {
	c.meshMu.Lock()
	defer c.meshMu.Unlock()
	c.meshJobsMu.Lock()
	defer c.meshJobsMu.Unlock()
	for key := range c.meshRevisions {
		pos := world.ChunkPos{X: key[0], Z: key[2]}
		if c.loadedChunks[pos] {
			continue
		}
		if _, pending := c.meshJobsPending[key]; pending {
			continue
		}
		if _, running := c.meshJobsRunning[key]; running {
			continue
		}
		delete(c.meshRevisions, key)
		delete(c.meshCache, key)
	}
}

func affectedMeshSections(pos world.BlockPos) [][3]int32 {
	secX := int32(math.Floor(float64(pos.X) / 16.0))
	secY := int32(math.Floor(float64(pos.Y) / 16.0))
	secZ := int32(math.Floor(float64(pos.Z) / 16.0))
	keys := [][3]int32{{secX, secY, secZ}}
	remX := ((pos.X % 16) + 16) % 16
	remY := ((pos.Y % 16) + 16) % 16
	remZ := ((pos.Z % 16) + 16) % 16
	if remX == 0 {
		keys = append(keys, [3]int32{secX - 1, secY, secZ})
	} else if remX == 15 {
		keys = append(keys, [3]int32{secX + 1, secY, secZ})
	}
	if remY == 0 {
		keys = append(keys, [3]int32{secX, secY - 1, secZ})
	} else if remY == 15 {
		keys = append(keys, [3]int32{secX, secY + 1, secZ})
	}
	if remZ == 0 {
		keys = append(keys, [3]int32{secX, secY, secZ - 1})
	} else if remZ == 15 {
		keys = append(keys, [3]int32{secX, secY, secZ + 1})
	}
	return keys
}

func (c *Client) resetMeshCache(resetLoadedChunks bool) {
	c.meshMu.Lock()
	c.meshGeneration++
	c.meshCache = make(map[[3]int32]*render.Mesh)
	c.meshRevisions = make(map[[3]int32]uint64)
	if resetLoadedChunks {
		c.loadedChunks = make(map[world.ChunkPos]bool)
	}
	c.meshMu.Unlock()
	c.streamJobsMu.Lock()
	clear(c.streamJobsFailed)
	c.streamJobsPrunedAt = time.Time{}
	c.streamJobsMu.Unlock()
}

func (c *Client) RebuildMeshes() {
	if c.player != nil {
		c.resetMeshCache(false)

		c.lastChunkX = -999
		c.EnsureChunksAroundPlayer(c.player.Position.X, c.player.Position.Z, 3)
	}
}

func (c *Client) Connect() error {
	var err error
	c.conn, err = net.DialTimeout("tcp", c.serverAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.serverAddr, err)
	}
	connected := false
	defer func() {
		if !connected && c.conn != nil {
			_ = c.conn.Close()
		}
	}()
	if err := c.conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		_ = c.conn.Close()
		return fmt.Errorf("set login deadline: %w", err)
	}
	log.Printf("[Client] Connected to server at %s", c.serverAddr)

	hs := &protocol.HandshakePacket{
		ProtocolVersion: protocol.ProtocolVersion,
		ServerAddress:   "localhost",
		ServerPort:      25565,
		NextState:       2,
	}
	if err := protocol.SendPacket(c.conn, hs); err != nil {
		return err
	}

	c.state = protocol.StateLogin
	ls := &protocol.LoginStartPacket{
		Username:   c.username,
		PlayerUUID: c.uuid,
	}
	if err := protocol.SendPacket(c.conn, ls); err != nil {
		return err
	}

	raw, err := protocol.ReadRawPacket(c.conn)
	if err != nil {
		return err
	}
	var succ protocol.LoginSuccessPacket
	if err := succ.Decode(raw.Reader()); err != nil {
		return err
	}
	if err := c.conn.SetDeadline(time.Time{}); err != nil {
		_ = c.conn.Close()
		return fmt.Errorf("clear login deadline: %w", err)
	}
	c.state = protocol.StatePlay
	log.Printf("[Client] Logged in successfully as %s", succ.Username)

	// Initialize local client world replica and authoritative player entity
	c.engine = sim.NewEngine()
	spawnPos := world.ChunkPos{X: 0, Z: 0}
	spawnChunk := world.NewChunk(spawnPos)
	for x := int32(0); x < 16; x++ {
		for z := int32(0); z < 16; z++ {
			spawnChunk.SetBlock(x, -64, z, 1)
			for y := int32(-63); y <= 60; y++ {
				spawnChunk.SetBlock(x, y, z, 1)
			}
			spawnChunk.SetBlock(x, 61, z, 3)
			spawnChunk.SetBlock(x, 62, z, 3)
			spawnChunk.SetBlock(x, 63, z, 3)
			spawnChunk.SetBlock(x, 64, z, 2)
		}
	}
	c.engine.LoadChunk(spawnPos, spawnChunk)

	spawnVec := sim.Vec3{X: 8.5, Y: 65.0, Z: 8.5}
	c.player = c.engine.AddPlayer(c.username, c.uuid, spawnVec)
	c.engine.SetPlayerClientDriven(c.player.ID, true)
	c.camera.X = spawnVec.X
	c.camera.Y = spawnVec.Y + 1.62
	c.camera.Z = spawnVec.Z

	// Classic starter items in hotbar
	c.player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemWoodenPickaxe, Count: 1}
	c.player.Inventory.Slots[1] = inventory.ItemStack{ItemID: inventory.ItemOakPlanks, Count: 64}
	c.player.Inventory.Slots[2] = inventory.ItemStack{ItemID: inventory.ItemCobblestone, Count: 64}
	c.player.Inventory.Slots[3] = inventory.ItemStack{ItemID: inventory.ItemDirt, Count: 64}
	c.player.Inventory.Slots[4] = inventory.ItemStack{ItemID: 268, Count: 1} // Wooden Sword
	c.player.Inventory.Slots[5] = inventory.ItemStack{ItemID: 271, Count: 1} // Wooden Axe
	c.player.Inventory.Slots[6] = inventory.ItemStack{ItemID: 269, Count: 1} // Wooden Shovel
	c.player.Inventory.Slots[7] = inventory.ItemStack{ItemID: inventory.ItemCookedPorkchop, Count: 16}
	c.player.Inventory.Slots[8] = inventory.ItemStack{ItemID: inventory.ItemApple, Count: 16}
	c.player.Inventory.SelectedHotbar = 1

	// Launch background network receive loop
	c.networkDone = make(chan struct{})
	c.networkSendQueue = make(chan protocol.Packet, 256)
	c.networkSendDone = make(chan struct{})
	c.networkSendStop = make(chan struct{})
	go c.networkSendLoop()
	go c.networkReceiveLoop()
	connected = true

	return nil
}

func (c *Client) networkReceiveLoop() {
	done := c.networkDone
	if done != nil {
		defer close(done)
	}
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			msg := fmt.Sprintf("FATAL NETWORK RECEIVE PANIC: %v\n\nStack:\n%s\n", r, string(stack))
			log.Println(msg)
			writeCrashLog(msg)
			c.networkDisconnected.Store(true)
			c.stopNetworkSender()
		}
	}()
	for {
		if c.conn == nil {
			return
		}
		raw, err := protocol.ReadRawPacket(c.conn)
		if err != nil {
			log.Printf("[Client] Network connection closed: %v", err)
			if !c.quitSaving.Load() {
				c.networkDisconnected.Store(true)
			}
			c.stopNetworkSender()
			return
		}

		switch raw.ID {
		case 0x09: // BlockUpdatePacket
			var bu protocol.BlockUpdatePacket
			if err := bu.Decode(raw.Reader()); err == nil && world.ValidBlockCoordinates(bu.X, bu.Y, bu.Z) && bu.BlockID >= 0 {
				if c.engine != nil {
					if c.engine.SetBlockChecked(bu.X, bu.Y, bu.Z, bu.BlockID) {
						c.RebuildAffectedMeshes(world.BlockPos{X: bu.X, Y: bu.Y, Z: bu.Z})
					}
				}
			}
		case 0x1A: // PlayerPositionPacket
			var pos protocol.PlayerPositionPacket
			if err := pos.Decode(raw.Reader()); err != nil {
				log.Printf("[Client] Invalid remote player position packet: %v", err)
				c.networkDisconnected.Store(true)
				c.stopNetworkSender()
				return
			}
			if !validRemotePlayerPosition(pos) {
				log.Printf("[Client] Closing connection after invalid remote player position: %+v", pos)
				c.networkDisconnected.Store(true)
				c.stopNetworkSender()
				return
			}
			c.remoteMu.Lock()
			c.lastRemotePos = sim.Vec3{X: pos.X, Y: pos.Y, Z: pos.Z}
			c.remotePlayers["Remote"] = &RemotePlayer{
				Pos:   c.lastRemotePos,
				Yaw:   pos.Yaw,
				Pitch: pos.Pitch,
			}
			c.remoteMu.Unlock()
		case 0x26: // KeepAlivePacket
			var ka protocol.KeepAlivePacket
			if err := ka.Decode(raw.Reader()); err == nil {
				_ = c.sendPacket(&ka)
			}
		}
	}
}

// sendPacket queues gameplay traffic so a slow or stalled server cannot block
// the graphical thread while it writes to the socket.
func (c *Client) sendPacket(pkt protocol.Packet) error {
	if c.networkSendQueue == nil || c.networkSendStop == nil {
		return net.ErrClosed
	}
	select {
	case <-c.networkSendStop:
		return net.ErrClosed
	default:
	}
	select {
	case <-c.networkSendStop:
		return net.ErrClosed
	case c.networkSendQueue <- pkt:
		return nil
	default:
		err := fmt.Errorf("client outbound packet queue is full")
		log.Printf("[Client] Disconnecting after outbound queue overflow")
		c.networkDisconnected.Store(true)
		c.stopNetworkSender()
		return err
	}
}

func (c *Client) networkSendLoop() {
	defer close(c.networkSendDone)
	defer func() {
		if recovered := recover(); recovered != nil {
			msg := fmt.Sprintf("FATAL NETWORK SEND PANIC: %v\n\nStack:\n%s", recovered, debug.Stack())
			log.Print(msg)
			writeCrashLog(msg)
			c.networkDisconnected.Store(true)
			c.stopNetworkSender()
		}
	}()
	for {
		select {
		case <-c.networkSendStop:
			return
		case pkt := <-c.networkSendQueue:
			if err := c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				log.Printf("[Client] Failed to set network write deadline: %v", err)
				c.networkDisconnected.Store(true)
				c.stopNetworkSender()
				return
			}
			if err := protocol.SendPacket(c.conn, pkt); err != nil {
				log.Printf("[Client] Network packet write failed: %v", err)
				writeCrashLog(fmt.Sprintf("NETWORK SEND FAILED: %v", err))
				c.networkDisconnected.Store(true)
				c.stopNetworkSender()
				return
			}
			if err := c.conn.SetWriteDeadline(time.Time{}); err != nil {
				log.Printf("[Client] Failed to clear network write deadline: %v", err)
				c.networkDisconnected.Store(true)
				c.stopNetworkSender()
				return
			}
		}
	}
}

func (c *Client) stopNetworkSender() {
	c.networkSendStopOnce.Do(func() {
		if c.networkSendStop != nil {
			close(c.networkSendStop)
		}
		if c.conn != nil {
			_ = c.conn.Close()
		}
	})
}

// checkWaterState examines the player's bounding volume to determine if they are in water,
// and whether they are positioned near the water surface.
func (c *Client) checkWaterState() (inWater bool, atSurface bool) {
	if c.player == nil || c.engine == nil {
		return false, false
	}
	px := c.player.Position.X
	py := c.player.P…18749 tokens truncated…3]]

		uv := faceUVs[fIdx]
		if tex != nil && (uv.U1 > uv.U0 && uv.V1 > uv.V0) {
			c.framebuf.DrawTexturedTriangle(p0, p1, p2, uv.U0, uv.V0, uv.U1, uv.V0, uv.U1, uv.V1, tex, tint)
			c.framebuf.DrawTexturedTriangle(p0, p2, p3, uv.U0, uv.V0, uv.U1, uv.V1, uv.U0, uv.V1, tex, tint)
		} else {
			col := fallbackCols[fIdx]
			if tint != 0 {
				col = tint
			}
			c.framebuf.DrawTriangle(p0, p1, p2, col)
			c.framebuf.DrawTriangle(p0, p2, p3, col)
		}
	}
}

// render3DBox renders a volumetric 3D cuboid transformed by position, yaw, and pitch with depth testing.
func (c *Client) render3DBox(cx, cy, cz, hx, hy, hz, yawDeg, pitchDeg float64, colTop, colBottom, colSide uint32) {
	c.renderTextured3DBox(cx, cy, cz, hx, hy, hz, yawDeg, pitchDeg, nil, [6]BoxFaceUV{}, [6]uint32{colTop, colBottom, render.ApplyLighting(colSide, 0.85), render.ApplyLighting(colSide, 0.85), render.ApplyLighting(colSide, 0.70), render.ApplyLighting(colSide, 0.70)}, 0)
}

// drawPlayerModel renders the authentic volumetric 3D Steve character in third-person view.
func (c *Client) drawPlayerModel() {
	if c.player == nil {
		return
	}
	pos := c.player.Position

	// Steve color palette
	headSkin := uint32(0xFFD6A07E)
	headHair := uint32(0xFF4D3B26)
	torsoCol := uint32(0xFF2E8690) // Classic Cyan shirt
	pantsCol := uint32(0xFF2A3663) // Indigo pants
	shoesCol := uint32(0xFF555555) // Gray shoes

	if c.hurtFlashTimer > 0 {
		headSkin = 0xFFFF4444
		headHair = 0xFFFF2222
		torsoCol = 0xFFFF3333
		pantsCol = 0xFFFF2222
	}

	yaw := c.camera.Yaw
	pitch := c.camera.Pitch

	walkSwing := 0.0
	if c.isMoving && c.player.OnGround {
		walkSwing = math.Sin(c.walkCycle) * 28.0
	}

	armSwing := 0.0
	if c.isSwinging {
		armSwing = math.Sin(math.Sqrt(c.swingProgress)*math.Pi) * 65.0
	}

	yOffset := 0.0
	torsoPitch := 0.0
	if c.isSneaking {
		yOffset = -0.20
		torsoPitch = 12.0
	}

	// 1. Head (rotates with camera yaw & pitch; 0.5x0.5x0.5m Minecraft Java scale)
	c.render3DBox(pos.X, pos.Y+1.67+yOffset, pos.Z, 0.25, 0.25, 0.25, yaw, pitch, headHair, headSkin, headSkin)

	// 2. Torso (0.5m wide x 0.75m tall x 0.25m deep; rotates with body yaw & sneak pitch)
	c.render3DBox(pos.X, pos.Y+1.125+yOffset, pos.Z, 0.25, 0.375, 0.125, yaw, torsoPitch, torsoCol, torsoCol, torsoCol)

	// Compute shoulder offsets relative to body yaw (0.375m center-to-arm)
	radY := yaw * math.Pi / 180.0
	cosY, sinY := math.Cos(radY), math.Sin(radY)
	armOffX := 0.375 * cosY
	armOffZ := -0.375 * sinY

	// 3. Left Arm (0.25x0.75x0.25m; swings opposite)
	c.render3DBox(pos.X-armOffX, pos.Y+1.125+yOffset, pos.Z-armOffZ, 0.125, 0.375, 0.125, yaw, walkSwing, torsoCol, headSkin, torsoCol)

	// 4. Right Arm (0.25x0.75x0.25m; swings with walk cycle + chopping attack arc)
	c.render3DBox(pos.X+armOffX, pos.Y+1.125+yOffset, pos.Z+armOffZ, 0.125, 0.375, 0.125, yaw, -walkSwing+armSwing, torsoCol, headSkin, torsoCol)

	// Hip offsets relative to body yaw (0.125m center-to-leg)
	legOffX := 0.125 * cosY
	legOffZ := -0.125 * sinY

	// 5. Left Leg (0.25x0.75x0.25m; swings forward/backward)
	c.render3DBox(pos.X-legOffX, pos.Y+0.375, pos.Z-legOffZ, 0.125, 0.375, 0.125, yaw, -walkSwing, pantsCol, shoesCol, pantsCol)

	// 6. Right Leg (0.25x0.75x0.25m; swings opposite)
	c.render3DBox(pos.X+legOffX, pos.Y+0.375, pos.Z+legOffZ, 0.125, 0.375, 0.125, yaw, walkSwing, pantsCol, shoesCol, pantsCol)

	// 7. Render Held Item/Block in Steve's right hand in 3D
	if c.player.Inventory != nil {
		sel := c.player.Inventory.GetSelectedItem()
		if !sel.IsEmpty() {
			rightArmPitch := (-walkSwing + armSwing) * math.Pi / 180.0
			handForward := math.Sin(rightArmPitch) * 0.375
			handDown := math.Cos(rightArmPitch) * 0.375

			hx := pos.X + armOffX + math.Sin(radY)*handForward
			hy := pos.Y + 1.125 + yOffset - handDown
			hz := pos.Z + armOffZ + math.Cos(radY)*handForward

			if inventory.IsPlaceableBlock(sel.ItemID) {
				blockCol := render.SampleBlockColor(sel.ItemID, 2, 0.5, 0.5, 1.0)
				c.render3DBox(hx, hy, hz, 0.14, 0.14, 0.14, yaw, 0, blockCol, blockCol, blockCol)
			} else {
				toolCol := uint32(0xFFB8945F)
				if props, ok := inventory.GetToolProperties(sel.ItemID); ok {
					switch props.Tier {
					case inventory.TierStone:
						toolCol = 0xFF7D7D7D
					case inventory.TierIron:
						toolCol = 0xFFD8D8D8
					case inventory.TierDiamond:
						toolCol = 0xFF2CCCCC
					}
				}
				c.render3DBox(hx, hy, hz, 0.08, 0.20, 0.08, yaw, -walkSwing+armSwing, toolCol, 0xFF6F4F37, toolCol)
			}
		}
	}
}

// DropHeldItem throws 1 item (or full stack if dropAll is true) from active hotbar slot into the world.
func (c *Client) DropHeldItem(dropAll bool) {
	if c.player == nil || c.engine == nil || c.player.Inventory == nil {
		return
	}
	sel := c.player.Inventory.GetSelectedItem()
	if sel.IsEmpty() {
		return
	}

	dropCount := 1
	if dropAll || sel.Count <= 1 {
		dropCount = sel.Count
	}

	_ = c.player.Inventory.RemoveItem(c.player.Inventory.SelectedHotbar, dropCount)

	lx, ly, lz := c.camera.GetLookVector()
	spawnPos := sim.Vec3{
		X: c.camera.X + lx*0.5,
		Y: c.camera.Y + ly*0.5 - 0.1,
		Z: c.camera.Z + lz*0.5,
	}
	vel := sim.Vec3{
		X: lx * 0.35,
		Y: ly*0.35 + 0.18,
		Z: lz * 0.35,
	}

	c.engine.SpawnItemDropWithDelay(sel.ItemID, dropCount, spawnPos, vel, 1.2)
	sound.PlayHardwareFeedback(sound.SoundBlockWoodStep)
	c.isSwinging = true
	c.swingProgress = 0.0
}

func (c *Client) AttackMob() bool {
	if c.player == nil || c.engine == nil || c.player.Inventory == nil {
		return false
	}
	dmg := float32(1.0) // Bare fist base damage
	if sel := c.player.Inventory.GetSelectedItem(); !sel.IsEmpty() {
		if props, ok := inventory.GetToolProperties(sel.ItemID); ok {
			dmg = props.AttackDamage
		}
	}

	camPos := sim.Vec3{X: c.camera.X, Y: c.camera.Y, Z: c.camera.Z}
	lx, ly, lz := c.camera.GetLookVector()
	look := sim.Vec3{X: lx, Y: ly, Z: lz}

	mobID, killed, hit := c.engine.AttackMobInFront(camPos, look, 3.5, dmg)
	if hit {
		c.isSwinging = true
		c.swingProgress = 0.0
		sound.PlayHardwareFeedback("minecraft:entity.generic.hurt")

		// Emit red blood / strike particles
		for i := 0; i < 8; i++ {
			rx := (randFloat() - 0.5) * 0.5
			ry := (randFloat() - 0.5) * 0.5
			rz := (randFloat() - 0.5) * 0.5
			c.particles = append(c.particles, render.Particle{
				X:       c.camera.X + lx*2.0 + rx,
				Y:       c.camera.Y + ly*2.0 + ry,
				Z:       c.camera.Z + lz*2.0 + rz,
				Vx:      (randFloat() - 0.5) * 2.0,
				Vy:      randFloat()*2.0 + 1.0,
				Vz:      (randFloat() - 0.5) * 2.0,
				Color:   0xFFFF2222,
				Life:    0.3,
				MaxLife: 0.3,
			})
		}

		if killed {
			sound.PlayHardwareFeedback("minecraft:entity.generic.death")
			// Mob death puff particles
			for i := 0; i < 16; i++ {
				rx := (randFloat() - 0.5) * 0.8
				ry := (randFloat() - 0.5) * 0.8
				rz := (randFloat() - 0.5) * 0.8
				c.particles = append(c.particles, render.Particle{
					X:       c.camera.X + lx*2.0 + rx,
					Y:       c.camera.Y + ly*2.0 + ry,
					Z:       c.camera.Z + lz*2.0 + rz,
					Vx:      (randFloat() - 0.5) * 3.0,
					Vy:      randFloat()*3.0 + 1.0,
					Vz:      (randFloat() - 0.5) * 3.0,
					Color:   0xFFEEEEEE, // Smoke white
					Life:    0.5,
					MaxLife: 0.5,
				})
			}
			log.Printf("[Combat] Slew mob #%d!", mobID)
		}
		return true
	}
	return false
}

func (c *Client) TryQuickCraft(recipeIndex int) bool {
	if c.player == nil || c.player.Inventory == nil {
		return false
	}
	inv := c.player.Inventory

	findItem := func(id int32, minCount int) int {
		for i, s := range inv.Slots {
			if s.ItemID == id && s.Count >= minCount {
				return i
			}
		}
		return -1
	}

	switch recipeIndex {
	case 1: // 1 Log -> 4 Planks
		slot := findItem(inventory.ItemOakLog, 1)
		if slot >= 0 {
			_ = inv.RemoveItem(slot, 1)
			inv.AddItem(inventory.ItemOakPlanks, 4)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	case 2: // 2 Planks -> 4 Sticks (ID 280)
		slot := findItem(inventory.ItemOakPlanks, 2)
		if slot >= 0 {
			_ = inv.RemoveItem(slot, 2)
			inv.AddItem(280, 4)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	case 3: // 4 Planks -> Crafting Table
		slot := findItem(inventory.ItemOakPlanks, 4)
		if slot >= 0 {
			_ = inv.RemoveItem(slot, 4)
			inv.AddItem(inventory.ItemCraftingTable, 1)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	case 4: // 3 Planks + 2 Sticks -> Wooden Pickaxe
		pSlot := findItem(inventory.ItemOakPlanks, 3)
		sSlot := findItem(280, 2)
		if pSlot >= 0 && sSlot >= 0 {
			_ = inv.RemoveItem(pSlot, 3)
			_ = inv.RemoveItem(sSlot, 2)
			inv.AddItem(inventory.ItemWoodenPickaxe, 1)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	case 5: // 2 Planks + 1 Stick -> Wooden Sword
		pSlot := findItem(inventory.ItemOakPlanks, 2)
		sSlot := findItem(280, 1)
		if pSlot >= 0 && sSlot >= 0 {
			_ = inv.RemoveItem(pSlot, 2)
			_ = inv.RemoveItem(sSlot, 1)
			inv.AddItem(268, 1)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	case 6: // 3 Cobble + 2 Sticks -> Stone Pickaxe
		cSlot := findItem(inventory.ItemCobblestone, 3)
		sSlot := findItem(280, 2)
		if cSlot >= 0 && sSlot >= 0 {
			_ = inv.RemoveItem(cSlot, 3)
			_ = inv.RemoveItem(sSlot, 2)
			inv.AddItem(inventory.ItemStonePickaxe, 1)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			c.UnlockAdvancement("minecraft:story/upgrade_tools", "装配升级")
			return true
		}
	case 7: // 8 Cobble -> Furnace
		cSlot := findItem(inventory.ItemCobblestone, 8)
		if cSlot >= 0 {
			_ = inv.RemoveItem(cSlot, 8)
			inv.AddItem(inventory.ItemFurnace, 1)
			sound.PlayHardwareFeedback("minecraft:item.craft")
			return true
		}
	}
	return false
}

// drawItemDrop renders a rotating, bobbing 3D voxel item entity floating in the world.
func (c *Client) drawItemDrop(drop sim.ItemDrop) {
	bobY := math.Sin(drop.Age*3.5) * 0.08
	cx, cy, cz := drop.Position.X, drop.Position.Y+bobY+0.20, drop.Position.Z
	topCol := render.SampleBlockColor(drop.ItemID, 2, 0.5, 0.5, 1.0)
	sideCol := render.SampleBlockColor(drop.ItemID, 0, 0.5, 0.5, 0.80)
	darkCol := render.SampleBlockColor(drop.ItemID, 1, 0.5, 0.5, 0.60)

	// Authentic Minecraft item rotation: continuous 90 deg/s around Y axis
	spinYaw := drop.Age * 90.0
	c.render3DBox(cx, cy, cz, 0.12, 0.12, 0.12, spinYaw, 15.0, topCol, darkCol, sideCol)
}

func darkenColor(c uint32, factor float32) uint32 {
	a := (c >> 24) & 0xFF
	r := uint32(float32((c>>16)&0xFF) * factor)
	g := uint32(float32((c>>8)&0xFF) * factor)
	b := uint32(float32(c&0xFF) * factor)
	return (a << 24) | (r << 16) | (g << 8) | b
}

func getToolSprite16x16(toolType inventory.ToolType, tier inventory.ToolTier) [16][16]uint32 {
	var s [16][16]uint32
	stickCol := uint32(0xFF8F6849)
	stickShade := uint32(0xFF5C3E28)

	headCol := uint32(0xFFB8945F) // Wood
	shineCol := uint32(0xFFD4B07B)
	darkCol := uint32(0xFF8A6B3A)
	switch tier {
	case inventory.TierStone:
		headCol = 0xFF7D7D7D
		shineCol = 0xFF9E9E9E
		darkCol = 0xFF545454
	case inventory.TierIron:
		headCol = 0xFFD8D8D8
		shineCol = 0xFFFFFFFF
		darkCol = 0xFFA0A0A0
	case inventory.TierDiamond:
		headCol = 0xFF4AEDD7
		shineCol = 0xFF8CFFEE
		darkCol = 0xFF28A896
	}

	// Stick handle from (2, 2) to (9, 9)
	for d := 1; d <= 9; d++ {
		s[d][d] = stickCol
		if d > 1 {
			s[d-1][d] = stickShade
		}
	}

	switch toolType {
	case inventory.ToolPickaxe:
		// Pickaxe head arched from (6, 13) through (10, 10) to (13, 6)
		s[10][10] = darkCol
		s[11][10] = headCol
		s[10][11] = headCol
		s[11][11] = shineCol
		s[12][9] = headCol
		s[13][8] = headCol
		s[14][7] = headCol
		s[14][6] = shineCol
		s[15][5] = headCol
		s[9][12] = headCol
		s[8][13] = headCol
		s[7][14] = headCol
		s[6][14] = shineCol
		s[5][15] = headCol
		s[11][12] = shineCol
		s[12][11] = shineCol
	case inventory.ToolSword:
		// Double-edged blade from (5, 5) to (14, 14)
		s[1][1] = headCol // Pommel
		s[2][2] = stickCol
		s[3][3] = stickCol
		// Crossguard
		s[3][5] = darkCol
		s[4][4] = darkCol
		s[5][3] = darkCol
		s[4][5] = headCol
		s[5][4] = headCol
		s[2][5] = darkCol
		s[5][2] = darkCol
		// Blade
		for d := 5; d <= 13; d++ {
			s[d][d] = headCol
			s[d+1][d] = shineCol
			s[d][d+1] = darkCol
		}
		s[14][14] = shineCol // Blade tip
	case inventory.ToolAxe:
		// Broad axe blade
		s[10][10] = darkCol
		s[10][11] = headCol
		s[11][11] = shineCol
		s[11][12] = shineCol
		s[12][12] = headCol
		s[12][13] = shineCol
		s[13][13] = headCol
		s[10][12] = headCol
		s[9][12] = headCol
		s[9][11] = darkCol
		s[13][12] = headCol
		s[13][11] = darkCol
		s[12][10] = darkCol
	case inventory.ToolShovel:
		// Spade scoop
		s[10][10] = darkCol
		s[11][11] = headCol
		s[12][12] = shineCol
		s[13][13] = shineCol
		s[12][11] = headCol
		s[11][12] = headCol
		s[13][12] = shineCol
		s[12][13] = headCol
		s[14][13] = shineCol
		s[13][14] = shineCol
	}
	return s
}

func getTorchSprite16x16() [16][16]uint32 {
	var s [16][16]uint32
	// Handle: columns 7 and 8, rows 1 to 9
	for y := 1; y <= 9; y++ {
		s[y][7] = 0xFF6F4F37
		s[y][8] = 0xFF8F6849
	}
	s[10][7] = 0xFF352B20 // Burnt top
	s[10][8] = 0xFF453628
	// Flame head
	s[11][7] = 0xFFFF6600
	s[11][8] = 0xFFFF8800
	s[12][6] = 0xFFFF6600
	s[12][7] = 0xFFFFEE55 // Hot core
	s[12][8] = 0xFFFFEE55
	s[12][9] = 0xFFFF8800
	s[13][7] = 0xFFFFFFFF // White hot tip
	s[13][8] = 0xFFFFEE55
	s[14][7] = 0xFFFF8800
	return s
}

func getFoodSprite16x16(itemID int32) [16][16]uint32 {
	var s [16][16]uint32
	switch itemID {
	case inventory.ItemApple:
		// Iconic Red Apple
		for y := 3; y <= 11; y++ {
			for x := 4; x <= 12; x++ {
				s[y][x] = 0xFFD82020
			}
		}
		s[3][4] = 0
		s[3][12] = 0
		s[11][4] = 0
		s[11][12] = 0
		s[11][8] = 0
		s[3][8] = 0
		// Glint
		s[9][5] = 0xFFFFA090
		s[8][5] = 0xFFFFA090
		s[9][6] = 0xFFFFA090
		// Stem & Leaf
		s[12][8] = 0xFF5C3A21
		s[13][8] = 0xFF5C3A21
		s[13][9] = 0xFF4E9333
		s[14][10] = 0xFF69BD46
	case inventory.ItemBread:
		// Golden Bread Loaf
		for y := 4; y <= 11; y++ {
			for x := 2; x <= 13; x++ {
				s[y][x] = 0xFFC99742
			}
		}
		s[4][2] = 0
		s[4][13] = 0
		s[11][2] = 0
		s[11][13] = 0
		for x := 4; x <= 11; x++ {
			s[11][x] = 0xFFE0B058
		}
		// Crust score lines
		s[7][5] = 0xFF8A5D20
		s[8][5] = 0xFF8A5D20
		s[7][8] = 0xFF8A5D20
		s[8][8] = 0xFF8A5D20
		s[7][11] = 0xFF8A5D20
		s[8][11] = 0xFF8A5D20
	default: // Cooked Porkchop / Steak
		for y := 4; y <= 12; y++ {
			for x := 3; x <= 12; x++ {
				s[y][x] = 0xFF9E432C
			}
		}
		s[4][3] = 0
		s[4][12] = 0
		s[12][3] = 0
		s[12][12] = 0
		for y := 6; y <= 10; y++ {
			for x := 5; x <= 10; x++ {
				s[y][x] = 0xFFB35238
			}
		}
		// Bone
		s[11][12] = 0xFFECE7E2
		s[12][13] = 0xFFECE7E2
		s[10][13] = 0xFFECE7E2
	}
	return s
}

func (c *Client) drawHeldItem() {
	if c.player == nil || c.player.Inventory == nil {
		return
	}
	selected := c.player.Inventory.GetSelectedItem()

	// Walking sway calculation
	swayX := int(math.Cos(c.walkCycle*0.5) * 8.0)
	swayY := int(math.Abs(math.Sin(c.walkCycle)) * 6.0)

	// Arm / Tool swing animation trajectory using authentic Minecraft Java curve
	swingOffset := 0
	swingDrop := 0
	if c.isSwinging {
		f1 := math.Sin(math.Sqrt(c.swingProgress) * math.Pi)
		f2 := math.Sin(c.swingProgress * math.Pi)
		swingOffset = int(f1 * 75.0)
		swingDrop = int(f2 * 45.0)
	}

	// Hotbar equip raise/lower animation
	equipDrop := int((1.0 - c.equipProgress) * 60.0)

	hx := c.framebuf.Width - 150 + swayX - swingOffset
	hy := c.framebuf.Height - 165 + swayY + swingDrop + equipDrop

	skinTone := uint32(0xFFD6A07E)   // Tan skin
	skinShade := uint32(0xFFB57C5E)  // Shading
	sleeveTone := uint32(0xFF2E8690) // Classic Cyan shirt sleeve
	cuffTone := uint32(0xFF1B5960)   // Cuff border

	if selected.IsEmpty() {
		// Steve's authentic first-person player arm / fist (prominent 3D angled limb)
		c.framebuf.DrawRect(hx+20, hy+40, 75, 120, sleeveTone)
		c.framebuf.DrawRect(hx+15, hy+32, 80, 8, cuffTone)
		// Forearm & wrist
		c.framebuf.DrawRect(hx+22, hy-4, 65, 36, skinTone)
		c.framebuf.DrawRect(hx+18, hy-20, 70, 20, skinShade)
		// Knuckles & thumb details
		c.framebuf.DrawRect(hx+16, hy-20, 22, 16, skinTone)
		c.framebuf.DrawRect(hx+36, hy-26, 15, 10, skinTone)
		c.framebuf.DrawRect(hx+54, hy-26, 15, 10, skinTone)
		return
	}

	// 1. Food Eating Hand & Jitter
	if inventory.IsFood(selected.ItemID) {
		fx := hx + 5
		fy := hy - 15
		if c.isEating {
			fy -= 18 + int(math.Sin(c.eatTimer*40.0)*5.0)
			fx -= 12 + int(math.Cos(c.eatTimer*40.0)*3.0)
		}
		foodScale := 5
		sprite := render.GetItemSprite16x16(selected.ItemID)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				col := sprite[y][x]
				if (col >> 24) == 0 {
					continue
				}
				sx := fx + x*foodScale
				sy := fy + y*foodScale
				c.framebuf.DrawRect(sx+1, sy+1, foodScale, foodScale, darkenColor(col, 0.60))
				c.framebuf.DrawRect(sx, sy, foodScale, foodScale, col)
			}
		}
		// Steve's grasping hand holding bottom of food item
		c.framebuf.DrawRect(fx+16, fy+56, 36, 24, skinTone)
		c.framebuf.DrawRect(fx+24, fy+76, 44, 8, cuffTone)
		c.framebuf.DrawRect(fx+28, fy+84, 54, 75, sleeveTone)
		return
	}

	// 2. Torch (ID 50)
	if selected.ItemID == 50 {
		tx := hx + 15
		ty := hy - 35
		torchScale := 5
		sprite := render.GetItemSprite16x16(selected.ItemID)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				col := sprite[y][x]
				if (col >> 24) == 0 {
					continue
				}
				sx := tx + x*torchScale
				sy := ty + y*torchScale
				c.framebuf.DrawRect(sx+1, sy+1, torchScale, torchScale, darkenColor(col, 0.60))
				c.framebuf.DrawRect(sx, sy, torchScale, torchScale, col)
			}
		}
		// Hand grasping lower post
		c.framebuf.DrawRect(tx+20, ty+52, 32, 24, skinTone)
		c.framebuf.DrawRect(tx+28, ty+72, 42, 8, cuffTone)
		c.framebuf.DrawRect(tx+30, ty+80, 52, 75, sleeveTone)
		return
	}

	// 3. Tools (Pickaxe, Sword, Axe, Shovel, Hoe) - authentic hero scale
	props, isTool := inventory.GetToolProperties(selected.ItemID)
	if isTool {
		_ = props
		sprite := render.GetItemSprite16x16(selected.ItemID)
		toolScale := 5 // 5px per texel gives authentic ~80x80px weapon
		toolX := hx - 10
		toolY := hy - 35
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				col := sprite[y][x]
				if (col >> 24) == 0 {
					continue
				}
				sx := toolX + x*toolScale
				sy := toolY + y*toolScale
				c.framebuf.DrawRect(sx+1, sy+1, toolScale, toolScale, darkenColor(col, 0.60))
				c.framebuf.DrawRect(sx, sy, toolScale, toolScale, col)
			}
		}
		// Steve's fist and forearm grasping the lower handle
		handBaseX := toolX + 2*toolScale
		handBaseY := toolY + 10*toolScale
		c.framebuf.DrawRect(handBaseX-6, handBaseY-4, 32, 28, skinTone)
		c.framebuf.DrawRect(handBaseX-4, handBaseY-10, 16, 10, skinShade)
		c.framebuf.DrawRect(handBaseX+12, handBaseY-8, 12, 8, skinTone)
		c.framebuf.DrawRect(handBaseX+10, handBaseY+20, 48, 32, skinTone)
		c.framebuf.DrawRect(handBaseX+22, handBaseY+46, 46, 8, cuffTone)
		c.framebuf.DrawRect(handBaseX+26, handBaseY+54, 56, 80, sleeveTone)
		return
	}

	// 4. Non-block item materials (Diamond, Ingot, Coal, Redstone, Stick, Bone, etc.)
	if !inventory.IsPlaceableBlock(selected.ItemID) {
		ix := hx + 5
		iy := hy - 25
		itemScale := 5
		sprite := render.GetItemSprite16x16(selected.ItemID)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				col := sprite[y][x]
				if (col >> 24) == 0 {
					continue
				}
				sx := ix + x*itemScale
				sy := iy + y*itemScale
				c.framebuf.DrawRect(sx+1, sy+1, itemScale, itemScale, darkenColor(col, 0.60))
				c.framebuf.DrawRect(sx, sy, itemScale, itemScale, col)
			}
		}
		c.framebuf.DrawRect(ix+16, iy+54, 34, 24, skinTone)
		c.framebuf.DrawRect(ix+24, iy+74, 44, 8, cuffTone)
		c.framebuf.DrawRect(ix+28, iy+82, 54, 75, sleeveTone)
		return
	}

	// 5. Authentic 3D Isometric Textured Voxel Block (Held in palm)
	cx := hx + 10
	cy := hy - 5
	blockScale := 2 // 2px per texel gives authentic ~64px isometric block

	// Top Face (Diamond quad, Face 2, 1.0 full light)
	for ty := 0; ty < 16; ty++ {
		for tx := 0; tx < 16; tx++ {
			u := float32(tx) / 16.0
			v := float32(ty) / 16.0
			col := render.SampleBlockColor(selected.ItemID, 2, u, v, 1.0)
			sx := cx + (tx-ty)*blockScale
			sy := cy - 12 + (tx+ty)*blockScale/2
			c.framebuf.DrawRect(sx, sy, blockScale, (blockScale+1)/2, col)
		}
	}

	// Left Face (Parallelogram, Face 1, 0.8 directional shade)
	for ty := 0; ty < 16; ty++ {
		for tx := 0; tx < 16; tx++ {
			u := float32(tx) / 16.0
			v := float32(ty) / 16.0
			col := render.SampleBlockColor(selected.ItemID, 1, u, v, 0.80)
			sx := cx - 16*blockScale + tx*blockScale
			sy := cy + (tx+ty*2)*blockScale/2
			c.framebuf.DrawRect(sx, sy, blockScale, blockScale, col)
		}
	}

	// Right Face (Parallelogram, Face 0, 0.6 directional shade)
	for ty := 0; ty < 16; ty++ {
		for tx := 0; tx < 16; tx++ {
			u := float32(tx) / 16.0
			v := float32(ty) / 16.0
			col := render.SampleBlockColor(selected.ItemID, 0, u, v, 0.60)
			sx := cx + tx*blockScale
			sy := cy + 16 - (tx-ty*2)*blockScale/2
			c.framebuf.DrawRect(sx, sy, blockScale, blockScale, col)
		}
	}

	// Steve's hand supporting and grasping the bottom of the block
	c.framebuf.DrawRect(cx-8, cy+20, 32, 20, skinTone)
	c.framebuf.DrawRect(cx+4, cy+36, 36, 6, cuffTone)
	c.framebuf.DrawRect(cx+8, cy+42, 44, 60, sleeveTone)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			msg := fmt.Sprintf("FATAL CLIENT PANIC: %v\n\nStack:\n%s\n", r, string(stack))
			log.Println(msg)
			writeCrashLog(msg)
		}
	}()

	singleplayer := flag.Bool("singleplayer", true, "Run singleplayer embedded server with procedural world")
	serverAddr := flag.String("server", "", "Multiplayer server address (e.g. 127.0.0.1:25565)")
	username := flag.String("username", "Steve", "Player username")
	benchmark := flag.Bool("benchmark", false, "Run client rendering benchmark")
	cliMode := flag.Bool("cli", false, "Run in terminal CLI mode instead of GUI window")
	width := flag.Int("width", 800, "Window width")
	height := flag.Int("height", 600, "Window height")
	flag.Parse()

	isSP := *singleplayer
	if *serverAddr != "" {
		isSP = false
	}

	client := NewClient(*serverAddr, *username, isSP, *width, *height)
	if configDir, err := os.UserConfigDir(); err == nil {
		log.Printf("[Client] Crash log path: %s", filepath.Join(configDir, "minecraft-go", "crash.log"))
	}

	if *benchmark {
		log.Println("[Client] Running rendering benchmark (1000 frames)...")
		client.StartSingleplayerWorld("benchmark_world", true, 12345, "Benchmark", 1)
		start := time.Now()
		for i := 0; i < 1000; i++ {
			client.RenderFrame()
		}
		elapsed := time.Since(start)
		fps := 1000.0 / elapsed.Seconds()
		mspt := elapsed.Seconds() * 1000.0 / 1000.0
		log.Printf("[Benchmark] Completed: %.2f FPS, %.3f ms/frame", fps, mspt)
		return
	}

	if !isSP {
		if err := client.Connect(); err != nil {
			log.Fatalf("[Client] Connection error: %v", err)
		}
	}

	log.Printf("==================================================")
	log.Printf("   Minecraft-Go (experimental voxel sandbox)     ")
	log.Printf("   Player: %s | Mode: %s                        ", *username, map[bool]string{true: "Singleplayer (Embedded)", false: "Multiplayer"}[isSP])
	log.Printf("==================================================")

	if *cliMode {
		if isSP && client.engine == nil {
			client.StartSingleplayerWorld("cli_world", true, 12345, "CLI World", 0)
		}
		runInteractiveLoop(client)
		return
	}

	runGraphicalWindow(client, *width, *height)
}

func writeCrashLog(contents string) {
	crashLogMu.Lock()
	defer crashLogMu.Unlock()
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("[Client] Could not locate user config directory for crash log: %v", err)
		return
	}
	logDir := filepath.Join(configDir, "minecraft-go")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("[Client] Could not create crash log directory %q: %v", logDir, err)
		return
	}
	logPath := filepath.Join(logDir, "crash.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[Client] Could not write crash log %q: %v", logPath, err)
		return
	}
	_, writeErr := fmt.Fprintf(file, "\n===== %s =====\n%s\n", time.Now().Format(time.RFC3339Nano), contents)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		if writeErr != nil {
			log.Printf("[Client] Could not append crash details to %q: %v", logPath, writeErr)
		}
		if closeErr != nil {
			log.Printf("[Client] Could not close crash log %q: %v", logPath, closeErr)
		}
		return
	}
	log.Printf("[Client] Crash details saved to %s", logPath)
}

func runGraphicalWindow(c *Client, w, h int) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			msg := fmt.Sprintf("FATAL GRAPHICAL WINDOW PANIC: %v\n\nStack:\n%s\n", r, string(stack))
			log.Println(msg)
			writeCrashLog(msg)
		}
	}()

	title := fmt.Sprintf("Minecraft-Go - %s", c.username)
	wnd, err := render.NewNativeWindow(title, w, h)
	if err != nil {
		log.Printf("[Client] Failed to open native graphical window: %v. Falling back to CLI.", err)
		if c.isLocal && c.engine == nil {
			c.StartSingleplayerWorld("cli_world", true, 12345, "CLI World", 0)
		}
		runInteractiveLoop(c)
		return
	}
	c.wndRef = wnd
	wnd.SetPanicHandler(func(message string, recovered any, stack []byte) {
		msg := fmt.Sprintf("CLIENT WINDOW CALLBACK PANIC (%s): %v\n\nStack:\n%s", message, recovered, stack)
		log.Print(msg)
		writeCrashLog(msg)
	})
	watchdogDone := make(chan struct{})
	defer close(watchdogDone)
	defer func() {
		if c.worldStartDone != nil {
			<-c.worldStartDone
		}
		if c.quitSaveDone != nil {
			<-c.quitSaveDone
		}
		c.wndRef = nil
		c.SaveAndQuitToTitle()
	}()
	// Destroy the native window before the potentially slow final disk flush.
	// This keeps Windows from treating an intentionally saving process as a
	// window that stopped responding.
	defer wnd.Close()
	var loopHeartbeat atomic.Int64
	loopHeartbeat.Store(time.Now().UnixNano())
	var hangReported atomic.Bool
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-watchdogDone:
				return
			case <-ticker.C:
				lastProgress := time.Unix(0, loopHeartbeat.Load())
				stalledFor := time.Since(lastProgress)
				// Capture a stack before Windows' longer AppHang detection window
				// can terminate the process without leaving useful diagnostics.
				if stalledFor < 3*time.Second {
					hangReported.Store(false)
					continue
				}
				if !hangReported.CompareAndSwap(false, true) {
					continue
				}
				stacks := make([]byte, 1<<20)
				n := runtime.Stack(stacks, true)
				msg := fmt.Sprintf("CLIENT UI STALL: no main-loop progress for %s\n\nGoroutine stacks:\n%s", stalledFor.Round(time.Second), stacks[:n])
				log.Print(msg)
				writeCrashLog(msg)
			}
		}
	}()

	if c.screenState == ScreenTitle {
		wnd.SetMouseCaptured(false)
	}

	log.Printf("[Client] Native 3D game window opened (%dx%d)! Controls: WASD to move, Space to jump, Mouse rotates camera directly, Left-click to mine, Right-click to place, 1-9 to select hotbar.", w, h)

	// Single key-press actions
	wnd.OnKeyClick = func(vkCode int) {
		if c.quitSaving.Load() || c.worldStarting.Load() {
			return
		}
		// Global hotkeys (Fullscreen F11 and Alt+Enter work on any screen)
		if vkCode == 0x7A || (vkCode == 0x0D && wnd.IsKeyDown(0x12)) {
			wnd.ToggleFullscreen()
			return
		}

		if c.screenState == ScreenTitle {
			if vkCode == 0x1B || vkCode == 0x51 { // Esc or Q: quit
				wnd.Close()
				return
			}
			return
		}
		if c.screenState == ScreenWorldSelect {
			if vkCode == 0x1B { // Esc: return to title
				c.screenState = ScreenTitle
			}
			return
		}
		if c.screenState == ScreenNewWorld {
			if vkCode == 0x1B { // Esc: return to world select
				c.screenState = ScreenWorldSelect
			}
			return
		}

		if c.player != nil && c.player.Health <= 0 {
			return
		}

		if c.isChatOpen {
			if vkCode == 0x1B { // Escape: cancel chat
				c.isChatOpen = false
				c.chatInput = ""
				wnd.SetMouseCaptured(true)
			}
			return
		}
		if c.showAdvancementsGUI && (vkCode == 0x1B || vkCode == 0x4C) {
			c.showAdvancementsGUI = false
			wnd.SetMouseCaptured(true)
			return
		}
		switch vkCode {
		case 0x46: // F: Mine
			c.MineTargetedBlock()
		case 0x52: // R: Place
			c.PlaceBlockInFront()
		case 0x45: // E: Toggle Inventory & Crafting GUI
			if !c.showPauseMenu {
				if c.showInventoryGUI || c.isFurnaceOpen {
					c.CloseCurrentGUI()
				} else {
					c.showInventoryGUI = true
					c.isCraftingTableOpen = false
					c.isFurnaceOpen = false
					wnd.SetMouseCaptured(false)
				}
			}
		case 0x4C: // L: Advancements
			if !c.showPauseMenu && !c.showInventoryGUI && !c.isFurnaceOpen && !c.isChatOpen {
				c.showAdvancementsGUI = !c.showAdvancementsGUI
				wnd.SetMouseCaptured(!c.showAdvancementsGUI)
			}
		case 0x54: // T: Open Chat
			if !c.showPauseMenu && !c.showInventoryGUI && !c.isFurnaceOpen {
				c.isChatOpen = true
				c.chatInput = ""
				wnd.SetMouseCaptured(false)
			}
		case 0xBF: // / (Slash): Open Command Line
			if !c.showPauseMenu && !c.showInventoryGUI && !c.isFurnaceOpen {
				c.isChatOpen = true
				c.chatInput = "/"
				wnd.SetMouseCaptured(false)
			}
		case 0x1B: // Escape: Close GUI or toggle Pause Menu
			if c.showInventoryGUI || c.isFurnaceOpen {
				c.CloseCurrentGUI()
			} else {
				c.showPauseMenu = !c.showPauseMenu
				if c.engine != nil {
					c.engine.SetPaused(c.showPauseMenu)
				}
				wnd.SetMouseCaptured(!c.showPauseMenu)
			}
		case 0x72: // F3: Toggle Debug Screen
			c.showDebugF3 = !c.showDebugF3
		case 0x74: // F5: Toggle Perspective (First-Person, Third-Person Back, Third-Person Front)
			c.perspectiveMode = (c.perspectiveMode + 1) % 3
		case 0x7A: // F11: Toggle Fullscreen
			wnd.ToggleFullscreen()
		case 0x0D: // Enter: Alt+Enter toggles fullscreen
			if wnd.IsKeyDown(0x12) { // VK_MENU (Alt)
				wnd.ToggleFullscreen()
			}
		case 0x51: // Q: Quit immediately if in pause menu, or throw/drop held item in game
			if c.showPauseMenu {
				c.beginSaveAndQuitToTitle(wnd)
			} else if !c.showInventoryGUI && !c.isFurnaceOpen && !c.isChatOpen {
				c.DropHeldItem(wnd.IsKeyDown(0x11)) // Ctrl+Q drops full stack
			}
		case 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39: // 1-9
			idx := vkCode - 0x31 + 1
			if c.player != nil && c.player.Inventory != nil && !c.showPauseMenu && !c.showInventoryGUI && !c.isFurnaceOpen {
				c.player.Inventory.SelectedHotbar = idx - 1
			}
		}
	}

	// Mouse wheel hotbar switching
	wnd.OnMouseWheel = func(delta int) {
		if c.quitSaving.Load() || c.worldStarting.Load() {
			return
		}
		if c.screenState != ScreenPlaying || c.showPauseMenu || c.showInventoryGUI || c.showAdvancementsGUI || c.isFurnaceOpen || c.isChatOpen || c.player == nil || c.player.Inventory == nil {
			return
		}
		if delta > 0 {
			c.player.Inventory.SelectedHotbar = (c.player.Inventory.SelectedHotbar + 8) % 9
		} else if delta < 0 {
			c.player.Inventory.SelectedHotbar = (c.player.Inventory.SelectedHotbar + 1) % 9
		}
	}

	// Character input
	wnd.OnCharInput = func(ch rune) {
		if c.quitSaving.Load() || c.worldStarting.Load() {
			return
		}
		if c.screenState == ScreenNewWorld {
			if ch == '\r' || ch == '\n' { // Enter: create world
				dirName := fmt.Sprintf("world_%d", time.Now().Unix())
				var seed int64
				if c.newWorldSeedText != "" {
					if sVal, err := strconv.ParseInt(c.newWorldSeedText, 10, 64); err == nil {
						seed = sVal
					}
				}
				c.beginSingleplayerWorldStart(dirName, true, seed, c.newWorldName, c.newWorldMode)
				return
			}
			if ch == 27 { // Escape: cancel
				c.screenState = ScreenWorldSelect
				return
			}
			if ch == '\t' { // Tab: toggle between name and seed input
				c.newWorldActiveBox = 1 - c.newWorldActiveBox
				return
			}
			if ch == '\b' { // Backspace
				if c.newWorldActiveBox == 0 {
					if len(c.newWorldName) > 0 {
						r := []rune(c.newWorldName)
						c.newWorldName = string(r[:len(r)-1])
					}
				} else {
					if len(c.newWorldSeedText) > 0 {
						r := []rune(c.newWorldSeedText)
						c.newWorldSeedText = string(r[:len(r)-1])
					}
				}
				return
			}
			if ch >= 32 {
				if c.newWorldActiveBox == 0 {
					c.newWorldName += string(ch)
				} else {
					c.newWorldSeedText += string(ch)
				}
			}
			return
		}

		if !c.isChatOpen {
			return
		}
		if ch == '\r' || ch == '\n' { // Enter: execute
			cmdToRun := c.chatInput
			c.chatInput = ""
			c.isChatOpen = false
			wnd.SetMouseCaptured(true)
			c.ExecuteCommand(cmdToRun)
			return
		}
		if ch == '\b' { // Backspace
			if len(c.chatInput) > 0 {
				runes := []rune(c.chatInput)
				c.chatInput = string(runes[:len(runes)-1])
			}
			return
		}
		if ch == 27 { // Escape: cancel
			c.isChatOpen = false
			c.chatInput = ""
			wnd.SetMouseCaptured(true)
			return
		}
		if ch >= 32 { // Printable characters
			c.chatInput += string(ch)
		}
	}

	// Mouse button clicks
	wnd.OnMouseClick = func(btn int) {
		if c.quitSaving.Load() || c.worldStarting.Load() {
			return
		}
		mx, my := wnd.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)

		if c.screenState == ScreenTitle {
			if btn == 1 {
				// Singleplayer button
				if mx >= c.titleSpBtn[0] && mx <= c.titleSpBtn[0]+c.titleSpBtn[2] && my >= c.titleSpBtn[1] && my <= c.titleSpBtn[1]+c.titleSpBtn[3] {
					c.scanSavedWorlds()
					c.screenState = ScreenWorldSelect
					return
				}
				// Multiplayer button
				if mx >= c.titleMpBtn[0] && mx <= c.titleMpBtn[0]+c.titleMpBtn[2] && my >= c.titleMpBtn[1] && my <= c.titleMpBtn[1]+c.titleMpBtn[3] {
					c.serverAddr = "127.0.0.1:25565"
					if err := c.Connect(); err != nil {
						log.Printf("[Client] Failed to connect: %v", err)
					} else {
						c.screenState = ScreenPlaying
						wnd.SetMouseCaptured(true)
					}
					return
				}
				// Quit button
				if mx >= c.titleQuitBtn[0] && mx <= c.titleQuitBtn[0]+c.titleQuitBtn[2] && my >= c.titleQuitBtn[1] && my <= c.titleQuitBtn[1]+c.titleQuitBtn[3] {
					wnd.Close()
					return
				}
			}
			return
		}

		if c.screenState == ScreenWorldSelect {
			if btn == 1 {
				for i, r := range c.wsRects {
					if mx >= r[0] && mx <= r[0]+r[2] && my >= r[1] && my <= r[1]+r[3] {
						c.selectedWorldIdx = i
						return
					}
				}
				if mx >= c.wsPlayBtn[0] && mx <= c.wsPlayBtn[0]+c.wsPlayBtn[2] && my >= c.wsPlayBtn[1] && my <= c.wsPlayBtn[1]+c.wsPlayBtn[3] {
					if c.selectedWorldIdx >= 0 && c.selectedWorldIdx < len(c.availableWorlds) {
						target := c.availableWorlds[c.selectedWorldIdx]
						c.beginSingleplayerWorldStart(target.DirName, false, 0, target.LevelName, target.GameMode)
					}
					return
				}
				if mx >= c.wsNewBtn[0] && mx <= c.wsNewBtn[0]+c.wsNewBtn[2] && my >= c.wsNewBtn[1] && my <= c.wsNewBtn[1]+c.wsNewBtn[3] {
					c.newWorldName = "新世界"
					c.newWorldMode = 0
					c.newWorldSeedText = ""
					c.newWorldActiveBox = 0
					c.screenState = ScreenNewWorld
					return
				}
				if mx >= c.wsBackBtn[0] && mx <= c.wsBackBtn[0]+c.wsBackBtn[2] && my >= c.wsBackBtn[1] && my <= c.wsBackBtn[1]+c.wsBackBtn[3] {
					c.screenState = ScreenTitle
					return
				}
			}
			return
		}

		if c.screenState == ScreenNewWorld {
			if btn == 1 {
				if mx >= c.nwNameBox[0] && mx <= c.nwNameBox[0]+c.nwNameBox[2] && my >= c.nwNameBox[1] && my <= c.nwNameBox[1]+c.nwNameBox[3] {
					c.newWorldActiveBox = 0
					return
				}
				if mx >= c.nwSeedBox[0] && mx <= c.nwSeedBox[0]+c.nwSeedBox[2] && my >= c.nwSeedBox[1] && my <= c.nwSeedBox[1]+c.nwSeedBox[3] {
					c.newWorldActiveBox = 1
					return
				}
				if mx >= c.nwModeBtn[0] && mx <= c.nwModeBtn[0]+c.nwModeBtn[2] && my >= c.nwModeBtn[1] && my <= c.nwModeBtn[1]+c.nwModeBtn[3] {
					c.newWorldMode = 1 - c.newWorldMode
					return
				}
				if mx >= c.nwCreateBtn[0] && mx <= c.nwCreateBtn[0]+c.nwCreateBtn[2] && my >= c.nwCreateBtn[1] && my <= c.nwCreateBtn[1]+c.nwCreateBtn[3] {
					dirName := fmt.Sprintf("world_%d", time.Now().Unix())
					var seed int64
					if c.newWorldSeedText != "" {
						if sVal, err := strconv.ParseInt(c.newWorldSeedText, 10, 64); err == nil {
							seed = sVal
						}
					}
					c.beginSingleplayerWorldStart(dirName, true, seed, c.newWorldName, c.newWorldMode)
					return
				}
				if mx >= c.nwCancelBtn[0] && mx <= c.nwCancelBtn[0]+c.nwCancelBtn[2] && my >= c.nwCancelBtn[1] && my <= c.nwCancelBtn[1]+c.nwCancelBtn[3] {
					c.screenState = ScreenWorldSelect
					return
				}
			}
			return
		}

		if c.isChatOpen {
			return
		}

		if c.player != nil && c.player.Health <= 0 {
			if btn == 1 {
				r := c.respawnBtnRect
				if mx >= r[0] && mx <= r[0]+r[2] && my >= r[1] && my <= r[1]+r[3] {
					c.RespawnLocalPlayer()
					wnd.SetMouseCaptured(true)
					return
				}
				d := c.deathTitleBtnRect
				if mx >= d[0] && mx <= d[0]+d[2] && my >= d[1] && my <= d[1]+d[3] {
					c.beginSaveAndQuitToTitle(wnd)
					return
				}
			}
			return
		}

		if c.showPauseMenu {
			if btn == 1 {
				b := c.backBtnRect
				if mx >= b[0] && mx <= b[0]+b[2] && my >= b[1] && my <= b[1]+b[3] {
					c.showPauseMenu = false
					if c.engine != nil {
						c.engine.SetPaused(false)
					}
					wnd.SetMouseCaptured(true)
					return
				}
				q := c.quitBtnRect
				if mx >= q[0] && mx <= q[0]+q[2] && my >= q[1] && my <= q[1]+q[3] {
					c.beginSaveAndQuitToTitle(wnd)
					return
				}
			}
			return
		}

		if c.showInventoryGUI && c.player != nil && c.player.Inventory != nil {
			hit := render.HitTestCraftingGUI(c.framebuf.Width, c.framebuf.Height, mx, my, c.isCraftingTableOpen)
			if hit.Type == render.SlotNone {
				return
			}
			if btn == 1 { // Left click
				if hit.Type == render.SlotCraftResult {
					if !c.craftResult.IsEmpty() {
						canCraft := false
						if c.cursorItem.IsEmpty() {
							canCraft = true
						} else if c.cursorItem.ItemID == c.craftResult.ItemID && c.cursorItem.Count+c.craftResult.Count <= 64 {
							canCraft = true
						}
						if canCraft {
							craftedItemID := c.craftResult.ItemID
							if c.cursorItem.IsEmpty() {
								c.cursorItem = c.craftResult
							} else {
								c.cursorItem.Count += c.craftResult.Count
							}
							if craftedItemID == inventory.ItemStonePickaxe || craftedItemID == inventory.ItemStoneAxe || craftedItemID == inventory.ItemStoneShovel || craftedItemID == inventory.ItemStoneSword {
								c.UnlockAdvancement("minecraft:story/upgrade_tools", "装配升级")
							}
							if craftedItemID == inventory.ItemIronPickaxe {
								c.UnlockAdvancement("minecraft:story/iron_tools", "不再是木镐")
							}
							sound.PlayHardwareFeedback("minecraft:item.craft")
							gridSize := 4
							if c.isCraftingTableOpen {
								gridSize = 9
							}
							for i := 0; i < gridSize; i++ {
								if c.craftGrid[i].Count > 0 {
									c.craftGrid[i].Count--
									if c.craftGrid[i].Count <= 0 {
										c.craftGrid[i] = inventory.ItemStack{}
									}
								}
							}
							c.updateCraftResult()
						}
					}
					return
				}
				if hit.Type == render.SlotCraftGrid {
					if hit.Index >= 0 && hit.Index < len(c.craftGrid) {
						handleSlotLeftClick(&c.cursorItem, &c.craftGrid[hit.Index])
						c.updateCraftResult()
					}
					return
				}
				if hit.Type == render.SlotInventory {
					if hit.Index >= 0 && hit.Index < len(c.player.Inventory.Slots) {
						if wnd.IsKeyDown(0x10) { // Shift-click quick transfer
							if hit.Index < 9 { // Hotbar -> Main
								for s := 9; s < 36 && s < len(c.player.Inventory.Slots); s++ {
									if c.player.Inventory.Slots[s].IsEmpty() {
										c.player.Inventory.Slots[s] = c.player.Inventory.Slots[hit.Index]
										c.player.Inventory.Slots[hit.Index] = inventory.ItemStack{}
										break
									}
								}
							} else { // Main -> Hotbar
								for s := 0; s < 9 && s < len(c.player.Inventory.Slots); s++ {
									if c.player.Inventory.Slots[s].IsEmpty() {
										c.player.Inventory.Slots[s] = c.player.Inventory.Slots[hit.Index]
										c.player.Inventory.Slots[hit.Index] = inventory.ItemStack{}
										break
									}
								}
							}
						} else {
							handleSlotLeftClick(&c.cursorItem, &c.player.Inventory.Slots[hit.Index])
						}
					}
					return
				}
			} else if btn == 2 { // Right click
				if hit.Type == render.SlotCraftGrid {
					if hit.Index >= 0 && hit.Index < len(c.craftGrid) {
						handleSlotRightClick(&c.cursorItem, &c.craftGrid[hit.Index])
						c.updateCraftResult()
					}
					return
				}
				if hit.Type == render.SlotInventory {
					if hit.Index >= 0 && hit.Index < len(c.player.Inventory.Slots) {
						handleSlotRightClick(&c.cursorItem, &c.player.Inventory.Slots[hit.Index])
					}
					return
				}
			}
			return
		}

		if c.isFurnaceOpen && c.engine != nil && c.player != nil && c.player.Inventory != nil {
			furnace := c.engine.GetOrCreateFurnace(c.currentFurnacePos)
			hit := render.HitTestFurnaceGUI(c.framebuf.Width, c.framebuf.Height, mx, my)
			if hit.Type == render.SlotNone {
				return
			}
			if btn == 1 { // Left click
				if hit.Type == render.SlotFurnaceOutput {
					if !furnace.OutputSlot.IsEmpty() {
						if c.cursorItem.IsEmpty() {
							c.cursorItem = furnace.OutputSlot
							furnace.OutputSlot = inventory.ItemStack{}
						} else if c.cursorItem.ItemID == furnace.OutputSlot.ItemID && c.cursorItem.Count+furnace.OutputSlot.Count <= 64 {
							c.cursorItem.Count += furnace.OutputSlot.Count
							furnace.OutputSlot = inventory.ItemStack{}
						}
					}
					return
				}
				if hit.Type == render.SlotFurnaceInput {
					handleSlotLeftClick(&c.cursorItem, &furnace.InputSlot)
					return
				}
				if hit.Type == render.SlotFurnaceFuel {
					handleSlotLeftClick(&c.cursorItem, &furnace.FuelSlot)
					return
				}
				if hit.Type == render.SlotInventory {
					if hit.Index >= 0 && hit.Index < len(c.player.Inventory.Slots) {
						handleSlotLeftClick(&c.cursorItem, &c.player.Inventory.Slots[hit.Index])
					}
					return
				}
			} else if btn == 2 { // Right click
				if hit.Type == render.SlotFurnaceInput {
					handleSlotRightClick(&c.cursorItem, &furnace.InputSlot)
					return
				}
				if hit.Type == render.SlotFurnaceFuel {
					handleSlotRightClick(&c.cursorItem, &furnace.FuelSlot)
					return
				}
				if hit.Type == render.SlotInventory {
					if hit.Index >= 0 && hit.Index < len(c.player.Inventory.Slots) {
						handleSlotRightClick(&c.cursorItem, &c.player.Inventory.Slots[hit.Index])
					}
					return
				}
			}
			return
		}

		if !wnd.MouseCaptured {
			wnd.SetMouseCaptured(true)
		}
		if btn == 1 {
			if c.AttackMob() {
				return
			}
			if !c.isSwinging {
				c.isSwinging = true
				c.swingProgress = 0.0
			}
		} else if btn == 2 {
			if c.hasTarget && c.engine != nil {
				targetID := c.engine.GetBlockIfLoaded(c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
				if targetID == 58 { // Crafting Table
					c.showInventoryGUI = true
					c.isCraftingTableOpen = true
					c.isFurnaceOpen = false
					wnd.SetMouseCaptured(false)
					return
				}
				if targetID == 61 || targetID == 62 { // Furnace
					c.showInventoryGUI = false
					c.isCraftingTableOpen = false
					c.isFurnaceOpen = true
					c.currentFurnacePos = c.targetedPos
					wnd.SetMouseCaptured(false)
					return
				}
			}
			if c.placeCooldown <= 0 {
				c.UpdateRaycastTarget()
				if c.PlaceBlockInFront() {
					c.placeCooldown = 0.25
				}
			}
		}
	}

	// High-FPS precision delta-time game loop
	mouseSensitivity := 0.15
	lastTime := time.Now()
	lastAutosave := lastTime

	for {
		loopHeartbeat.Store(time.Now().UnixNano())
		if !wnd.PollEvents() {
			break
		}
		if c.worldStarting.Load() {
			c.framebuf.Clear(0xFF171717)
			text := "正在加载世界，请稍候..."
			textWidth := render.MeasureText5x7(text, 1)
			c.framebuf.DrawText5x7((c.framebuf.Width-textWidth)/2, c.framebuf.Height/2, text, 0xFFFFFFFF, 1)
			wnd.Present(c.framebuf)
			time.Sleep(16 * time.Millisecond)
			continue
		}
		if c.quitSaving.Load() {
			c.framebuf.Clear(0xFF171717)
			text := "正在保存世界，请稍候..."
			if c.quitSaveFailed.Load() {
				text = "保存过程异常，请关闭窗口重试并检查 crash.log"
			}
			textWidth := render.MeasureText5x7(text, 1)
			c.framebuf.DrawText5x7((c.framebuf.Width-textWidth)/2, c.framebuf.Height/2, text, 0xFFFFFFFF, 1)
			wnd.Present(c.framebuf)
			time.Sleep(16 * time.Millisecond)
			continue
		}
		if c.worldStartCapturePending.Swap(false) {
			wnd.SetMouseCaptured(true)
		}
		c.pollSavedWorldScan()
		// Detect window resize (Fullscreen toggle or maximize) and clamp internal software rasterizer resolution
		// Internal CPU rasterizer target is optimized to max 960x540 (or 800x600 for 4:3), preserving exact window aspect ratio.
		// Windows GDI's StretchDIBits hardware-upscales seamlessly to full 1080p/2K/4K display with zero CPU overhead.
		if wnd.Width > 0 && wnd.Height > 0 {
			targetW := wnd.Width
			targetH := wnd.Height
			const maxRenderW = 960
			const maxRenderH = 540
			if targetW > maxRenderW || targetH > maxRenderH {
				aspect := float64(targetW) / float64(targetH)
				if aspect >= float64(maxRenderW)/float64(maxRenderH) {
					targetW = maxRenderW
					targetH = int(float64(maxRenderW) / aspect)
				} else {
					targetH = maxRenderH
					targetW = int(float64(maxRenderH) * aspect)
				}
			}
			if targetW < 400 {
				targetW = 400
			}
			if targetH < 300 {
				targetH = 300
			}
			c.framebuf.Resize(targetW, targetH)
		}

		now := time.Now()
		dt := now.Sub(lastTime).Seconds()
		if dt <= 0 {
			dt = 0.001
		}
		if dt > 0.05 {
			dt = 0.05
		}
		lastTime = now
		c.AdvanceSimulation(dt)
		if !c.quitSaving.Load() && !c.worldStarting.Load() && c.isLocal && c.screenState == ScreenPlaying && c.engine != nil && c.saveMgr != nil &&
			c.player != nil && c.player.Health > 0 && now.Sub(lastAutosave) >= time.Minute &&
			c.autosaveRunning.CompareAndSwap(false, true) {
			lastAutosave = now
			engine, storage, level := c.engine, c.saveMgr, c.snapshotLevelData()
			if level != nil {
				c.autosaveWG.Add(1)
				go func() {
					defer c.autosaveWG.Done()
					defer c.autosaveRunning.Store(false)
					defer func() {
						if recovered := recover(); recovered != nil {
							msg := fmt.Sprintf("PERIODIC AUTOSAVE PANIC: %v\n\nStack:\n%s", recovered, debug.Stack())
							log.Print(msg)
							writeCrashLog(msg)
						}
					}()
					if err := engine.SaveAll(); err != nil {
						log.Printf("[Client] Periodic chunk autosave failed: %v", err)
					}
					if err := storage.SaveLevel(level); err != nil {
						log.Printf("[Client] Periodic level autosave failed: %v", err)
					}
				}()
			} else {
				c.autosaveRunning.Store(false)
			}
		}

		// Update item equip animation progress
		if c.equipProgress < 1.0 {
			c.equipProgress += dt * 5.0
			if c.equipProgress > 1.0 {
				c.equipProgress = 1.0
			}
		}

		if c.screenState == ScreenTitle {
			time.Sleep(16 * time.Millisecond)
			mx, my := wnd.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
			c.titleSpBtn, c.titleMpBtn, c.titleQuitBtn = c.framebuf.DrawTitleScreen(mx, my)
			if c.quitSaveFailed.Load() || c.worldStartFailed.Load() || c.networkDisconnected.Load() {
				msg := "操作失败，请查看 crash.log"
				if c.networkDisconnected.Load() {
					msg = "服务器连接已断开"
				} else if c.quitSaveFailed.Load() {
					msg = "存档保存失败，详情见 crash.log"
				}
				textWidth := render.MeasureText5x7(msg, 1)
				c.framebuf.DrawText5x7((c.framebuf.Width-textWidth)/2, c.framebuf.Height-60, msg, 0xFFFF7777, 1)
			}
			wnd.Present(c.framebuf)
			continue
		}

		if c.screenState == ScreenWorldSelect {
			time.Sleep(16 * time.Millisecond)
			mx, my := wnd.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
			c.wsPlayBtn, c.wsNewBtn, c.wsBackBtn, c.wsRects = c.framebuf.DrawWorldSelectScreenWithLoading(c.availableWorlds, c.selectedWorldIdx, c.worldScanPending, mx, my)
			if c.worldStartFailed.Load() {
				msg := "世界加载失败，请检查日志"
				textWidth := render.MeasureText5x7(msg, 1)
				c.framebuf.DrawText5x7((c.framebuf.Width-textWidth)/2, c.framebuf.Height-60, msg, 0xFFFF7777, 1)
			}
			wnd.Present(c.framebuf)
			continue
		}

		if c.screenState == ScreenNewWorld {
			time.Sleep(16 * time.Millisecond)
			mx, my := wnd.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
			c.nwCreateBtn, c.nwCancelBtn, c.nwModeBtn, c.nwNameBox, c.nwSeedBox = c.framebuf.DrawNewWorldScreen(c.newWorldName, c.newWorldMode, c.newWorldSeedText, c.newWorldActiveBox, mx, my)
			if c.worldStartFailed.Load() {
				msg := "世界创建失败，请检查日志"
				textWidth := render.MeasureText5x7(msg, 1)
				c.framebuf.DrawText5x7((c.framebuf.Width-textWidth)/2, c.framebuf.Height-18, msg, 0xFFFF7777, 1)
			}
			wnd.Present(c.framebuf)
			continue
		}

		// If Player is Dead, release mouse capture, show Death Screen, and halt world controls
		if c.player != nil && c.player.Health <= 0 {
			wnd.SetMouseCaptured(false)
			time.Sleep(16 * time.Millisecond)
			if !c.renderFrameSafely() {
				if c.renderPanicCount >= 3 {
					break
				}
				continue
			}
			wnd.Present(c.framebuf)
			continue
		}

		// If Pause Menu is open, freeze world controls and display interactive Pause Menu
		if c.showPauseMenu {
			time.Sleep(16 * time.Millisecond)
			if !c.renderFrameSafely() {
				if c.renderPanicCount >= 3 {
					break
				}
				continue
			}
			mx, my := wnd.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
			c.backBtnRect, c.quitBtnRect = c.framebuf.DrawPauseMenu(mx, my)
			wnd.Present(c.framebuf)
			continue
		}

		if c.showAdvancementsGUI {
			time.Sleep(16 * time.Millisecond)
			if !c.renderFrameSafely() {
				if c.renderPanicCount >= 3 {
					break
				}
				continue
			}
			c.framebuf.DrawAdvancementScreen(c.advancementLines())
			wnd.Present(c.framebuf)
			continue
		}

		// If Inventory / Crafting Table / Furnace GUI is open
		if c.showInventoryGUI || c.isFurnaceOpen {
			time.Sleep(16 * time.Millisecond)
			if !c.renderFrameSafely() {
				if c.renderPanicCount >= 3 {
					break
				}
				continue
			}
			wnd.Present(c.framebuf)
			continue
		}

		// If Chat / Command input is open, freeze world controls and display chat overlay
		if c.isChatOpen {
			time.Sleep(16 * time.Millisecond)
			if !c.renderFrameSafely() {
				if c.renderPanicCount >= 3 {
					break
				}
				continue
			}
			wnd.Present(c.framebuf)
			continue
		}

		// 1. Spacebar Jump & Swim input (Authentic Minecraft: continuous hold for water swimming, flight ascent & bunny hop)
		spaceDown := wnd.IsKeyDown(0x20)
		c.wasSpaceDown = spaceDown

		// 2. Sprint & Sneak modifiers
		c.isSneaking = wnd.IsKeyDown(0x10)  // Shift
		c.isSprinting = wnd.IsKeyDown(0x11) // Ctrl
		if c.player != nil && c.player.Hunger <= 6 {
			c.isSprinting = false
		}

		// 3. Movement input polling
		var fwdInput, strafeInput float64
		if wnd.IsKeyDown(0x57) { // W
			fwdInput += 1.0
		}
		if wnd.IsKeyDown(0x53) { // S
			fwdInput -= 1.0
		}
		if wnd.IsKeyDown(0x41) { // A
			strafeInput -= 1.0
		}
		if wnd.IsKeyDown(0x44) { // D
			strafeInput += 1.0
		}

		c.PhysicsStep(fwdInput, strafeInput, spaceDown, dt)
		c.publishPlayerSnapshot()

		// Synchronize crosshair raycast target with current position & orientation
		c.UpdateRaycastTarget()

		// 4. Authentic Minecraft Mining: Hold Left Mouse to mine with cracking & tool times
		isLeftDown := wnd.IsMouseDown(1)
		c.UpdateMining(isLeftDown, dt)

		// 5. Placing Blocks or Consuming Food: Hold Right Mouse Button
		isRightDown := wnd.IsMouseDown(2)
		c.UpdateRightClick(isRightDown, dt)

		// Keyboard arrow view look fallback (smooth dt scaled)
		if wnd.IsKeyDown(0x25) { // Left (turn left)
			c.RotateView(-120.0*dt, 0)
		}
		if wnd.IsKeyDown(0x27) { // Right (turn right)
			c.RotateView(120.0*dt, 0)
		}
		if wnd.IsKeyDown(0x26) { // Up (look up)
			c.RotateView(0, 120.0*dt)
		}
		if wnd.IsKeyDown(0x28) { // Down (look down)
			c.RotateView(0, -120.0*dt)
		}

		// 2. Hardware 1:1 Mouse delta polling (Horizon & Minecraft standard centering)
		// Standard FPS mouse look: +mdx turns yaw right, -mdy pitches camera up
		mdx, mdy := wnd.PollMouseDelta()
		if mdx != 0 || mdy != 0 {
			c.RotateView(mdx*mouseSensitivity, -mdy*mouseSensitivity)
		}

		// 3. Render frame and present directly to native window
		if !c.renderFrameSafely() {
			if c.renderPanicCount >= 3 {
				break
			}
			continue
		}
		wnd.Present(c.framebuf)

		// Brief yield to avoid pegging 100% CPU while delivering maximum smoothness
		time.Sleep(1 * time.Millisecond)
	}

	log.Println("[Client] Game window closed.")
}

func runInteractiveLoop(c *Client) {
	defer c.SaveAndQuitToTitle()

	fmt.Println("\nCommands:")
	fmt.Println("  w / s / a / d   : Move player")
	fmt.Println("  goto <x> <y> <z>: Teleport/move to coordinates")
	fmt.Println("  look <yaw> <pitch> : Rotate camera view")
	fmt.Println("  mine            : Mine targeted block in crosshair")
	fmt.Println("  place           : Place active hotbar block")
	fmt.Println("  block <x> <y> <z> : Query block ID at coordinates")
	fmt.Println("  place_at <x> <y> <z> <id> : Place specific block ID at coordinates")
	fmt.Println("  mine_at <x> <y> <z>       : Mine block at coordinates")
	fmt.Println("  slot <1-9>      : Switch active hotbar slot")
	fmt.Println("  inv             : View inventory")
	fmt.Println("  craft <recipe>  : Craft items")
	fmt.Println("  status          : View health, hunger, XYZ position")
	fmt.Println("  render          : Render a full 3D frame and HUD")
	fmt.Println("  quit            : Exit game")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("minecraft> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		c.AdvanceSimulation(0.05)

		parts := strings.Fields(input)
		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "w":
			if c.player != nil {
				c.MoveRelative(1.0, 0)
				fmt.Printf("Moved forward. Position: (%.2f, %.2f, %.2f)\n", c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
			} else {
				fmt.Println("Player entity not initialized.")
			}
		case "s":
			if c.player != nil {
				c.MoveRelative(-1.0, 0)
				fmt.Printf("Moved backward. Position: (%.2f, %.2f, %.2f)\n", c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
			} else {
				fmt.Println("Player entity not initialized.")
			}
		case "a":
			if c.player != nil {
				c.MoveRelative(0, -1.0)
				fmt.Printf("Strafed left. Position: (%.2f, %.2f, %.2f)\n", c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
			} else {
				fmt.Println("Player entity not initialized.")
			}
		case "d":
			if c.player != nil {
				c.MoveRelative(0, 1.0)
				fmt.Printf("Strafed right. Position: (%.2f, %.2f, %.2f)\n", c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
			} else {
				fmt.Println("Player entity not initialized.")
			}
		case "goto":
			if len(parts) >= 4 {
				gx, _ := strconv.ParseFloat(parts[1], 64)
				gy, _ := strconv.ParseFloat(parts[2], 64)
				gz, _ := strconv.ParseFloat(parts[3], 64)
				if c.player != nil {
					c.player.Position = sim.Vec3{X: gx, Y: gy, Z: gz}
					c.camera.X = gx
					c.camera.Y = gy + 1.62
					c.camera.Z = gz
				}
				if c.conn != nil {
					_ = c.sendPacket(&protocol.PlayerPositionPacket{
						X:        gx,
						Y:        gy,
						Z:        gz,
						Yaw:      float32(c.camera.Yaw),
						Pitch:    float32(c.camera.Pitch),
						OnGround: true,
					})
				}
				fmt.Printf("Moved to (%.2f, %.2f, %.2f)\n", gx, gy, gz)
			}
		case "look":
			if len(parts) >= 3 {
				yaw, _ := strconv.ParseFloat(parts[1], 64)
				pitch, _ := strconv.ParseFloat(parts[2], 64)
				c.camera.Yaw = yaw
				c.camera.Pitch = pitch
				fmt.Printf("Camera rotated: Yaw=%.1f, Pitch=%.1f\n", yaw, pitch)
			}
		case "block":
			if len(parts) >= 4 {
				bx, _ := strconv.Atoi(parts[1])
				by, _ := strconv.Atoi(parts[2])
				bz, _ := strconv.Atoi(parts[3])
				bid := int32(0)
				if c.engine != nil {
					bid = c.engine.GetBlock(int32(bx), int32(by), int32(bz))
				}
				fmt.Printf("Block at (%d, %d, %d): ID=%d\n", bx, by, bz, bid)
			}
		case "place_at":
			if len(parts) >= 5 {
				bx, _ := strconv.Atoi(parts[1])
				by, _ := strconv.Atoi(parts[2])
				bz, _ := strconv.Atoi(parts[3])
				bid, _ := strconv.Atoi(parts[4])
				applied := c.engine != nil && c.engine.SetBlockChecked(int32(bx), int32(by), int32(bz), int32(bid))
				if applied {
					c.RebuildAffectedMeshes(world.BlockPos{X: int32(bx), Y: int32(by), Z: int32(bz)})
				}
				if applied && c.conn != nil {
					_ = c.sendPacket(&protocol.BlockUpdatePacket{
						X:       int32(bx),
						Y:       int32(by),
						Z:       int32(bz),
						BlockID: int32(bid),
					})
				}
				if applied {
					fmt.Printf("Placed block ID %d at (%d, %d, %d)\n", bid, bx, by, bz)
				} else {
					fmt.Printf("Could not update block at (%d, %d, %d)\n", bx, by, bz)
				}
			}
		case "mine_at":
			if len(parts) >= 4 {
				bx, _ := strconv.Atoi(parts[1])
				by, _ := strconv.Atoi(parts[2])
				bz, _ := strconv.Atoi(parts[3])
				applied := c.engine != nil && c.engine.SetBlockChecked(int32(bx), int32(by), int32(bz), 0)
				if applied {
					c.RebuildAffectedMeshes(world.BlockPos{X: int32(bx), Y: int32(by), Z: int32(bz)})
				}
				if applied && c.conn != nil {
					_ = c.sendPacket(&protocol.BlockUpdatePacket{
						X:       int32(bx),
						Y:       int32(by),
						Z:       int32(bz),
						BlockID: 0,
					})
				}
				if applied {
					fmt.Printf("Mined block at (%d, %d, %d)\n", bx, by, bz)
				} else {
					fmt.Printf("Could not update block at (%d, %d, %d)\n", bx, by, bz)
				}
			}
		case "mine":
			drop, ok := c.MineTargetedBlock()
			if ok {
				fmt.Printf("Mined block! Gained item ID: %d in inventory.\n", drop)
			}
		case "place":
			ok := c.PlaceBlockInFront()
			if ok {
				fmt.Println("Placed block successfully!")
			}
		case "slot":
			if len(parts) >= 2 && c.player != nil && c.player.Inventory != nil {
				s, _ := strconv.Atoi(parts[1])
				if s >= 1 && s <= 9 {
					c.player.Inventory.SelectedHotbar = s - 1
					fmt.Printf("Active hotbar slot switched to: %d (%+v)\n", s, c.player.Inventory.GetSelectedItem())
				}
			}
		case "inv":
			if c.player != nil && c.player.Inventory != nil {
				fmt.Println("=== Player Hotbar ===")
				for i := 0; i < 9; i++ {
					marker := " "
					if i == c.player.Inventory.SelectedHotbar {
						marker = "*"
					}
					slot := c.player.Inventory.Slots[i]
					fmt.Printf("[%sSlot %d] ID: %d, Count: %d\n", marker, i+1, slot.ItemID, slot.Count)
				}
			}
		case "craft":
			if len(parts) >= 2 && c.player != nil && c.player.Inventory != nil {
				target := strings.ToLower(parts[1])
				switch target {
				case "planks":
					if err := c.player.Inventory.RemoveItem(0, 1); err == nil {
						c.player.Inventory.AddItem(inventory.ItemOakPlanks, 4)
						fmt.Println("Crafted 4 Oak Planks!")
					}
				case "table":
					if err := c.player.Inventory.RemoveItem(0, 4); err == nil {
						c.player.Inventory.AddItem(inventory.ItemCraftingTable, 1)
						fmt.Println("Crafted 1 Crafting Table!")
					}
				case "pickaxe":
					c.player.Inventory.AddItem(inventory.ItemWoodenPickaxe, 1)
					fmt.Println("Crafted 1 Wooden Pickaxe!")
				}
			}
		case "status":
			if c.player != nil {
				fmt.Printf("Player: %s | Health: %.1f/20 | Hunger: %d/20 | Pos: (%.2f, %.2f, %.2f)\n",
					c.player.Username, c.player.Health, c.player.Hunger, c.player.Position.X, c.player.Position.Y, c.player.Position.Z)
			} else {
				fmt.Println("Player entity not initialized.")
			}
			c.remoteMu.RLock()
			for name, rp := range c.remotePlayers {
				fmt.Printf("  Remote Player %s: Pos: (%.2f, %.2f, %.2f)\n", name, rp.Pos.X, rp.Pos.Y, rp.Pos.Z)
			}
			c.remoteMu.RUnlock()
		case "render":
			if c.player != nil {
				c.RenderFrame()
				fmt.Printf("Rendered 640x480 frame with 3D terrain and HUD. Active Slot: %d\n", c.player.Inventory.SelectedHotbar+1)
			}
		case "quit", "exit":
			fmt.Println("Goodbye!")
			return
		}
	}
}

