package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
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
	priority   int64
	engine     *sim.Engine
}

type chunkStreamJob struct {
	pos        world.ChunkPos
	generation uint64
	dimension  int
	priority   int64
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
	streamCenterChunkX       atomic.Int32
	streamCenterChunkZ       atomic.Int32
	streamRadius             atomic.Int32
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
	isMining               bool
	breakProgress          float64 // 0.0 to 1.0
	miningPos              world.BlockPos
	hasMiningBlock         bool
	swingProgress          float64 // 0.0 to 1.0
	isSwinging             bool
	lastHitSoundTime       time.Time
	placeCooldown          float64
	particles              []render.Particle
	showInventoryGUI       bool
	creativeTab            inventory.CreativeCategory
	creativeSearch         string
	creativeSearchActive   bool
	creativeScrollRow      int
	creativeScrollDrag     bool
	creativeScrollOffset   int
	creativeWheelRemainder int
	creativeDrag           creativeInventoryDrag
	creativeHotbars        inventory.CreativeHotbarPresets
	showAdvancementsGUI    bool
	showPauseMenu          bool
	backBtnRect            [4]int
	quitBtnRect            [4]int
	isEating               bool
	eatTimer               float64
	lastMunchSound         time.Time
	showDebugF3            bool
	lastStepTime           time.Time
	fpsCounter             int
	currentFPS             int
	fpsTimer               time.Time
	lastFrameDuration      float64
	isChatOpen             bool
	chatInput              string
	chatMessages           []render.ChatMessage
	advancements           map[string]bool
	isCreative             bool
	isFlying               bool
	perspectiveMode        int // 0: 1st person, 1: 3rd person back, 2: 3rd person front
	lastPlayerHealth       float32
	hurtFlashTimer         float64
	currentEyeHeight       float64
	targetFov              float64
	equipProgress          float64
	lastSelectedSlot       int
	respawnBtnRect         [4]int
	deathTitleBtnRect      [4]int
}

func buildSectionMeshSafely(engine *sim.Engine, key [3]int32) (mesh *render.Mesh) {
	defer func() {
		if recovered := recover(); recovered != nil {
			stack := debug.Stack()
			msg := fmt.Sprintf("CLIENT SECTION MESH BUILD PANIC at section %v: %v\n\nStack:\n%s", key, recovered, stack)
			log.Print(msg)
			writeCrashLog(msg)
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
		if item.Slot == world.PlayerOffhandInventorySlot {
			if target.Offhand.IsEmpty() && item.ItemID > 0 && item.Count > 0 {
				count := min(item.Count, inventory.GetMaxStackSize(item.ItemID))
				target.Offhand = inventory.ItemStack{ItemID: item.ItemID, Count: count, Damage: max(0, item.Damage)}
			}
			continue
		}
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

func restoreSavedArmor(target *inventory.PlayerInventory, saved []world.SavedItemStack) {
	if target == nil {
		return
	}
	for _, item := range saved {
		if item.Slot < 0 || item.Slot >= len(target.Armor) || item.ItemID <= 0 || item.Count <= 0 || !inventory.CanEquipArmor(item.Slot, item.ItemID) {
			continue
		}
		if target.Armor[item.Slot].IsEmpty() {
			damage := max(0, item.Damage)
			target.Armor[item.Slot] = inventory.ItemStack{ItemID: item.ItemID, Count: 1, Damage: damage}
		}
	}
}

func savedArmorStacks(inv *inventory.PlayerInventory) []world.SavedItemStack {
	if inv == nil {
		return nil
	}
	saved := make([]world.SavedItemStack, 0, len(inv.Armor))
	for slot, item := range inv.Armor {
		if !item.IsEmpty() {
			saved = append(saved, world.SavedItemStack{Slot: slot, ItemID: item.ItemID, Count: 1, Damage: item.Damage})
		}
	}
	return saved
}

func (c *Client) syncPlayerArmorProtection() {
	if c == nil || c.engine == nil || c.player == nil || c.player.Inventory == nil {
		return
	}
	c.engine.SetPlayerArmorProtection(c.player.ID, inventory.CalculateArmorProtection(c.player.Inventory.Armor))
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
			if !math.IsNaN(float64(savedPlayer.Saturation)) && !math.IsInf(float64(savedPlayer.Saturation), 0) {
				c.player.Saturation = float32(math.Max(0, math.Min(float64(c.player.Hunger), float64(savedPlayer.Saturation))))
			}
			if !math.IsNaN(float64(savedPlayer.Absorption)) && !math.IsInf(float64(savedPlayer.Absorption), 0) {
				c.player.Absorption = float32(math.Max(0, math.Min(20, float64(savedPlayer.Absorption))))
			}
			for _, effect := range savedPlayer.Effects {
				if c.player.EffectCount >= len(c.player.ActiveEffects) || effect.ID == "" || effect.DurationTicks <= 0 || effect.Amplifier < 0 {
					continue
				}
				c.player.ActiveEffects[c.player.EffectCount] = effect
				c.player.EffectCount++
			}
			if !math.IsNaN(float64(savedPlayer.Yaw)) && !math.IsInf(float64(savedPlayer.Yaw), 0) {
				c.camera.Yaw = float64(savedPlayer.Yaw)
			}
			if !math.IsNaN(float64(savedPlayer.Pitch)) && !math.IsInf(float64(savedPlayer.Pitch), 0) {
				c.camera.Pitch = math.Max(-90, math.Min(90, float64(savedPlayer.Pitch)))
			}
			restoreSavedInventory(c.player.Inventory, savedPlayer.Inventory)
			restoreSavedArmor(c.player.Inventory, savedPlayer.Armor)
			c.syncPlayerArmorProtection()
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
			savedArmor := savedArmorStacks(c.player.Inventory)
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
					Saturation:   c.player.Saturation,
					Absorption:   c.player.Absorption,
					Effects:      append([]world.PlayerStatusEffect(nil), c.player.ActiveEffects[:c.player.EffectCount]...),
					GameMode:     gm,
					Inventory:    savedInv,
					Armor:        savedArmor,
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
	inventory := make([]world.SavedItemStack, 0, len(c.player.Inventory.Slots)+1)
	for slot, stack := range c.player.Inventory.Slots {
		if stack.IsEmpty() {
			continue
		}
		inventory = append(inventory, world.SavedItemStack{Slot: slot, ItemID: stack.ItemID, Count: stack.Count, Damage: stack.Damage})
	}
	if !c.player.Inventory.Offhand.IsEmpty() {
		stack := c.player.Inventory.Offhand
		inventory = append(inventory, world.SavedItemStack{Slot: world.PlayerOffhandInventorySlot, ItemID: stack.ItemID, Count: stack.Count, Damage: stack.Damage})
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
			Saturation: c.player.Saturation, Absorption: c.player.Absorption,
			Effects:   append([]world.PlayerStatusEffect(nil), c.player.ActiveEffects[:c.player.EffectCount]...),
			Inventory: inventory, Advancements: c.advancementIDs(),
			Armor: savedArmorStacks(c.player.Inventory),
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
		creativeHotbars:   loadCreativeHotbarPresets(),
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

func creativeHotbarPresetPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "minecraft-go", "hotbar-presets.json"), nil
}

func loadCreativeHotbarPresets() inventory.CreativeHotbarPresets {
	var presets inventory.CreativeHotbarPresets
	path, err := creativeHotbarPresetPath()
	if err != nil {
		return presets
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return presets
	}
	if err := json.Unmarshal(data, &presets); err != nil {
		log.Printf("[Creative] Ignoring invalid hotbar presets at %q: %v", path, err)
		return inventory.CreativeHotbarPresets{}
	}
	return presets
}

func saveCreativeHotbarPresets(presets inventory.CreativeHotbarPresets) error {
	path, err := creativeHotbarPresetPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(presets, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "hotbar-presets-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err == nil {
		return nil
	}
	// Windows does not replace an existing destination with Rename.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmpName, path)
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
				if batch[i].priority != batch[j].priority {
					return batch[i].priority < batch[j].priority
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
	if job.engine == nil {
		return
	}
	if job.unload {
		if job.engine.GetDimension() == job.dimension && c.chunkWithinCurrentRadius(job.pos, 2) {
			return
		}
		if err := job.engine.UnloadChunkInDimension(job.dimension, job.pos); err != nil {
			log.Printf("[Client] Failed to save chunk %v while unloading; retaining it in memory: %v", job.pos, err)
			return
		}
		c.meshMu.Lock()
		if c.meshGeneration == job.generation && job.engine.GetDimension() == job.dimension {
			delete(c.loadedChunks, job.pos)
			c.invalidateChunkMeshNeighborhoodLocked(job.pos)
		}
		c.meshMu.Unlock()
		c.pruneMeshRevisions()
		return
	}
	if !c.meshGenerationIs(job.generation) || !c.chunkWithinCurrentRadius(job.pos, 1) {
		return
	}

	chunk, loadErr := job.engine.GetChunkInDimensionWithError(job.dimension, job.pos)
	if loadErr != nil {
		c.suppressChunkStreamRetry(job)
		log.Printf("[Client] Refusing to regenerate unreadable chunk %v in dimension %d: %v", job.pos, job.dimension, loadErr)
		return
	}
	if !c.meshGenerationIs(job.generation) || !c.chunkWithinCurrentRadius(job.pos, 1) {
		if job.engine.GetDimension() != job.dimension || !c.chunkWithinCurrentRadius(job.pos, 1) {
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
		if !c.meshGenerationIs(job.generation) || !c.chunkWithinCurrentRadius(job.pos, 1) {
			return
		}
		job.engine.LoadGeneratedChunkInDimension(job.dimension, job.pos, chunk)
	}
	if !c.meshGenerationIs(job.generation) || !c.chunkWithinCurrentRadius(job.pos, 1) {
		if err := job.engine.UnloadChunkInDimension(job.dimension, job.pos); err != nil {
			log.Printf("[Client] Failed to save stale chunk %v in dimension %d: %v", job.pos, job.dimension, err)
		}
		return
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
		c.invalidateChunkMeshNeighborhoodLocked(job.pos)
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

func (c *Client) chunkWithinCurrentRadius(pos world.ChunkPos, extra int32) bool {
	radius := c.streamRadius.Load() + extra
	if radius < 0 {
		return false
	}
	dx := int64(pos.X) - int64(c.streamCenterChunkX.Load())
	dz := int64(pos.Z) - int64(c.streamCenterChunkZ.Load())
	if dx < 0 {
		dx = -dx
	}
	if dz < 0 {
		dz = -dz
	}
	return dx <= int64(radius) && dz <= int64(radius)
}

func (c *Client) meshJobWithinCurrentRadius(key [3]int32) bool {
	return c.chunkWithinCurrentRadius(world.ChunkPos{X: key[0], Z: key[2]}, 0)
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
	c.streamCenterChunkX.Store(pcx)
	c.streamCenterChunkZ.Store(pcz)
	c.streamRadius.Store(max(int32(0), radius))
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
	for _, cp := range nearbyChunkPositions(pcx, pcz, radius+1) {
		c.meshMu.Lock()
		alreadyLoaded := c.loadedChunks[cp]
		c.meshMu.Unlock()
		if alreadyLoaded || c.chunkStreamPending(cp, dimension, generation) {
			continue
		}
		dx, dz := int64(cp.X-pcx), int64(cp.Z-pcz)
		streamJobs = append(streamJobs, chunkStreamJob{pos: cp, generation: generation, dimension: dimension, priority: dx*dx + dz*dz, engine: engine, generator: generator})
		queuedCount++
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

	for _, cp := range nearbyChunkPositions(pcx, pcz, radius) {
		dx, dz := int64(cp.X-pcx), int64(cp.Z-pcz)
		horizontalPriority := (dx*dx + dz*dz) * int64(world.NumSections+1)
		for verticalPriority, sy := range orderedSY {
			key := [3]int32{cp.X, sy, cp.Z}
			if c.meshCache[key] != nil {
				continue
			}
			job := sectionMeshJob{
				key:        key,
				generation: c.meshGeneration,
				revision:   c.meshRevisions[key],
				priority:   horizontalPriority + int64(verticalPriority),
				engine:     engine,
			}
			if !c.meshJobPending(key, job.generation, job.revision) {
				unmeshed = append(unmeshed, job)
				if len(unmeshed) == cap(unmeshed) {
					goto meshesSelected
				}
			}
		}
	}

meshesSelected:
	c.meshMu.RUnlock()

	if len(unmeshed) == 0 {
		return
	}

	c.enqueueMeshJobs(unmeshed)
}

// nearbyChunkPositions returns a deterministic nearest-first square around the
// player so nearby terrain is available before distant chunks in the same batch.
func nearbyChunkPositions(centerX, centerZ, radius int32) []world.ChunkPos {
	if radius < 0 {
		return nil
	}
	positions := make([]world.ChunkPos, 0, (2*int(radius)+1)*(2*int(radius)+1))
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			positions = append(positions, world.ChunkPos{X: centerX + dx, Z: centerZ + dz})
		}
	}
	sort.Slice(positions, func(i, j int) bool {
		dxi, dzi := int64(positions[i].X-centerX), int64(positions[i].Z-centerZ)
		dxj, dzj := int64(positions[j].X-centerX), int64(positions[j].Z-centerZ)
		di, dj := dxi*dxi+dzi*dzi, dxj*dxj+dzj*dzj
		if di != dj {
			return di < dj
		}
		if positions[i].X != positions[j].X {
			return positions[i].X < positions[j].X
		}
		return positions[i].Z < positions[j].Z
	})
	return positions
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
				if batch[i].priority != batch[j].priority {
					return batch[i].priority < batch[j].priority
				}
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
	if !current || job.engine == nil || !c.meshJobWithinCurrentRadius(job.key) {
		return
	}
	mesh := buildSectionMeshSafely(job.engine, job.key)
	if !c.meshJobWithinCurrentRadius(job.key) {
		return
	}
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

// invalidateChunkMeshNeighborhoodLocked invalidates the loaded chunk column and
// all eight horizontal neighbors. Ambient occlusion samples diagonal blocks,
// so a cardinal-only invalidation leaves stale corner shading at chunk seams.
// The caller must hold meshMu.
func (c *Client) invalidateChunkMeshNeighborhoodLocked(pos world.ChunkPos) {
	if c.meshRevisions == nil {
		c.meshRevisions = make(map[[3]int32]uint64)
	}
	for dz := int32(-1); dz <= 1; dz++ {
		for dx := int32(-1); dx <= 1; dx++ {
			for sy := int32(world.MinSectionY); sy <= world.MaxSectionY; sy++ {
				key := [3]int32{pos.X + dx, sy, pos.Z + dz}
				c.meshRevisions[key]++
				delete(c.meshCache, key)
			}
		}
	}
}

func affectedMeshSections(pos world.BlockPos) [][3]int32 {
	secX := int32(math.Floor(float64(pos.X) / 16.0))
	secY := int32(math.Floor(float64(pos.Y) / 16.0))
	secZ := int32(math.Floor(float64(pos.Z) / 16.0))
	remX := ((pos.X % 16) + 16) % 16
	remY := ((pos.Y % 16) + 16) % 16
	remZ := ((pos.Z % 16) + 16) % 16
	xSections := []int32{secX}
	ySections := []int32{secY}
	zSections := []int32{secZ}
	if remX == 0 {
		xSections = append(xSections, secX-1)
	} else if remX == 15 {
		xSections = append(xSections, secX+1)
	}
	if remY == 0 {
		ySections = append(ySections, secY-1)
	} else if remY == 15 {
		ySections = append(ySections, secY+1)
	}
	if remZ == 0 {
		zSections = append(zSections, secZ-1)
	} else if remZ == 15 {
		zSections = append(zSections, secZ+1)
	}
	keys := make([][3]int32, 0, len(xSections)*len(ySections)*len(zSections))
	for _, x := range xSections {
		for _, y := range ySections {
			for _, z := range zSections {
				keys = append(keys, [3]int32{x, y, z})
			}
		}
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
	py := c.player.Position.Y
	pz := c.player.Position.Z

	isWater := func(b int32) bool {
		return b == 8 || b == 9
	}

	bx := int32(math.Floor(px))
	bz := int32(math.Floor(pz))

	feetY := int32(math.Floor(py + 0.1))
	torsoY := int32(math.Floor(py + 0.8))
	belowY := int32(math.Floor(py - 0.2))
	headY := int32(math.Floor(py + 1.6))

	bFeet := c.engine.GetBlockIfLoaded(bx, feetY, bz)
	bTorso := c.engine.GetBlockIfLoaded(bx, torsoY, bz)
	bBelow := c.engine.GetBlockIfLoaded(bx, belowY, bz)
	bHead := c.engine.GetBlockIfLoaded(bx, headY, bz)

	waterFeet := isWater(bFeet)
	waterTorso := isWater(bTorso)
	waterBelow := isWater(bBelow) && (py-math.Floor(py)) < 0.35

	// Check 4 horizontal corners at feet level
	cornerWater := false
	for _, dx := range []float64{-0.25, 0.25} {
		for _, dz := range []float64{-0.25, 0.25} {
			cb := c.engine.GetBlockIfLoaded(int32(math.Floor(px+dx)), feetY, int32(math.Floor(pz+dz)))
			if isWater(cb) {
				cornerWater = true
				break
			}
		}
		if cornerWater {
			break
		}
	}

	inWater = waterFeet || waterTorso || cornerWater || waterBelow
	atSurface = inWater && !isWater(bHead)
	return inWater, atSurface
}

// PhysicsStep executes one delta-time scaled physics step with authentic Minecraft Java Edition physics.
func (c *Client) PhysicsStep(fwdInput, strafeInput float64, jumpRequested bool, dt float64) {
	if c.player == nil || c.player.Health <= 0 {
		return
	}

	// 0. Creative Flight Handling
	if c.isFlying {
		speed := 9.0
		var fwd, strafe float64
		lenInput := math.Hypot(fwdInput, strafeInput)
		if lenInput > 0 {
			fwd = (fwdInput / lenInput) * speed * dt
			strafe = (strafeInput / lenInput) * speed * dt
		}
		radYaw := c.camera.Yaw * math.Pi / 180.0
		dx := math.Sin(radYaw)*fwd + math.Cos(radYaw)*strafe
		dz := math.Cos(radYaw)*fwd - math.Sin(radYaw)*strafe
		dy := 0.0
		if jumpRequested {
			dy = speed * dt // Fly upward smoothly while holding space
		} else if c.isSneaking {
			dy = -speed * dt // Fly downward
		}
		c.player.Position.X += dx
		c.player.Position.Y += dy
		c.player.Position.Z += dz
		c.player.Velocity = sim.Vec3{X: 0, Y: 0, Z: 0}
		c.player.OnGround = false
		c.camera.X = c.player.Position.X
		c.camera.Y = c.player.Position.Y + 1.62
		c.camera.Z = c.player.Position.Z
		c.EnsureChunksAroundPlayer(c.player.Position.X, c.player.Position.Z, 3)
		return
	}

	// 0.5 Ladder Detection & Climbing (Authentic Minecraft: touching ladder allows climbing with +3.0 m/s when Space or W is held, and halts falling)
	onLadder := false
	if c.engine != nil {
		bx := int32(math.Floor(c.player.Position.X))
		by := int32(math.Floor(c.player.Position.Y))
		bz := int32(math.Floor(c.player.Position.Z))
		onLadder = c.engine.GetBlockIfLoaded(bx, by, bz) == 65 || c.engine.GetBlockIfLoaded(bx, by+1, bz) == 65
	}

	// 1. Water Detection & Aquatic Physics vs Air Gravity
	inWater, atSurface := c.checkWaterState()

	if onLadder {
		c.player.FallDistance = 0
		if jumpRequested || fwdInput > 0 {
			c.player.Velocity.Y = 3.0
		} else if c.isSneaking {
			c.player.Velocity.Y = 0 // Cling stationary to ladder when sneaking
		} else {
			c.player.Velocity.Y = -2.0 // Slow steady slide down ladder
		}
	} else if inWater {
		c.player.FallDistance = 0 // No fall damage in water
		if jumpRequested {
			// Swimming upward in water: continuous upward propulsion while holding Space
			if atSurface {
				// Surface breach: authentic Minecraft leap out of water onto riverbanks / land
				if c.player.Velocity.Y < 4.5 {
					c.player.Velocity.Y = 6.2
				}
			} else {
				// Submerged: steady continuous swim ascent (~4.2 m/s vanilla equilibrium)
				c.player.Velocity.Y += 24.0 * dt
				if c.player.Velocity.Y > 4.2 {
					c.player.Velocity.Y = 4.2
				}
				if c.player.Velocity.Y < 1.5 {
					c.player.Velocity.Y = 1.5
				}
			}
			c.player.OnGround = false
		} else if c.isSneaking {
			// Fast dive downward in water (Sneak / Shift)
			c.player.Velocity.Y = -4.0
		} else {
			// Gentle sinking drift in water (vanilla water buoyancy)
			c.player.Velocity.Y -= 6.0 * dt
			if c.player.Velocity.Y < -3.0 {
				c.player.Velocity.Y = -3.0
			}
		}
	} else {
		// 2. Air / Land Physics
		// Jumping: Authentic Minecraft initial upward impulse of 8.4 m/s (0.42 blocks/tick * 20)
		if jumpRequested && c.player.OnGround {
			c.player.Velocity.Y = 8.4
			c.player.OnGround = false
			if c.isSprinting {
				c.player.Exhaustion += 0.2
			} else {
				c.player.Exhaustion += 0.05
			}
		} else {
			// Air Gravity: Authentic Minecraft 32.0 m/s^2 (0.08 blocks/tick^2 * 400), terminal velocity -78.4 m/s
			c.player.Velocity.Y -= 32.0 * dt
			if c.player.Velocity.Y < -78.4 {
				c.player.Velocity.Y = -78.4
			}
		}
	}

	// 3. Horizontal Speeds: Walk 4.317 m/s, Sprint 5.612 m/s, Sneak/Eat 1.295 m/s, Swimming 2.2 m/s, Sprint-Swimming 3.6 m/s
	speed := 4.317
	if inWater {
		if c.isSprinting {
			speed = 3.6
		} else {
			speed = 2.2
		}
	} else if c.isSneaking || c.isEating {
		speed = 1.295
	} else if c.isSprinting {
		speed = 5.612
	}

	var fwd, strafe float64
	lenInput := math.Hypot(fwdInput, strafeInput)
	if lenInput > 0 {
		fwd = (fwdInput / lenInput) * speed * dt
		strafe = (strafeInput / lenInput) * speed * dt
	}

	radYaw := c.camera.Yaw * math.Pi / 180.0
	// Authentic Minecraft forward & right vectors:
	// Forward = (sin(Yaw), 0, cos(Yaw))
	// Right = (cos(Yaw), 0, -sin(Yaw))
	dx := math.Sin(radYaw)*fwd + math.Cos(radYaw)*strafe
	dz := math.Cos(radYaw)*fwd - math.Sin(radYaw)*strafe
	dy := c.player.Velocity.Y * dt

	// Sneak Cliff Edge Safeguard: prevent stepping off ledges when sneaking on ground
	if c.isSneaking && c.player.OnGround {
		checkGroundAt := func(px, pz float64) bool {
			feetY := int32(math.Floor(c.player.Position.Y - 0.5))
			minBX := int32(math.Floor(px - 0.25))
			maxBX := int32(math.Floor(px + 0.25))
			minBZ := int32(math.Floor(pz - 0.25))
			maxBZ := int32(math.Floor(pz + 0.25))
			for bx := minBX; bx <= maxBX; bx++ {
				for bz := minBZ; bz <= maxBZ; bz++ {
					if c.engine.GetBlockIfLoaded(bx, feetY, bz) != 0 {
						return true
					}
				}
			}
			return false
		}
		if !checkGroundAt(c.player.Position.X+dx, c.player.Position.Z) {
			dx = 0
		}
		if !checkGroundAt(c.player.Position.X, c.player.Position.Z+dz) {
			dz = 0
		}
	}

	delta := sim.Vec3{X: dx, Y: dy, Z: dz}
	resolvedPos, onGround, hitCeiling := c.engine.MovePlayerRelative(c.player.ID, delta, float32(c.camera.Yaw), float32(c.camera.Pitch))

	c.player.OnGround = onGround
	if onGround {
		if !inWater || !jumpRequested {
			c.player.Velocity.Y = 0
		}
	} else if hitCeiling && c.player.Velocity.Y > 0 {
		// Hit ceiling! Stop upward vertical momentum immediately
		c.player.Velocity.Y = 0
	}
	c.player.Position = resolvedPos

	// 4. Update Camera & Bobbing
	c.isMoving = (lenInput > 0)
	if c.isMoving && onGround {
		if c.isSprinting {
			c.player.Exhaustion += float32(0.1 * dt * 5.6)
		}
		c.walkCycle += 10.0 * dt // Authentic Minecraft walk cycle speed

		// Authentic Minecraft Footstep Sound Feedback
		stepInterval := 380 * time.Millisecond
		if c.isSprinting {
			stepInterval = 270 * time.Millisecond
		} else if c.isSneaking {
			stepInterval = 550 * time.Millisecond
		}
		if time.Since(c.lastStepTime) > stepInterval {
			c.lastStepTime = time.Now()
			bx := int32(math.Floor(resolvedPos.X))
			by := int32(math.Floor(resolvedPos.Y - 0.2))
			bz := int32(math.Floor(resolvedPos.Z))
			underBlock := c.engine.GetBlockIfLoaded(bx, by, bz)
			switch underBlock {
			case 1, 4: // Stone, Cobblestone
				sound.PlayHardwareFeedback(sound.SoundBlockStoneStep)
			case 5, 17: // Planks, Log
				sound.PlayHardwareFeedback(sound.SoundBlockWoodStep)
			default: // Grass, Dirt, etc.
				sound.PlayHardwareFeedback(sound.SoundBlockGrassStep)
			}
		}
	} else if onGround {
		c.walkCycle = 0
	}

	bobY := 0.0
	bobX := 0.0
	if onGround && c.isMoving {
		bobY = math.Sin(c.walkCycle) * 0.035
		bobX = math.Cos(c.walkCycle*0.5) * 0.02
	}

	// Dynamic Sneak Eye-Height transition (1.62m walking -> 1.27m sneaking)
	targetEye := 1.62
	if c.isSneaking {
		targetEye = 1.27
	}
	c.currentEyeHeight += (targetEye - c.currentEyeHeight) * math.Min(1.0, dt*14.0)

	eyeX := resolvedPos.X + bobX
	eyeY := resolvedPos.Y + c.currentEyeHeight + bobY
	eyeZ := resolvedPos.Z

	// Dynamic FOV for sprint speed rush feel
	if c.isSprinting && c.isMoving && onGround {
		c.targetFov = 82.0
	} else {
		c.targetFov = 70.0
	}
	c.camera.Fov += (c.targetFov - c.camera.Fov) * math.Min(1.0, dt*8.0)

	// Perspective camera positioning with obstacle raycast clipping (F5)
	lx, ly, lz := c.camera.GetLookVector()
	switch c.perspectiveMode {
	case 1: // Third-person back: 3.5 blocks behind with obstacle raycast clipping
		camDist := 3.5
		ray := sim.Raycast(sim.Vec3{X: eyeX, Y: eyeY, Z: eyeZ}, sim.Vec3{X: -lx, Y: -ly, Z: -lz}, 3.5, func(x, y, z int32) bool {
			return sim.IsSolidBlock(c.engine.GetBlockIfLoaded(x, y, z))
		})
		if ray.Hit {
			camDist = math.Max(0.25, ray.Distance-0.25)
		}
		c.camera.X = eyeX - lx*camDist
		c.camera.Y = eyeY - ly*camDist
		c.camera.Z = eyeZ - lz*camDist
	case 2: // Third-person front: 3.5 blocks in front with obstacle raycast clipping
		camDist := 3.5
		ray := sim.Raycast(sim.Vec3{X: eyeX, Y: eyeY, Z: eyeZ}, sim.Vec3{X: lx, Y: ly, Z: lz}, 3.5, func(x, y, z int32) bool {
			return sim.IsSolidBlock(c.engine.GetBlockIfLoaded(x, y, z))
		})
		if ray.Hit {
			camDist = math.Max(0.25, ray.Distance-0.25)
		}
		c.camera.X = eyeX + lx*camDist
		c.camera.Y = eyeY + ly*camDist
		c.camera.Z = eyeZ + lz*camDist
	default: // First-person
		c.camera.X = eyeX
		c.camera.Y = eyeY
		c.camera.Z = eyeZ
	}

	// Hurt flash & sound detection
	if c.player.Health < c.lastPlayerHealth && c.lastPlayerHealth > 0 {
		sound.PlayHardwareFeedback(sound.SoundPlayerHurt)
		c.hurtFlashTimer = 0.4
	}
	c.lastPlayerHealth = c.player.Health
	if c.hurtFlashTimer > 0 {
		c.hurtFlashTimer -= dt
	}

	// Portal immersion & dimension transition check
	if c.engine != nil {
		bx := int32(math.Floor(resolvedPos.X))
		by := int32(math.Floor(resolvedPos.Y))
		bz := int32(math.Floor(resolvedPos.Z))
		curBlock := c.engine.GetBlockIfLoaded(bx, by, bz)
		headBlock := c.engine.GetBlockIfLoaded(bx, by+1, bz)
		inNetherPortal := (curBlock == 90 || headBlock == 90)
		inEndPortal := (curBlock == 119 || headBlock == 119)

		if inEndPortal {
			if c.currentDim == gen.DimensionEnd {
				c.SwitchDimension(gen.DimensionOverworld)
			} else {
				c.SwitchDimension(gen.DimensionEnd)
			}
		} else if inNetherPortal {
			c.portalStandTime += dt
			if c.portalStandTime >= 1.5 {
				c.portalStandTime = 0
				if c.currentDim == gen.DimensionNether {
					c.SwitchDimension(gen.DimensionOverworld)
				} else {
					c.SwitchDimension(gen.DimensionNether)
				}
			}
		} else {
			c.portalStandTime = math.Max(0, c.portalStandTime-dt*2.0)
		}
	}

	// Stream chunks throttled (only when crossing chunk boundaries or periodically every 120ms)
	pcx := int32(math.Floor(resolvedPos.X / 16.0))
	pcz := int32(math.Floor(resolvedPos.Z / 16.0))
	nowTime := time.Now()
	if pcx != c.lastChunkX || pcz != c.lastChunkZ || nowTime.Sub(c.lastChunkStreamTime) > 120*time.Millisecond {
		c.lastChunkX = pcx
		c.lastChunkZ = pcz
		c.lastChunkStreamTime = nowTime
		c.EnsureChunksAroundPlayer(resolvedPos.X, resolvedPos.Z, 3)
	}

	// Send authoritative position update to multiplayer server
	if c.conn != nil && c.player != nil {
		posPkt := &protocol.PlayerPositionPacket{
			X:        resolvedPos.X,
			Y:        resolvedPos.Y,
			Z:        resolvedPos.Z,
			Yaw:      float32(c.camera.Yaw),
			Pitch:    float32(c.camera.Pitch),
			OnGround: c.player.OnGround,
		}
		if nowTime.Sub(c.lastNetworkPositionSend) >= 50*time.Millisecond {
			c.lastNetworkPositionSend = nowTime
			_ = c.sendPacket(posPkt)
		}
	}
}

func (c *Client) MoveRelative(forward, strafe float64) {
	c.PhysicsStep(forward, strafe, false, 0.05)
	c.publishPlayerSnapshot()
}

func (c *Client) Jump() {
	if c.player == nil {
		return
	}
	inWater, atSurface := c.checkWaterState()
	if inWater {
		if atSurface {
			c.player.Velocity.Y = 6.2
		} else {
			c.player.Velocity.Y = 4.2
		}
		c.player.OnGround = false
	} else if c.player.OnGround {
		c.player.Velocity.Y = 8.4
		c.player.OnGround = false
	}
}

func (c *Client) RotateView(dYaw, dPitch float64) {
	c.camera.Yaw += dYaw
	c.camera.Pitch += dPitch
	if c.camera.Pitch > 89.0 {
		c.camera.Pitch = 89.0
	}
	if c.camera.Pitch < -89.0 {
		c.camera.Pitch = -89.0
	}
}

func (c *Client) UpdateRaycastTarget() {
	if c.player == nil || c.engine == nil {
		c.hasTarget = false
		return
	}
	origin := sim.Vec3{X: c.camera.X, Y: c.camera.Y, Z: c.camera.Z}
	lx, ly, lz := c.camera.GetLookVector()
	dir := sim.Vec3{X: lx, Y: ly, Z: lz}

	isSolid := func(x, y, z int32) bool {
		b := c.engine.GetBlockIfLoaded(x, y, z)
		return b != 0 && b != 8 && b != 9 && b != 10 && b != 11
	}

	res := sim.Raycast(origin, dir, 5.0, isSolid)
	if res.Hit {
		c.hasTarget = true
		c.targetedPos = world.BlockPos{X: res.BlockX, Y: res.BlockY, Z: res.BlockZ}
		c.targetedFace = world.BlockPos{X: res.FaceX, Y: res.FaceY, Z: res.FaceZ}
	} else {
		c.hasTarget = false
	}
}

func (c *Client) UpdateMining(isLeftMouseDown bool, dt float64) {
	if c.player == nil || c.engine == nil || c.player.Inventory == nil {
		return
	}

	// Update block place cooldown
	if c.placeCooldown > 0 {
		c.placeCooldown -= dt
	}

	// Update active debris particles
	var activeParticles []render.Particle
	for i := range c.particles {
		p := &c.particles[i]
		p.Life -= dt
		if p.Life > 0 {
			p.X += p.Vx * dt
			p.Y += p.Vy * dt
			p.Z += p.Vz * dt
			p.Vy -= 12.0 * dt // Particle gravity
			activeParticles = append(activeParticles, *p)
		}
	}
	c.particles = activeParticles

	// Arm swing animation decay when not actively mining
	if !isLeftMouseDown {
		c.breakProgress = 0.0
		c.hasMiningBlock = false
		c.isMining = false
		if c.isSwinging {
			c.swingProgress += dt * 4.0
			if c.swingProgress >= 1.0 {
				c.isSwinging = false
				c.swingProgress = 0.0
			}
		}
		return
	}

	// Left click is held down: start or continue swinging arm
	c.isSwinging = true
	c.swingProgress += dt * 4.0 // Authentic rhythmic arm swing (4 swings per second)
	if c.swingProgress >= 1.0 {
		c.swingProgress -= 1.0
	}

	// Mining target resolution with authentic target-lock:
	// If already mining a block, stay locked to it as long as it remains within reach (~5.5 blocks)
	var activePos world.BlockPos
	if c.hasMiningBlock {
		blockPosVec := sim.Vec3{X: float64(c.miningPos.X) + 0.5, Y: float64(c.miningPos.Y) + 0.5, Z: float64(c.miningPos.Z) + 0.5}
		camPosVec := sim.Vec3{X: c.camera.X, Y: c.camera.Y, Z: c.camera.Z}
		dist := camPosVec.DistanceTo(blockPosVec)
		blockID := c.engine.GetBlockIfLoaded(c.miningPos.X, c.miningPos.Y, c.miningPos.Z)
		if dist <= 5.5 && blockID != 0 && blockID != 7 {
			activePos = c.miningPos
		} else {
			c.hasMiningBlock = false
			c.breakProgress = 0.0
		}
	}

	if !c.hasMiningBlock {
		if !c.hasTarget {
			c.breakProgress = 0.0
			c.isMining = false
			return
		}
		blockID := c.engine.GetBlockIfLoaded(c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
		if blockID == 0 || blockID == 7 { // Air or Bedrock
			c.breakProgress = 0.0
			c.isMining = false
			return
		}
		activePos = c.targetedPos
		c.miningPos = activePos
		c.hasMiningBlock = true
		c.breakProgress = 0.0
	}

	blockID := c.engine.GetBlockIfLoaded(activePos.X, activePos.Y, activePos.Z)
	if blockID == 0 || blockID == 7 {
		c.breakProgress = 0.0
		c.hasMiningBlock = false
		c.isMining = false
		return
	}

	c.isMining = true
	heldToolID := int32(0)
	if sel := c.player.Inventory.GetSelectedItem(); !sel.IsEmpty() {
		heldToolID = sel.ItemID
	}

	dps := inventory.CalculateBlockDamagePerSecond(heldToolID, blockID)
	if c.isCreative {
		c.breakProgress = 1.0 // Instant block destruction in Creative Mode
	} else {
		c.breakProgress += dps * dt
	}

	// Periodically emit hitting sounds and small debris particles
	if time.Since(c.lastHitSoundTime) > 220*time.Millisecond {
		c.lastHitSoundTime = time.Now()
		sound.PlayHardwareFeedback("minecraft:block.hit")
		c.spawnBlockDebris(activePos, blockID, 3)
	}

	// Block breaking completion threshold (100% progress)
	if c.breakProgress >= 1.0 {
		minedPos := activePos
		c.engine.BreakBlockWithTool(c.player.ID, minedPos, heldToolID)
		c.spawnBlockDebris(minedPos, blockID, 24) // Explosion of particles
		sound.PlayHardwareFeedback(sound.GetBlockBreakSound(blockID))

		c.breakProgress = 0.0
		c.hasMiningBlock = false
		c.isMining = false
		c.RebuildAffectedMeshes(minedPos)
	}
}

func (c *Client) spawnBlockDebris(pos world.BlockPos, blockID int32, count int) {
	col := render.SampleBlockColor(blockID, 2, 0.5, 0.5, 1.0)
	for i := 0; i < count; i++ {
		rx := (randFloat() - 0.5) * 0.8
		ry := (randFloat() - 0.5) * 0.8
		rz := (randFloat() - 0.5) * 0.8
		vx := (randFloat() - 0.5) * 3.5
		vy := randFloat()*3.0 + 1.2
		vz := (randFloat() - 0.5) * 3.5
		c.particles = append(c.particles, render.Particle{
			X:       float64(pos.X) + 0.5 + rx,
			Y:       float64(pos.Y) + 0.5 + ry,
			Z:       float64(pos.Z) + 0.5 + rz,
			Vx:      vx,
			Vy:      vy,
			Vz:      vz,
			Color:   col,
			Life:    0.4 + randFloat()*0.3,
			MaxLife: 0.7,
		})
	}
}

func randFloat() float64 {
	var b [1]byte
	rand.Read(b[:])
	return float64(b[0]) / 255.0
}

// AddChatMessage appends a new message to the client's in-game chat history.
func (c *Client) AddChatMessage(text string) {
	c.chatMessages = append(c.chatMessages, render.ChatMessage{
		Text:      text,
		Timestamp: time.Now(),
	})
	if len(c.chatMessages) > 50 {
		c.chatMessages = c.chatMessages[len(c.chatMessages)-50:]
	}
}

func (c *Client) UnlockAdvancement(id, title string) {
	if c.advancements == nil {
		c.advancements = make(map[string]bool)
	}
	if c.advancements[id] {
		return
	}
	c.advancements[id] = true
	c.AddChatMessage("[进度已达成] " + title)
}

func (c *Client) advancementIDs() []string {
	ids := make([]string, 0, len(c.advancements))
	for id := range c.advancements {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (c *Client) publishPlayerSnapshot() {
	if c.engine == nil || c.player == nil {
		return
	}
	mode := 0
	if c.isCreative {
		mode = 1
	}
	c.engine.UpdateClientPlayerSnapshot(c.player.ID, c.player.Position, mode)
}

// AdvanceSimulation runs client-owned world simulation on the render thread.
// Keeping entity ticks and rendering on one thread avoids races on mutable world state.
func (c *Client) AdvanceSimulation(dt float64) {
	if c.engine == nil || c.screenState != ScreenPlaying || dt <= 0 {
		return
	}
	c.tickAccumulator += dt
	const tickDuration = 0.05
	for c.tickAccumulator >= tickDuration {
		c.engine.Tick()
		c.tickAccumulator -= tickDuration
	}
}

func (c *Client) advancementLines() []string {
	definitions := []struct{ ID, Title string }{
		{"minecraft:story/root", "进入世界"},
		{"minecraft:story/mine_wood", "获取木头"},
		{"minecraft:story/mine_stone", "石器时代"},
		{"minecraft:story/upgrade_tools", "装配升级"},
		{"minecraft:story/mine_diamond", "钻石！"},
		{"minecraft:story/smelt_iron", "来硬的"},
		{"minecraft:story/iron_tools", "不再是木镐"},
		{"minecraft:story/obtain_armor", "穿上盔甲"},
		{"minecraft-go:story/place_crafting_table", "搭建工作台"},
		{"minecraft:nether/root", "进入下界"},
		{"minecraft:story/enter_the_end", "进入末地"},
		{"minecraft:end/kill_dragon", "解放末地"},
	}
	lines := make([]string, 0, len(definitions))
	for _, advancement := range definitions {
		if c.advancements[advancement.ID] {
			lines = append(lines, "✓  "+advancement.Title)
		} else {
			lines = append(lines, "○  "+advancement.Title)
		}
	}
	return lines
}

// ExecuteCommand parses and executes in-game commands or sends player chat.
func (c *Client) ExecuteCommand(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	if !strings.HasPrefix(input, "/") {
		// Normal player chat
		c.AddChatMessage("<" + c.username + "> " + input)
		return
	}

	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/help":
		c.AddChatMessage("Commands: /gamemode <c|s>, /fly, /time set <day|noon|night|midnight>, /give <item> [n], /tp <x> <y> <z>, /clear")

	case "/gamemode":
		if len(parts) < 2 {
			c.AddChatMessage("Usage: /gamemode <creative|survival|c|s|1|0>")
			return
		}
		mode := strings.ToLower(parts[1])
		if mode == "creative" || mode == "c" || mode == "1" {
			c.isCreative = true
			c.AddChatMessage("[GameMode] Set own game mode to Creative Mode")
		} else if mode == "survival" || mode == "s" || mode == "0" {
			c.isCreative = false
			c.isFlying = false
			c.AddChatMessage("[GameMode] Set own game mode to Survival Mode")
		} else {
			c.AddChatMessage("Unknown game mode: " + parts[1])
		}

	case "/fly":
		if !c.isCreative {
			c.isCreative = true
		}
		c.isFlying = !c.isFlying
		if c.isFlying {
			c.AddChatMessage("[Flight] Creative flight enabled (Space=Up, Shift=Down)")
		} else {
			c.AddChatMessage("[Flight] Creative flight disabled")
		}

	case "/time":
		if len(parts) >= 3 && strings.ToLower(parts[1]) == "set" {
			val := strings.ToLower(parts[2])
			if c.engine != nil {
				switch val {
				case "day":
					c.engine.SetWorldTime(1000)
					c.AddChatMessage("[Time] Set the time to 1000 (Day)")
				case "noon":
					c.engine.SetWorldTime(6000)
					c.AddChatMessage("[Time] Set the time to 6000 (Noon)")
				case "sunset":
					c.engine.SetWorldTime(12000)
					c.AddChatMessage("[Time] Set the time to 12000 (Sunset)")
				case "night":
					c.engine.SetWorldTime(13000)
					c.AddChatMessage("[Time] Set the time to 13000 (Night)")
				case "midnight":
					c.engine.SetWorldTime(18000)
					c.AddChatMessage("[Time] Set the time to 18000 (Midnight)")
				default:
					if t, err := strconv.ParseInt(val, 10, 64); err == nil {
						c.engine.SetWorldTime(t)
						c.AddChatMessage(fmt.Sprintf("[Time] Set the time to %d", t))
					} else {
						c.AddChatMessage("Usage: /time set <day|noon|sunset|night|midnight|<ticks>>")
					}
				}
			}
			return
		} else {
			c.AddChatMessage("Usage: /time set <day|noon|sunset|night|midnight|<ticks>>")
		}

	case "/give":
		if len(parts) < 2 {
			c.AddChatMessage("Usage: /give <item> [count]")
			return
		}
		itemStr := strings.ToLower(parts[1])
		count := 1
		if len(parts) >= 3 {
			if n, err := strconv.Atoi(parts[2]); err == nil && n > 0 {
				count = n
			}
		}

		var itemID int32
		switch itemStr {
		case "stone":
			itemID = 1
		case "grass", "grass_block":
			itemID = 2
		case "dirt":
			itemID = 3
		case "cobblestone", "cobble":
			itemID = 4
		case "planks", "wood_planks", "oak_planks":
			itemID = 5
		case "log", "oak_log", "wood":
			itemID = 17
		case "leaves", "oak_leaves":
			itemID = 18
		case "diamond":
			itemID = 264
		case "iron", "iron_ingot":
			itemID = 265
		case "gold", "gold_ingot":
			itemID = 266
		case "apple":
			itemID = inventory.ItemApple
		case "bread":
			itemID = inventory.ItemBread
		case "porkchop", "cooked_porkchop":
			itemID = inventory.ItemCookedPorkchop
		case "beef", "cooked_beef", "steak":
			itemID = inventory.ItemSteak
		case "pickaxe", "diamond_pickaxe":
			itemID = inventory.ItemDiamondPickaxe
		case "sword", "diamond_sword":
			itemID = 276
		case "crafting_table", "workbench":
			itemID = 58
		case "furnace":
			itemID = 61
		default:
			if id, err := strconv.Atoi(itemStr); err == nil && id > 0 {
				itemID = int32(id)
			} else {
				c.AddChatMessage("Unknown item: " + itemStr)
				return
			}
		}

		if c.player != nil && c.player.Inventory != nil {
			c.player.Inventory.AddItem(itemID, count)
			c.AddChatMessage(fmt.Sprintf("[Give] Gave %d [%s] to %s", count, itemStr, c.username))
			sound.PlayHardwareFeedback("minecraft:item.pickup")
		}

	case "/tp":
		if len(parts) < 4 {
			c.AddChatMessage("Usage: /tp <x> <y> <z>")
			return
		}
		x, err1 := strconv.ParseFloat(parts[1], 64)
		y, err2 := strconv.ParseFloat(parts[2], 64)
		z, err3 := strconv.ParseFloat(parts[3], 64)
		if err1 == nil && err2 == nil && err3 == nil && c.player != nil {
			c.player.Position = sim.Vec3{X: x, Y: y, Z: z}
			c.camera.X = x
			c.camera.Y = y + 1.62
			c.camera.Z = z
			c.EnsureChunksAroundPlayer(x, z, 3)
			c.AddChatMessage(fmt.Sprintf("[Teleport] Teleported %s to (%.1f, %.1f, %.1f)", c.username, x, y, z))
		}

	case "/dimension":
		if len(parts) < 2 {
			c.AddChatMessage("Usage: /dimension <overworld|nether|end>")
			return
		}
		target := strings.ToLower(parts[1])
		switch target {
		case "overworld", "0":
			c.SwitchDimension(gen.DimensionOverworld)
		case "nether", "1":
			c.SwitchDimension(gen.DimensionNether)
		case "end", "2":
			c.SwitchDimension(gen.DimensionEnd)
		default:
			c.AddChatMessage("Unknown dimension: " + parts[1])
		}

	case "/kill":
		if len(parts) >= 2 && (parts[1] == "@e[type=ender_dragon]" || parts[1] == "dragon") {
			if c.engine != nil && c.engine.GetEnderDragon() != nil {
				c.engine.DamageEnderDragon(200.0)
				c.AddChatMessage("[Kill] Slew Ender Dragon!")
				if dragon := c.engine.GetEnderDragon(); dragon != nil && !dragon.IsAlive {
					c.UnlockAdvancement("minecraft:end/kill_dragon", "解放末地")
				}
			}
			return
		}
		if c.player != nil {
			c.engine.DamagePlayer(c.player.ID, 100.0, "command")
			c.AddChatMessage("[Kill] Ouch! That looks like it hurt.")
		}

	case "/clear":
		if c.player != nil && c.player.Inventory != nil {
			for i := range c.player.Inventory.Slots {
				c.player.Inventory.Slots[i] = inventory.ItemStack{}
			}
			c.AddChatMessage("[Inventory] Cleared player inventory")
		}

	default:
		c.AddChatMessage("Unknown command. Type /help for help.")
	}
}

func (c *Client) MineTargetedBlock() (int32, bool) {
	if !c.hasTarget || c.player == nil || c.engine == nil || c.player.Inventory == nil {
		return 0, false
	}
	heldToolID := int32(0)
	if sel := c.player.Inventory.GetSelectedItem(); !sel.IsEmpty() {
		heldToolID = sel.ItemID
	}
	targetPos := c.targetedPos
	blockID := c.engine.GetBlockIfLoaded(targetPos.X, targetPos.Y, targetPos.Z)
	drop, ok := c.engine.BreakBlockWithTool(c.player.ID, targetPos, heldToolID)
	if ok && blockID == 1 {
		c.UnlockAdvancement("minecraft:story/mine_stone", "石器时代")
	}
	if ok && (blockID == 17 || blockID == 162) {
		c.UnlockAdvancement("minecraft:story/mine_wood", "获取木头")
	}
	c.spawnBlockDebris(targetPos, blockID, 20)
	sound.PlayHardwareFeedback(sound.GetBlockBreakSound(blockID))
	c.RebuildAffectedMeshes(targetPos)
	if ok && c.conn != nil {
		_ = c.sendPacket(&protocol.BlockUpdatePacket{
			X:       targetPos.X,
			Y:       targetPos.Y,
			Z:       targetPos.Z,
			BlockID: 0,
		})
	}
	return drop, ok
}

func (c *Client) PlaceBlockInFront() bool {
	if !c.hasTarget || c.player == nil || c.engine == nil || c.player.Inventory == nil {
		return false
	}
	selected := c.player.Inventory.GetSelectedItem()
	if selected.IsEmpty() {
		return false
	}
	if !inventory.IsPlaceableBlock(selected.ItemID) {
		return false
	}

	origin := sim.Vec3{X: c.camera.X, Y: c.camera.Y, Z: c.camera.Z}
	lx, ly, lz := c.camera.GetLookVector()
	dir := sim.Vec3{X: lx, Y: ly, Z: lz}

	isSolid := func(x, y, z int32) bool {
		b := c.engine.GetBlockIfLoaded(x, y, z)
		return b != 0 && b != 8 && b != 9 && b != 10 && b != 11
	}
	res := sim.Raycast(origin, dir, 5.0, isSolid)
	if !res.Hit {
		return false
	}

	hitBlockID := c.engine.GetBlockIfLoaded(res.BlockX, res.BlockY, res.BlockZ)
	var targetPos world.BlockPos
	if inventory.IsReplaceableBlock(hitBlockID) {
		// Replaceable grass/flowers are directly replaced at their position
		targetPos = world.BlockPos{X: res.BlockX, Y: res.BlockY, Z: res.BlockZ}
		if sim.IsSolidBlock(selected.ItemID) {
			bBox := sim.NewBlockAABB(targetPos.X, targetPos.Y, targetPos.Z)
			pBox := sim.NewPlayerAABB(c.player.Position)
			if sim.PlacementIntersects(pBox, bBox) {
				// Player is standing in this foliage! Pass through foliage to hit the solid block underneath/behind
				solidRes := sim.Raycast(origin, dir, 5.0, func(x, y, z int32) bool {
					b := c.engine.GetBlockIfLoaded(x, y, z)
					return sim.IsSolidBlock(b)
				})
				if solidRes.Hit {
					targetPos = world.BlockPos{
						X: solidRes.BlockX + solidRes.FaceX,
						Y: solidRes.BlockY + solidRes.FaceY,
						Z: solidRes.BlockZ + solidRes.FaceZ,
					}
				}
			}
		}
	} else {
		// Solid block: place against exposed face
		targetPos = world.BlockPos{
			X: res.BlockX + res.FaceX,
			Y: res.BlockY + res.FaceY,
			Z: res.BlockZ + res.FaceZ,
		}
	}

	ok := c.engine.PlaceBlock(c.player.ID, targetPos, selected.ItemID)
	if ok {
		if selected.ItemID == 58 {
			c.UnlockAdvancement("minecraft-go:story/place_crafting_table", "搭建工作台")
		}
		sound.PlayHardwareFeedback(sound.GetBlockPlaceSound(selected.ItemID))
		c.RebuildAffectedMeshes(targetPos)
		c.isSwinging = true
		c.swingProgress = 0.0
		if c.conn != nil {
			_ = c.sendPacket(&protocol.BlockUpdatePacket{
				X:       targetPos.X,
				Y:       targetPos.Y,
				Z:       targetPos.Z,
				BlockID: selected.ItemID,
			})
		}
	}
	return ok
}

// UpdateRightClick handles eating food and placing blocks/interacting with blocks.
func (c *Client) UpdateRightClick(isRightDown bool, dt float64) {
	if c.player == nil || c.engine == nil || c.player.Inventory == nil {
		return
	}

	sel := c.player.Inventory.GetSelectedItem()
	if egg, ok := inventory.SpawnEggByID(sel.ItemID); ok {
		if isRightDown && c.isLocal && c.placeCooldown <= 0 && c.hasTarget {
			ground := c.engine.GetBlockIfLoaded(c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
			spawnY := c.targetedPos.Y + 1
			feet := c.engine.GetBlockIfLoaded(c.targetedPos.X, spawnY, c.targetedPos.Z)
			head := c.engine.GetBlockIfLoaded(c.targetedPos.X, spawnY+1, c.targetedPos.Z)
			if world.ValidBlockCoordinates(c.targetedPos.X, spawnY+1, c.targetedPos.Z) &&
				sim.IsSolidBlock(ground) && !sim.IsSolidBlock(feet) && !sim.IsSolidBlock(head) {
				if mobType, supported := spawnEggMobType(egg.EntityName); supported {
					c.engine.SpawnMob(mobType, sim.Vec3{
						X: float64(c.targetedPos.X) + 0.5,
						Y: float64(spawnY),
						Z: float64(c.targetedPos.Z) + 0.5,
					})
					if !c.isCreative {
						slot := &c.player.Inventory.Slots[c.player.Inventory.SelectedHotbar]
						slot.Count--
						if slot.Count <= 0 {
							*slot = inventory.ItemStack{}
						}
					}
					c.placeCooldown = 0.3
				}
			}
		}
		return
	}
	if sel.ItemID == inventory.ItemBoneMeal {
		if isRightDown && c.isLocal && c.placeCooldown <= 0 && c.hasTarget {
			if grown, ok := c.engine.UseBoneMealOnRedShrub(c.player.ID, c.targetedPos); ok {
				c.RebuildAffectedMeshes(grown)
				c.spawnBlockDebris(grown, 10100, 6)
				sound.PlayHardwareFeedback("minecraft:item.bone_meal.use")
				c.placeCooldown = 0.3
			}
		}
		return
	}
	if isRightDown && c.isLocal && c.placeCooldown <= 0 && c.hasTarget {
		if props, ok := inventory.GetToolProperties(sel.ItemID); ok && props.Type == inventory.ToolAxe {
			blockID := c.engine.GetBlockIfLoaded(c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
			if strippedID, canStrip := inventory.StrippedWoodBlockVariant(blockID); canStrip &&
				c.engine.SetBlockChecked(c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z, strippedID) {
				if !c.isCreative {
					slot := &c.player.Inventory.Slots[c.player.Inventory.SelectedHotbar]
					inventory.ApplyToolWear(slot)
				}
				c.RebuildAffectedMeshes(c.targetedPos)
				c.placeCooldown = 0.3
				return
			}
		}
	}

	// 1. Food Consumption
	if !sel.IsEmpty() && inventory.IsFood(sel.ItemID) && c.player.Hunger < 20 {
		if isRightDown {
			c.isEating = true
			c.eatTimer += dt

			// Periodic munching crunch sound (every 0.22s)
			if time.Since(c.lastMunchSound) > 220*time.Millisecond {
				c.lastMunchSound = time.Now()
				sound.PlayHardwareFeedback("minecraft:player.eat")
				c.spawnBlockDebris(world.BlockPos{
					X: int32(math.Floor(c.camera.X)),
					Y: int32(math.Floor(c.camera.Y)),
					Z: int32(math.Floor(c.camera.Z)),
				}, sel.ItemID, 2)
			}

			// Eat complete (1.6s canonical Minecraft duration)
			if c.eatTimer >= 1.6 {
				c.engine.ConsumeFood(c.player.ID, c.player.Inventory.SelectedHotbar)
				sound.PlayHardwareFeedback("minecraft:player.burp")
				c.isEating = false
				c.eatTimer = 0.0
			}
			return
		} else {
			c.isEating = false
			c.eatTimer = 0.0
		}
	} else {
		c.isEating = false
		c.eatTimer = 0.0
	}

	// 2. Block Placement or Interactive Block Clicking (when Right Click is held)
	if isRightDown {
		if c.placeCooldown <= 0 {
			// Check if clicking on Crafting Table or Furnace
			if c.hasTarget {
				targetID := c.engine.GetBlockIfLoaded(c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
				if targetID == 58 { // Crafting Table
					c.showInventoryGUI = true
					c.isCraftingTableOpen = true
					c.isFurnaceOpen = false
					if c.wndRef != nil {
						c.wndRef.SetMouseCaptured(false)
					}
					c.placeCooldown = 0.3
					return
				}
				if targetID == 61 || targetID == 62 { // Furnace / Lit Furnace
					c.showInventoryGUI = false
					c.isCraftingTableOpen = false
					c.isFurnaceOpen = true
					c.currentFurnacePos = c.targetedPos
					if c.wndRef != nil {
						c.wndRef.SetMouseCaptured(false)
					}
					c.placeCooldown = 0.3
					return
				}
			}

			if c.PlaceBlockInFront() {
				c.placeCooldown = 0.25
				sound.PlayHardwareFeedback(sound.GetBlockPlaceSound(sel.ItemID))
			}
		}
	}
}

func spawnEggMobType(entityName string) (sim.MobType, bool) {
	switch entityName {
	case "zombie":
		return sim.MobZombie, true
	case "skeleton":
		return sim.MobSkeleton, true
	case "creeper":
		return sim.MobCreeper, true
	case "cow":
		return sim.MobCow, true
	case "pig":
		return sim.MobPig, true
	case "sheep":
		return sim.MobSheep, true
	case "chicken":
		return sim.MobChicken, true
	case "enderman":
		return sim.MobEnderman, true
	case "spider":
		return sim.MobSpider, true
	default:
		return 0, false
	}
}

// renderFrameSafely keeps a transient renderer panic from immediately terminating the native client.
func (c *Client) renderFrameSafely() bool {
	defer func() {
		if recovered := recover(); recovered != nil {
			c.renderPanicCount++
			msg := fmt.Sprintf("CLIENT RENDER PANIC (%d consecutive frame failures): %v\n\nStack:\n%s", c.renderPanicCount, recovered, debug.Stack())
			log.Print(msg)
			if c.renderPanicCount == 1 || c.renderPanicCount == 3 {
				writeCrashLog(msg)
			}
			if c.renderPanicCount == 3 && c.wndRef != nil {
				log.Printf("[Client] Closing the window after repeated render failures; save shutdown will run.")
				func() {
					defer func() { _ = recover() }()
					c.wndRef.Close()
				}()
			}
		}
	}()
	c.RenderFrame()
	if c.networkDisconnected.Load() {
		msg := "服务器连接已断开"
		textWidth := render.MeasureText5x7(msg, 2)
		c.framebuf.DrawText5x7((c.framebuf.Width-textWidth)/2, 10, msg, 0xFFFF7777, 2)
	}
	c.renderPanicCount = 0
	return true
}

// RenderFrame renders Minecraft sky, perspective-correct textured voxels, block outline, animated hand, and HUD.
func (c *Client) RenderFrame() {
	c.processLocalWorldEvents()
	c.UpdateRaycastTarget()

	// Hotbar equip animation trigger on slot change
	if c.player != nil && c.player.Inventory != nil {
		if c.player.Inventory.SelectedHotbar != c.lastSelectedSlot {
			c.lastSelectedSlot = c.player.Inventory.SelectedHotbar
			c.equipProgress = 0.0
		}
	}

	// 1. Sky gradient background with dynamic day/night celestial simulation & distance fog
	worldTime := int64(6000)
	if c.engine != nil {
		worldTime = c.engine.GetWorldTime()
	}
	c.framebuf.ClearSkyWithCamera(worldTime, c.camera)
	c.framebuf.DrawCelestialBodies(c.camera, worldTime)
	c.framebuf.DrawClouds(c.camera, worldTime)

	// 2. Render cached 3D chunk meshes in opaque and sorted translucent passes.
	c.meshMu.RLock()
	camX, camY, camZ := c.camera.X, c.camera.Y, c.camera.Z

	type visibleSection struct {
		mesh *render.Mesh
	}
	visibleSections := make([]visibleSection, 0, len(c.meshCache))
	for key, mesh := range c.meshCache {
		if mesh == nil {
			continue
		}

		// Conservatively reject sections outside the camera frustum before walking
		// their triangle indices; the sphere encloses the full 16x16x16 section.
		scx := float64(key[0]*16 + 8)
		scy := float64(key[1]*16 + 8)
		scz := float64(key[2]*16 + 8)
		if !c.camera.SphereInFrustum(scx, scy, scz, 13.856406460551018, c.framebuf.Width, c.framebuf.Height, 0.10) {
			continue
		}
		visibleSections = append(visibleSections, visibleSection{mesh: mesh})
	}
	c.meshMu.RUnlock()

	// Pass 1: Render opaque and alpha-cutout terrain before translucent blocks.
	for _, sec := range visibleSections {
		mesh := sec.mesh
		for i := 0; i+2 < len(mesh.Indices); i += 3 {
			v0, v1, v2, valid := meshTriangle(mesh, i)
			if !valid {
				continue
			}
			if render.IsTranslucentBlock(v0.BlockID) {
				continue // Water, glass, and portals are composited after opaque terrain.
			}

			// World-space exact normal culling for planar voxel face
			switch v0.FaceIndex {
			case 0: // +X face (visible only if camX > v0.X)
				if camX <= float64(v0.X) {
					continue
				}
			case 1: // -X face (visible only if camX < v0.X)
				if camX >= float64(v0.X) {
					continue
				}
			case 2: // +Y face (top, visible only if camY > v0.Y)
				if camY <= float64(v0.Y) {
					continue
				}
			case 3: // -Y face (bottom, visible only if camY < v0.Y)
				if camY >= float64(v0.Y) {
					continue
				}
			case 4: // +Z face (visible only if camZ > v0.Z)
				if camZ <= float64(v0.Z) {
					continue
				}
			case 5: // -Z face (visible only if camZ < v0.Z)
				if camZ >= float64(v0.Z) {
					continue
				}
			}

			vx0, vy0, vz0 := c.camera.ToViewSpace(float64(v0.X), float64(v0.Y), float64(v0.Z))
			vx1, vy1, vz1 := c.camera.ToViewSpace(float64(v1.X), float64(v1.Y), float64(v1.Z))
			vx2, vy2, vz2 := c.camera.ToViewSpace(float64(v2.X), float64(v2.Y), float64(v2.Z))

			vv0 := render.ViewVertex{X: vx0, Y: vy0, Z: vz0, U: v0.U, V: v0.V, BlockID: v0.BlockID, FaceIndex: v0.FaceIndex, LightingMult: v0.LightingMult}
			vv1 := render.ViewVertex{X: vx1, Y: vy1, Z: vz1, U: v1.U, V: v1.V, BlockID: v1.BlockID, FaceIndex: v1.FaceIndex, LightingMult: v1.LightingMult}
			vv2 := render.ViewVertex{X: vx2, Y: vy2, Z: vz2, U: v2.U, V: v2.V, BlockID: v2.BlockID, FaceIndex: v2.FaceIndex, LightingMult: v2.LightingMult}

			clipped, count := render.ClipTriangleNear(vv0, vv1, vv2, 0.10)
			for ci := 0; ci < count; ci++ {
				tri := clipped[ci]
				p0 := c.camera.ProjectViewToScreen(tri.V0.X, tri.V0.Y, tri.V0.Z, c.framebuf.Width, c.framebuf.Height)
				p1 := c.camera.ProjectViewToScreen(tri.V1.X, tri.V1.Y, tri.V1.Z, c.framebuf.Width, c.framebuf.Height)
				p2 := c.camera.ProjectViewToScreen(tri.V2.X, tri.V2.Y, tri.V2.Z, c.framebuf.Width, c.framebuf.Height)

				sv0 := render.NewScreenVertex(p0, tri.V0.U, tri.V0.V, tri.V0.BlockID, tri.V0.FaceIndex, tri.V0.LightingMult)
				sv1 := render.NewScreenVertex(p1, tri.V1.U, tri.V1.V, tri.V1.BlockID, tri.V1.FaceIndex, tri.V1.LightingMult)
				sv2 := render.NewScreenVertex(p2, tri.V2.U, tri.V2.V, tri.V2.BlockID, tri.V2.FaceIndex, tri.V2.LightingMult)

				c.framebuf.DrawVoxelTrianglePerspective(sv0, sv1, sv2)
			}
		}
	}

	// 3. Draw targeted block wireframe outline and progressive destruction cracks
	if c.hasTarget {
		c.framebuf.DrawBlockOutline(c.camera, c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
		if c.isMining && c.breakProgress > 0 {
			stage := int(c.breakProgress * 10)
			if stage > 9 {
				stage = 9
			}
			c.framebuf.DrawBlockDamage(c.camera, c.miningPos.X, c.miningPos.Y, c.miningPos.Z, stage)
		}
	}

	// 3.5 Render 3D physical debris particles
	if len(c.particles) > 0 {
		c.framebuf.DrawParticles(c.camera, c.particles)
	}

	// 3.6 Render floating 3D item drops in the world
	if c.engine != nil {
		drops := c.engine.GetItemDrops()
		for _, drop := range drops {
			c.drawItemDrop(drop)
		}
	}

	// 3.7 Render 3D living mobs (Zombies, Creepers, Cows, Pigs, Skeletons, Endermen, Spiders)
	if c.engine != nil {
		mobs := c.engine.GetMobs()
		for i := range mobs {
			c.drawMob(&mobs[i])
		}
		if c.currentDim == gen.DimensionEnd {
			c.drawEndEntities()
		}
	}

	// 3.8 Pass 2: collect transparent faces and sort individual clipped triangles
	// back-to-front. Section-center sorting can reverse glass/water surfaces
	// inside the same chunk and made the result depend on map iteration order.
	type translucentTriangle struct {
		v0, v1, v2 render.ScreenVertex
		depth      float64
		blockID    int32
		faceIndex  int
	}
	translucent := make([]translucentTriangle, 0)
	for _, sec := range visibleSections {
		mesh := sec.mesh
		for i := 0; i+2 < len(mesh.Indices); i += 3 {
			v0, v1, v2, valid := meshTriangle(mesh, i)
			if !valid {
				continue
			}
			if !render.IsTranslucentBlock(v0.BlockID) {
				continue
			}

			// Exact face normal culling for translucent block faces.
			switch v0.FaceIndex {
			case 0: // +X face (visible only if camX > v0.X)
				if camX <= float64(v0.X) {
					continue
				}
			case 1: // -X face (visible only if camX < v0.X)
				if camX >= float64(v0.X) {
					continue
				}
			case 2: // +Y face (top, visible only if camY > v0.Y)
				if camY <= float64(v0.Y) {
					continue
				}
			case 3: // -Y face (bottom, visible only if camY < v0.Y)
				if camY >= float64(v0.Y) {
					continue
				}
			case 4: // +Z face (visible only if camZ > v0.Z)
				if camZ <= float64(v0.Z) {
					continue
				}
			case 5: // -Z face (visible only if camZ < v0.Z)
				if camZ >= float64(v0.Z) {
					continue
				}
			}

			vx0, vy0, vz0 := c.camera.ToViewSpace(float64(v0.X), float64(v0.Y), float64(v0.Z))
			vx1, vy1, vz1 := c.camera.ToViewSpace(float64(v1.X), float64(v1.Y), float64(v1.Z))
			vx2, vy2, vz2 := c.camera.ToViewSpace(float64(v2.X), float64(v2.Y), float64(v2.Z))

			vv0 := render.ViewVertex{X: vx0, Y: vy0, Z: vz0, U: v0.U, V: v0.V, BlockID: v0.BlockID, FaceIndex: v0.FaceIndex, LightingMult: v0.LightingMult}
			vv1 := render.ViewVertex{X: vx1, Y: vy1, Z: vz1, U: v1.U, V: v1.V, BlockID: v1.BlockID, FaceIndex: v1.FaceIndex, LightingMult: v1.LightingMult}
			vv2 := render.ViewVertex{X: vx2, Y: vy2, Z: vz2, U: v2.U, V: v2.V, BlockID: v2.BlockID, FaceIndex: v2.FaceIndex, LightingMult: v2.LightingMult}

			clipped, count := render.ClipTriangleNear(vv0, vv1, vv2, 0.10)
			for ci := 0; ci < count; ci++ {
				tri := clipped[ci]
				p0 := c.camera.ProjectViewToScreen(tri.V0.X, tri.V0.Y, tri.V0.Z, c.framebuf.Width, c.framebuf.Height)
				p1 := c.camera.ProjectViewToScreen(tri.V1.X, tri.V1.Y, tri.V1.Z, c.framebuf.Width, c.framebuf.Height)
				p2 := c.camera.ProjectViewToScreen(tri.V2.X, tri.V2.Y, tri.V2.Z, c.framebuf.Width, c.framebuf.Height)

				sv0 := render.NewScreenVertex(p0, tri.V0.U, tri.V0.V, tri.V0.BlockID, tri.V0.FaceIndex, tri.V0.LightingMult)
				sv1 := render.NewScreenVertex(p1, tri.V1.U, tri.V1.V, tri.V1.BlockID, tri.V1.FaceIndex, tri.V1.LightingMult)
				sv2 := render.NewScreenVertex(p2, tri.V2.U, tri.V2.V, tri.V2.BlockID, tri.V2.FaceIndex, tri.V2.LightingMult)
				depth := (tri.V0.Z + tri.V1.Z + tri.V2.Z) / 3
				translucent = append(translucent, translucentTriangle{
					v0: sv0, v1: sv1, v2: sv2, depth: depth,
					blockID: v0.BlockID, faceIndex: v0.FaceIndex,
				})
			}
		}
	}
	if len(translucent) > 1 {
		sort.SliceStable(translucent, func(i, j int) bool {
			a, b := translucent[i], translucent[j]
			if a.depth != b.depth {
				return a.depth > b.depth
			}
			if a.blockID != b.blockID {
				return a.blockID < b.blockID
			}
			if a.faceIndex != b.faceIndex {
				return a.faceIndex < b.faceIndex
			}
			if a.v0.Y != b.v0.Y {
				return a.v0.Y < b.v0.Y
			}
			if a.v0.X != b.v0.X {
				return a.v0.X < b.v0.X
			}
			return a.v0.Depth < b.v0.Depth
		})
	}
	for _, tri := range translucent {
		c.framebuf.DrawTranslucentVoxelTrianglePerspective(tri.v0, tri.v1, tri.v2)
	}

	// 4. Draw player model in third person or held item in first person
	if c.perspectiveMode != 0 {
		c.drawPlayerModel()
	} else {
		c.drawHeldItem()
	}

	// 4.5 Draw hurt vignette if recently damaged
	if c.hurtFlashTimer > 0 {
		c.framebuf.DrawHurtVignette(float32(c.hurtFlashTimer * 2.5))
	}

	// 4.6 Draw underwater aquatic blue overlay only when camera is truly submerged under water
	if c.engine != nil {
		bx := int32(math.Floor(c.camera.X))
		by := int32(math.Floor(c.camera.Y))
		bz := int32(math.Floor(c.camera.Z))
		camEyeBlock := c.engine.GetBlockIfLoaded(bx, by, bz)
		aboveEyeBlock := c.engine.GetBlockIfLoaded(bx, by+1, bz)
		// Truly submerged: block at eye is water AND block above eye is water, or eye is deeply submerged (not skimming air surface)
		if (camEyeBlock == 8 || camEyeBlock == 9) && (aboveEyeBlock == 8 || aboveEyeBlock == 9 || c.camera.Y-float64(by) < 0.75) {
			c.framebuf.DrawUnderwaterOverlay()
		}
	}

	// 4.7 Death Screen
	if c.player != nil && c.player.Health <= 0 {
		mx, my := -1, -1
		if c.wndRef != nil {
			mx, my = c.wndRef.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
		}
		score := 0
		if c.player != nil {
			score = c.player.Level*100 + int(c.player.ExpProgress*100)
		}
		c.respawnBtnRect, c.deathTitleBtnRect = c.framebuf.DrawDeathScreen(score, mx, my)
		return
	}

	// 5. Draw 2D HUD (Centered crosshair, conditional 20 HP hearts, 20 hunger drumsticks, experience bar, level, hotbar)
	gameMode := 0
	health := float32(20.0)
	hunger := 20
	expProgress := float32(0.0)
	level := 0
	selectedSlot := 0
	var hotbarSlots [9]inventory.ItemStack
	if c.player != nil && c.player.Inventory != nil {
		gameMode = c.player.GameMode
		health = c.player.Health
		hunger = c.player.Hunger
		expProgress = c.player.ExpProgress
		level = c.player.Level
		selectedSlot = c.player.Inventory.SelectedHotbar
		for i := 0; i < 9; i++ {
			hotbarSlots[i] = c.player.Inventory.Slots[i]
		}
	}
	c.framebuf.DrawFullHUD(gameMode, health, hunger, expProgress, level, selectedSlot, hotbarSlots)

	// 5.5 Render Ender Dragon Boss Bar if in The End
	if c.currentDim == gen.DimensionEnd && c.engine != nil {
		dragon := c.engine.GetEnderDragon()
		if dragon != nil && (dragon.IsAlive || dragon.IsDying) {
			c.framebuf.DrawBossBar("Ender Dragon", dragon.Health, dragon.MaxHealth)
		}
	}

	// 5.6 Render Nether Portal Swirling Effect
	if c.portalStandTime > 0 {
		c.framebuf.DrawNetherPortalOverlay(float32(c.portalStandTime / 1.5))
	}

	// 6. Draw 2D Inventory, Crafting Table, or Furnace GUI if toggled
	if c.showInventoryGUI && c.player != nil && c.player.Inventory != nil {
		mx, my := -1, -1
		if c.wndRef != nil {
			mx, my = c.wndRef.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
		}
		if c.isCreative && !c.isCraftingTableOpen {
			c.framebuf.DrawCreativeInventoryGUI(c.player.Inventory, c.cursorItem, c.creativeTab, c.creativeSearch, c.creativeScrollRow, mx, my, c.creativeHotbars)
		} else {
			c.framebuf.DrawCraftingGUI(c.player.Inventory, c.craftGrid[:], c.craftResult, c.isCraftingTableOpen, c.cursorItem, mx, my)
		}
	} else if c.isFurnaceOpen && c.player != nil && c.player.Inventory != nil && c.engine != nil {
		mx, my := -1, -1
		if c.wndRef != nil {
			mx, my = c.wndRef.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
		}
		furnace := c.engine.GetOrCreateFurnace(c.currentFurnacePos)
		c.framebuf.DrawFurnaceGUI(furnace, c.player.Inventory, c.cursorItem, mx, my)
	}

	// 7. Draw F3 Debug Screen overlay if toggled (F3 key)
	if c.showDebugF3 && c.player != nil {
		c.fpsCounter++
		if time.Since(c.fpsTimer) >= 1*time.Second {
			c.currentFPS = c.fpsCounter
			c.fpsCounter = 0
			c.fpsTimer = time.Now()
		}

		yawNorm := math.Mod(c.camera.Yaw, 360.0)
		if yawNorm < 0 {
			yawNorm += 360.0
		}
		facing := "south (towards +Z)"
		if yawNorm >= 45.0 && yawNorm < 135.0 {
			facing = "west (towards -X)"
		} else if yawNorm >= 135.0 && yawNorm < 225.0 {
			facing = "north (towards -Z)"
		} else if yawNorm >= 225.0 && yawNorm < 315.0 {
			facing = "east (towards +X)"
		}

		bx := int32(math.Floor(c.player.Position.X))
		by := int32(math.Floor(c.player.Position.Y))
		bz := int32(math.Floor(c.player.Position.Z))
		cx := int32(math.Floor(float64(bx) / 16.0))
		cz := int32(math.Floor(float64(bz) / 16.0))

		targetName := "minecraft:air"
		if c.hasTarget && c.engine != nil {
			tID := c.engine.GetBlockIfLoaded(c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z)
			switch tID {
			case 1:
				targetName = "minecraft:stone"
			case 2:
				targetName = "minecraft:grass_block"
			case 3:
				targetName = "minecraft:dirt"
			case 4:
				targetName = "minecraft:cobblestone"
			case 5:
				targetName = "minecraft:oak_planks"
			case 7:
				targetName = "minecraft:bedrock"
			case 12:
				targetName = "minecraft:sand"
			case 13:
				targetName = "minecraft:gravel"
			case 14:
				targetName = "minecraft:gold_ore"
			case 15:
				targetName = "minecraft:iron_ore"
			case 16:
				targetName = "minecraft:coal_ore"
			case 17:
				targetName = "minecraft:oak_log"
			case 18:
				targetName = "minecraft:oak_leaves"
			case 56:
				targetName = "minecraft:diamond_ore"
			case 58:
				targetName = "minecraft:crafting_table"
			case 61:
				targetName = "minecraft:furnace"
			default:
				targetName = "minecraft:block"
			}
		}

		fpsDisp := c.currentFPS
		if fpsDisp <= 0 {
			fpsDisp = 60
		}
		ftMs := 1000.0 / float64(fpsDisp)

		dInfo := render.DebugInfo{
			FPS:           fpsDisp,
			FrameTimeMs:   ftMs,
			X:             c.player.Position.X,
			Y:             c.player.Position.Y,
			Z:             c.player.Position.Z,
			BlockX:        bx,
			BlockY:        by,
			BlockZ:        bz,
			ChunkX:        cx,
			ChunkZ:        cz,
			Facing:        facing,
			Yaw:           c.camera.Yaw,
			Pitch:         c.camera.Pitch,
			TimeOfDay:     worldTime,
			TargetedBlock: targetName,
			HasTarget:     c.hasTarget,
			TargetPos:     [3]int32{c.targetedPos.X, c.targetedPos.Y, c.targetedPos.Z},
		}
		c.framebuf.DrawDebugScreen(dInfo)
	}

	// 8. Draw in-game chat log & command input bar
	c.framebuf.DrawChatOverlay(c.chatMessages, c.isChatOpen, c.chatInput)
}

func (c *Client) processLocalWorldEvents() {
	if !c.isLocal || c.engine == nil || c.player == nil {
		return
	}
	for {
		select {
		case event := <-c.engine.Events:
			if event.Type == "player_teleport" && event.PlayerID == c.player.ID {
				c.camera.X = event.Position.X
				c.camera.Y = event.Position.Y + 1.62
				c.camera.Z = event.Position.Z
				c.publishPlayerSnapshot()
			}
			if event.Type == "item_pickup" && event.PlayerID == c.player.ID {
				switch {
				case event.BlockID == inventory.ItemDiamond:
					c.UnlockAdvancement("minecraft:story/mine_diamond", "钻石！")
				case event.BlockID == inventory.ItemIronIngot:
					c.UnlockAdvancement("minecraft:story/smelt_iron", "来硬的")
				case event.BlockID >= inventory.ItemLeatherHelmet && event.BlockID <= inventory.ItemGoldenBoots:
					c.UnlockAdvancement("minecraft:story/obtain_armor", "穿上盔甲")
				}
			}
		default:
			return
		}
	}
}

func (c *Client) drawMob(m *sim.MobEntity) {
	pos := m.Position
	dist := math.Hypot(math.Hypot(c.camera.X-pos.X, c.camera.Y-pos.Y), c.camera.Z-pos.Z)
	if dist > 64.0 {
		return
	}

	yaw := float64(m.Yaw)
	pitch := float64(m.Pitch)
	hurtFlash := m.IsHurt()

	// Leg-swing animation: pitch angle in degrees alternating dynamically
	walkSwing := 0.0
	if m.IsMoving || math.Hypot(m.Velocity.X, m.Velocity.Z) > 0.005 {
		walkSwing = math.Sin(float64(m.WalkDistance)*2.6) * 32.0
	}

	radY := yaw * math.Pi / 180.0
	cosY, sinY := math.Cos(radY), math.Sin(radY)

	var mobTex *render.RawImage
	switch m.Type {
	case sim.MobZombie:
		mobTex, _ = render.GetOfficialImage("entity/zombie/zombie.png")
	case sim.MobSkeleton:
		mobTex, _ = render.GetOfficialImage("entity/skeleton/skeleton.png")
	case sim.MobCreeper:
		mobTex, _ = render.GetOfficialImage("entity/creeper/creeper.png")
	case sim.MobCow:
		mobTex, _ = render.GetOfficialImage("entity/cow/cow.png")
	case sim.MobPig:
		mobTex, _ = render.GetOfficialImage("entity/pig/pig.png")
	case sim.MobSheep:
		mobTex, _ = render.GetOfficialImage("entity/sheep/sheep.png")
	case sim.MobChicken:
		mobTex, _ = render.GetOfficialImage("entity/chicken/chicken.png")
	case sim.MobEnderman:
		mobTex, _ = render.GetOfficialImage("entity/enderman/enderman.png")
	case sim.MobSpider:
		mobTex, _ = render.GetOfficialImage("entity/spider/spider.png")
	}

	tint := uint32(0)
	if hurtFlash {
		tint = 0xFFFF2222
	}

	switch m.Type {
	case sim.MobZombie:
		headUV := makeBoxFaceUVs(64, 64, [4]float32{8, 0, 16, 8}, [4]float32{16, 0, 24, 8}, [4]float32{24, 8, 32, 16}, [4]float32{8, 8, 16, 16}, [4]float32{16, 8, 24, 16}, [4]float32{0, 8, 8, 16})
		fbHead := [6]uint32{0xFF325424, 0xFF325424, 0xFF325424, 0xFF4C7838, 0xFF4C7838, 0xFF4C7838}
		c.renderTextured3DBox(pos.X, pos.Y+1.45, pos.Z, 0.22, 0.22, 0.22, yaw, pitch, mobTex, headUV, fbHead, tint)

		torsoUV := makeBoxFaceUVs(64, 64, [4]float32{20, 16, 28, 20}, [4]float32{28, 16, 36, 20}, [4]float32{32, 20, 40, 32}, [4]float32{20, 20, 28, 32}, [4]float32{16, 20, 20, 32}, [4]float32{28, 20, 32, 32})
		fbTorso := [6]uint32{0xFF2E8690, 0xFF2E8690, 0xFF2E8690, 0xFF2E8690, 0xFF2E8690, 0xFF2E8690}
		c.renderTextured3DBox(pos.X, pos.Y+0.88, pos.Z, 0.24, 0.35, 0.12, yaw, 0, mobTex, torsoUV, fbTorso, tint)

		armOffX := 0.35 * cosY
		armOffZ := -0.35 * sinY
		forwardX := 0.25 * sinY
		forwardZ := 0.25 * cosY
		armBob := math.Sin(float64(m.WalkDistance)*1.3) * 5.0
		armUV := makeUniformFaceUV(64, 64, 44, 20, 48, 32)
		fbArm := [6]uint32{0xFF2E8690, 0xFF4C7838, 0xFF2E8690, 0xFF4C7838, 0xFF2E8690, 0xFF2E8690}
		c.renderTextured3DBox(pos.X-armOffX+forwardX, pos.Y+0.88, pos.Z-armOffZ+forwardZ, 0.11, 0.12, 0.35, yaw, armBob, mobTex, armUV, fbArm, tint)
		c.renderTextured3DBox(pos.X+armOffX+forwardX, pos.Y+0.88, pos.Z+armOffZ+forwardZ, 0.11, 0.12, 0.35, yaw, -armBob, mobTex, armUV, fbArm, tint)

		legOffX := 0.12 * cosY
		legOffZ := -0.12 * sinY
		legUV := makeUniformFaceUV(64, 64, 4, 20, 8, 32)
		fbLeg := [6]uint32{0xFF2A3663, 0xFF2A3663, 0xFF2A3663, 0xFF2A3663, 0xFF2A3663, 0xFF2A3663}
		c.renderTextured3DBox(pos.X-legOffX, pos.Y+0.36, pos.Z-legOffZ, 0.11, 0.35, 0.12, yaw, -walkSwing, mobTex, legUV, fbLeg, tint)
		c.renderTextured3DBox(pos.X+legOffX, pos.Y+0.36, pos.Z+legOffZ, 0.11, 0.35, 0.12, yaw, walkSwing, mobTex, legUV, fbLeg, tint)

	case sim.MobSkeleton:
		headUV := makeBoxFaceUVs(64, 32, [4]float32{8, 0, 16, 8}, [4]float32{16, 0, 24, 8}, [4]float32{24, 8, 32, 16}, [4]float32{8, 8, 16, 16}, [4]float32{16, 8, 24, 16}, [4]float32{0, 8, 8, 16})
		fbBone := [6]uint32{0xFFE8E8E8, 0xFFE8E8E8, 0xFFE8E8E8, 0xFFE8E8E8, 0xFFE8E8E8, 0xFFE8E8E8}
		c.renderTextured3DBox(pos.X, pos.Y+1.45, pos.Z, 0.20, 0.20, 0.20, yaw, pitch, mobTex, headUV, fbBone, tint)

		torsoUV := makeBoxFaceUVs(64, 32, [4]float32{20, 16, 28, 20}, [4]float32{28, 16, 36, 20}, [4]float32{32, 20, 40, 32}, [4]float32{20, 20, 28, 32}, [4]float32{16, 20, 20, 32}, [4]float32{28, 20, 32, 32})
		c.renderTextured3DBox(pos.X, pos.Y+0.88, pos.Z, 0.20, 0.35, 0.10, yaw, 0, mobTex, torsoUV, fbBone, tint)

		armOffX := 0.30 * cosY
		armOffZ := -0.30 * sinY
		forwardX := 0.20 * sinY
		forwardZ := 0.20 * cosY
		limbUV := makeUniformFaceUV(64, 32, 0, 20, 4, 32)
		c.renderTextured3DBox(pos.X-armOffX+forwardX, pos.Y+0.88, pos.Z-armOffZ+forwardZ, 0.08, 0.08, 0.35, yaw, 0, mobTex, limbUV, fbBone, tint)
		c.renderTextured3DBox(pos.X+armOffX+forwardX, pos.Y+0.88, pos.Z+armOffZ+forwardZ, 0.08, 0.08, 0.35, yaw, 0, mobTex, limbUV, fbBone, tint)

		legOffX := 0.10 * cosY
		legOffZ := -0.10 * sinY
		c.renderTextured3DBox(pos.X-legOffX, pos.Y+0.36, pos.Z-legOffZ, 0.08, 0.35, 0.08, yaw, -walkSwing, mobTex, limbUV, fbBone, tint)
		c.renderTextured3DBox(pos.X+legOffX, pos.Y+0.36, pos.Z+legOffZ, 0.08, 0.35, 0.08, yaw, walkSwing, mobTex, limbUV, fbBone, tint)

	case sim.MobCreeper:
		creeperTint := tint
		if m.FuseTicks > 0 && (m.FuseTicks%4 < 2) {
			creeperTint = 0xFFFFFFFF
		}
		headUV := makeBoxFaceUVs(64, 32, [4]float32{8, 0, 16, 8}, [4]float32{16, 0, 24, 8}, [4]float32{24, 8, 32, 16}, [4]float32{8, 8, 16, 16}, [4]float32{16, 8, 24, 16}, [4]float32{0, 8, 8, 16})
		fbCreeperHead := [6]uint32{0xFF48B33B, 0xFF48B33B, 0xFF48B33B, 0xFF48B33B, 0xFF48B33B, 0xFF48B33B}
		c.renderTextured3DBox(pos.X, pos.Y+1.35, pos.Z, 0.22, 0.22, 0.22, yaw, pitch, mobTex, headUV, fbCreeperHead, creeperTint)

		bodyUV := makeUniformFaceUV(64, 32, 20, 20, 24, 32)
		fbCreeperBody := [6]uint32{0xFF3DA330, 0xFF3DA330, 0xFF3DA330, 0xFF3DA330, 0xFF3DA330, 0xFF3DA330}
		c.renderTextured3DBox(pos.X, pos.Y+0.75, pos.Z, 0.20, 0.38, 0.12, yaw, 0, mobTex, bodyUV, fbCreeperBody, creeperTint)

		footUV := makeUniformFaceUV(64, 32, 4, 20, 8, 26)
		fbFoot := [6]uint32{0xFF359428, 0xFF359428, 0xFF359428, 0xFF359428, 0xFF359428, 0xFF359428}
		for i, f := range [][2]float64{{-0.12, 0.12}, {0.12, 0.12}, {-0.12, -0.12}, {0.12, -0.12}} {
			fx := f[0]*cosY + f[1]*sinY
			fz := -f[0]*sinY + f[1]*cosY
			legPitch := walkSwing
			if i%2 == 1 {
				legPitch = -walkSwing
			}
			c.renderTextured3DBox(pos.X+fx, pos.Y+0.18, pos.Z+fz, 0.08, 0.18, 0.08, yaw, legPitch, mobTex, footUV, fbFoot, creeperTint)
		}

	case sim.MobPig:
		bodyUV := makeBoxFaceUVs(64, 64, [4]float32{36, 16, 46, 32}, [4]float32{54, 16, 64, 32}, [4]float32{36, 8, 46, 16}, [4]float32{46, 8, 56, 16}, [4]float32{28, 16, 36, 32}, [4]float32{46, 16, 54, 32})
		fbPigBody := [6]uint32{0xFFF0A0A0, 0xFFF0A0A0, 0xFFF0A0A0, 0xFFF0A0A0, 0xFFF0A0A0, 0xFFF0A0A0}
		c.renderTextured3DBox(pos.X, pos.Y+0.625, pos.Z, 0.3125, 0.25, 0.50, yaw, 0, mobTex, bodyUV, fbPigBody, tint)

		frontX := 0.60 * sinY
		frontZ := 0.60 * cosY
		headUV := makeBoxFaceUVs(64, 64, [4]float32{8, 0, 16, 8}, [4]float32{16, 0, 24, 8}, [4]float32{24, 8, 32, 16}, [4]float32{8, 8, 16, 16}, [4]float32{0, 8, 8, 16}, [4]float32{16, 8, 24, 16})
		fbPigHead := [6]uint32{0xFFF0A0A0, 0xFFF0A0A0, 0xFFF0A0A0, 0xFFF0A0A0, 0xFFF0A0A0, 0xFFF0A0A0}
		c.renderTextured3DBox(pos.X+frontX, pos.Y+0.625, pos.Z+frontZ, 0.25, 0.25, 0.25, yaw, pitch, mobTex, headUV, fbPigHead, tint)

		snoutX := 0.86 * sinY
		snoutZ := 0.86 * cosY
		snoutUV := makeBoxFaceUVs(64, 64, [4]float32{17, 16, 21, 17}, [4]float32{21, 16, 25, 17}, [4]float32{22, 17, 26, 20}, [4]float32{17, 17, 21, 20}, [4]float32{16, 17, 17, 20}, [4]float32{21, 17, 22, 20})
		fbSnout := [6]uint32{0xFFD06868, 0xFFE88A8A, 0xFFE88A8A, 0xFFD06868, 0xFFE88A8A, 0xFFE88A8A}
		c.renderTextured3DBox(pos.X+snoutX, pos.Y+0.54, pos.Z+snoutZ, 0.125, 0.09375, 0.03125, yaw, pitch, mobTex, snoutUV, fbSnout, tint)

		tailX := 0.50 * sinY
		tailZ := 0.50 * cosY
		c.render3DBox(pos.X-tailX, pos.Y+0.68, pos.Z-tailZ, 0.04, 0.06, 0.04, yaw, 30.0, 0xFFE88A8A, 0xFFE88A8A, 0xFFE88A8A)

		legs := [][2]float64{
			{-0.18, 0.30},
			{0.18, 0.30},
			{-0.18, -0.30},
			{0.18, -0.30},
		}
		legUV := makeBoxFaceUVs(64, 64, [4]float32{4, 16, 8, 20}, [4]float32{8, 16, 12, 20}, [4]float32{12, 20, 16, 26}, [4]float32{4, 20, 8, 26}, [4]float32{0, 20, 4, 26}, [4]float32{8, 20, 12, 26})
		fbPigLeg := [6]uint32{0xFFE08E8E, 0xFFE08E8E, 0xFFE08E8E, 0xFFE08E8E, 0xFFE08E8E, 0xFFE08E8E}
		for i, f := range legs {
			lx := f[0]*cosY + f[1]*sinY
			lz := -f[0]*sinY + f[1]*cosY
			legPitch := walkSwing
			if i == 1 || i == 2 {
				legPitch = -walkSwing
			}
			c.renderTextured3DBox(pos.X+lx, pos.Y+0.1875, pos.Z+lz, 0.125, 0.1875, 0.125, yaw, legPitch, mobTex, legUV, fbPigLeg, tint)
		}

	case sim.MobCow:
		bodyUV := makeBoxFaceUVs(64, 64, [4]float32{28, 14, 40, 32}, [4]float32{50, 14, 62, 32}, [4]float32{28, 4, 40, 14}, [4]float32{40, 4, 52, 14}, [4]float32{18, 14, 28, 32}, [4]float32{40, 14, 50, 32})
		fbCowBody := [6]uint32{0xFF5A4432, 0xFFE5E5E5, 0xFF5A4432, 0xFF5A4432, 0xFF5A4432, 0xFF5A4432}
		c.renderTextured3DBox(pos.X, pos.Y+1.0625, pos.Z, 0.375, 0.3125, 0.5625, yaw, 0, mobTex, bodyUV, fbCowBody, tint)

		udderZ := -0.28 * cosY
		udderX := 0.28 * sinY
		c.render3DBox(pos.X-udderX, pos.Y+0.78, pos.Z-udderZ, 0.12, 0.08, 0.12, yaw, 0, 0xFFECA0A0, 0xFFECA0A0, 0xFFECA0A0)

		frontX := 0.74 * sinY
		frontZ := 0.74 * cosY
		headUV := makeBoxFaceUVs(64, 64, [4]float32{6, 0, 14, 6}, [4]float32{14, 0, 22, 6}, [4]float32{20, 6, 28, 14}, [4]float32{6, 6, 14, 14}, [4]float32{0, 6, 6, 14}, [4]float32{14, 6, 20, 14})
		fbCowHead := [6]uint32{0xFF5A4432, 0xFF5A4432, 0xFF5A4432, 0xFF5A4432, 0xFF5A4432, 0xFF5A4432}
		c.renderTextured3DBox(pos.X+frontX, pos.Y+1.18, pos.Z+frontZ, 0.25, 0.25, 0.1875, yaw, pitch, mobTex, headUV, fbCowHead, tint)

		muzzleX := 0.98 * sinY
		muzzleZ := 0.98 * cosY
		muzzleUV := makeBoxFaceUVs(64, 64, [4]float32{1, 0, 5, 2}, [4]float32{5, 0, 9, 2}, [4]float32{1, 2, 5, 5}, [4]float32{1, 2, 5, 5}, [4]float32{0, 2, 1, 5}, [4]float32{5, 2, 6, 5})
		fbMuzzle := [6]uint32{0xFF7D7D7D, 0xFF7D7D7D, 0xFF7D7D7D, 0xFF7D7D7D, 0xFF7D7D7D, 0xFF7D7D7D}
		c.renderTextured3DBox(pos.X+muzzleX, pos.Y+1.05, pos.Z+muzzleZ, 0.16, 0.10, 0.06, yaw, pitch, mobTex, muzzleUV, fbMuzzle, tint)

		c.render3DBox(pos.X+frontX-0.22*cosY, pos.Y+1.43, pos.Z+frontZ+0.22*sinY, 0.03, 0.08, 0.03, yaw, 0, 0xFFCCCCCC, 0xFFCCCCCC, 0xFFCCCCCC)
		c.render3DBox(pos.X+frontX+0.22*cosY, pos.Y+1.43, pos.Z+frontZ-0.22*sinY, 0.03, 0.08, 0.03, yaw, 0, 0xFFCCCCCC, 0xFFCCCCCC, 0xFFCCCCCC)

		legs := [][2]float64{
			{-0.24, 0.34},
			{0.24, 0.34},
			{-0.24, -0.36},
			{0.24, -0.36},
		}
		legUV := makeBoxFaceUVs(64, 64, [4]float32{4, 16, 8, 20}, [4]float32{8, 16, 12, 20}, [4]float32{12, 20, 16, 32}, [4]float32{4, 20, 8, 32}, [4]float32{0, 20, 4, 32}, [4]float32{8, 20, 12, 32})
		fbCowLeg := [6]uint32{0xFF423225, 0xFF221810, 0xFF423225, 0xFF423225, 0xFF423225, 0xFF423225}
		for i, f := range legs {
			lx := f[0]*cosY + f[1]*sinY
			lz := -f[0]*sinY + f[1]*cosY
			legPitch := walkSwing
			if i == 1 || i == 2 {
				legPitch = -walkSwing
			}
			c.renderTextured3DBox(pos.X+lx, pos.Y+0.375, pos.Z+lz, 0.125, 0.375, 0.125, yaw, legPitch, mobTex, legUV, fbCowLeg, tint)
		}

	case sim.MobSheep:
		woolUV := makeBoxFaceUVs(64, 32, [4]float32{28, 16, 40, 28}, [4]float32{40, 16, 52, 28}, [4]float32{40, 28, 52, 32}, [4]float32{28, 28, 40, 32}, [4]float32{22, 28, 28, 32}, [4]float32{52, 28, 58, 32})
		fbSheepBody := [6]uint32{0xFFEFEFEF, 0xFFD6D6D6, 0xFFEFEFEF, 0xFFEFEFEF, 0xFFEFEFEF, 0xFFEFEFEF}
		c.renderTextured3DBox(pos.X, pos.Y+0.65, pos.Z, 0.30, 0.26, 0.42, yaw, 0, mobTex, woolUV, fbSheepBody, tint)

		frontX := 0.58 * sinY
		frontZ := 0.58 * cosY
		headUV := makeBoxFaceUVs(64, 32, [4]float32{6, 0, 12, 6}, [4]float32{12, 0, 18, 6}, [4]float32{18, 6, 24, 12}, [4]float32{6, 6, 12, 12}, [4]float32{0, 6, 6, 12}, [4]float32{12, 6, 18, 12})
		fbSheepHead := [6]uint32{0xFFD8C0A8, 0xFFD8C0A8, 0xFFD8C0A8, 0xFFD8C0A8, 0xFFD8C0A8, 0xFFD8C0A8}
		c.renderTextured3DBox(pos.X+frontX, pos.Y+0.90, pos.Z+frontZ, 0.16, 0.16, 0.16, yaw, pitch, mobTex, headUV, fbSheepHead, tint)
		c.render3DBox(pos.X+frontX, pos.Y+1.02, pos.Z+frontZ, 0.17, 0.06, 0.15, yaw, pitch, 0xFFEFEFEF, 0xFFEFEFEF, 0xFFEFEFEF)

		legs := [][2]float64{
			{-0.15, 0.24},
			{0.15, 0.24},
			{-0.15, -0.24},
			{0.15, -0.24},
		}
		legUV := makeUniformFaceUV(64, 32, 0, 16, 4, 26)
		fbSheepLeg := [6]uint32{0xFFBBA088, 0xFF4A3828, 0xFFBBA088, 0xFFBBA088, 0xFFBBA088, 0xFFBBA088}
		for i, f := range legs {
			lx := f[0]*cosY + f[1]*sinY
			lz := -f[0]*sinY + f[1]*cosY
			legPitch := walkSwing
			if i == 1 || i == 2 {
				legPitch = -walkSwing
			}
			c.renderTextured3DBox(pos.X+lx, pos.Y+0.20, pos.Z+lz, 0.07, 0.20, 0.07, yaw, legPitch, mobTex, legUV, fbSheepLeg, tint)
		}

	case sim.MobChicken:
		bodyUV := makeBoxFaceUVs(64, 32, [4]float32{16, 9, 22, 17}, [4]float32{22, 9, 28, 17}, [4]float32{22, 17, 28, 22}, [4]float32{16, 17, 22, 22}, [4]float32{10, 17, 16, 22}, [4]float32{28, 17, 34, 22})
		fbChickenBody := [6]uint32{0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF}
		c.renderTextured3DBox(pos.X, pos.Y+0.28, pos.Z, 0.14, 0.12, 0.18, yaw, 0, mobTex, bodyUV, fbChickenBody, tint)

		wingFlap := 0.0
		if walkSwing != 0 || m.Velocity.Y != 0 {
			wingFlap = math.Sin(float64(m.WalkDistance)*4.5) * 22.0
		}
		wingUV := makeUniformFaceUV(64, 32, 24, 13, 26, 17)
		fbWing := [6]uint32{0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF}
		c.renderTextured3DBox(pos.X-0.16*cosY, pos.Y+0.30, pos.Z+0.16*sinY, 0.03, 0.10, 0.14, yaw, wingFlap, mobTex, wingUV, fbWing, tint)
		c.renderTextured3DBox(pos.X+0.16*cosY, pos.Y+0.30, pos.Z-0.16*sinY, 0.03, 0.10, 0.14, yaw, -wingFlap, mobTex, wingUV, fbWing, tint)

		frontX := 0.18 * sinY
		frontZ := 0.18 * cosY
		headUV := makeBoxFaceUVs(64, 32, [4]float32{3, 0, 6, 3}, [4]float32{6, 0, 9, 3}, [4]float32{9, 3, 12, 9}, [4]float32{3, 3, 6, 9}, [4]float32{0, 3, 3, 9}, [4]float32{6, 3, 9, 9})
		fbChickenHead := [6]uint32{0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF}
		c.renderTextured3DBox(pos.X+frontX, pos.Y+0.44, pos.Z+frontZ, 0.08, 0.14, 0.08, yaw, pitch, mobTex, headUV, fbChickenHead, tint)

		c.render3DBox(pos.X+frontX, pos.Y+0.56, pos.Z+frontZ, 0.03, 0.05, 0.06, yaw, pitch, 0xFFFF2020, 0xFFFF2020, 0xFFFF2020)

		beakX := 0.26 * sinY
		beakZ := 0.26 * cosY
		beakUV := makeBoxFaceUVs(64, 32, [4]float32{15, 0, 19, 2}, [4]float32{19, 0, 23, 2}, [4]float32{15, 2, 19, 4}, [4]float32{15, 2, 19, 4}, [4]float32{14, 2, 15, 4}, [4]float32{19, 2, 20, 4})
		fbBeak := [6]uint32{0xFFFFB010, 0xFFFFB010, 0xFFFFB010, 0xFFFFB010, 0xFFFFB010, 0xFFFFB010}
		c.renderTextured3DBox(pos.X+beakX, pos.Y+0.44, pos.Z+beakZ, 0.06, 0.04, 0.04, yaw, pitch, mobTex, beakUV, fbBeak, tint)
		c.render3DBox(pos.X+beakX, pos.Y+0.38, pos.Z+beakZ, 0.03, 0.04, 0.03, yaw, pitch, 0xFFFF2020, 0xFFFF2020, 0xFFFF2020)

		legOffX := 0.06 * cosY
		legOffZ := -0.06 * sinY
		legUV := makeUniformFaceUV(64, 32, 26, 0, 28, 4)
		fbLeg := [6]uint32{0xFFFFB010, 0xFFFFB010, 0xFFFFB010, 0xFFFFB010, 0xFFFFB010, 0xFFFFB010}
		c.renderTextured3DBox(pos.X-legOffX, pos.Y+0.08, pos.Z-legOffZ, 0.02, 0.08, 0.03, yaw, -walkSwing*1.4, mobTex, legUV, fbLeg, tint)
		c.renderTextured3DBox(pos.X+legOffX, pos.Y+0.08, pos.Z+legOffZ, 0.02, 0.08, 0.03, yaw, walkSwing*1.4, mobTex, legUV, fbLeg, tint)

	case sim.MobEnderman:
		headUV := makeBoxFaceUVs(64, 32, [4]float32{8, 0, 16, 8}, [4]float32{16, 0, 24, 8}, [4]float32{24, 8, 32, 16}, [4]float32{8, 8, 16, 16}, [4]float32{0, 8, 8, 16}, [4]float32{16, 8, 24, 16})
		fbEnder := [6]uint32{0xFF161616, 0xFF101010, 0xFF141414, 0xFF141414, 0xFF141414, 0xFF141414}
		c.renderTextured3DBox(pos.X, pos.Y+2.50, pos.Z, 0.16, 0.16, 0.16, yaw, pitch, mobTex, headUV, fbEnder, tint)

		// Glowing purple eyes
		eyeOffX := 0.06 * cosY
		eyeOffZ := -0.06 * sinY
		eyeFwdX := 0.17 * sinY
		eyeFwdZ := 0.17 * cosY
		c.render3DBox(pos.X-eyeOffX+eyeFwdX, pos.Y+2.50, pos.Z-eyeOffZ+eyeFwdZ, 0.03, 0.02, 0.02, yaw, pitch, 0xFFCC22DD, 0xFFCC22DD, 0xFFCC22DD)
		c.render3DBox(pos.X+eyeOffX+eyeFwdX, pos.Y+2.50, pos.Z+eyeOffZ+eyeFwdZ, 0.03, 0.02, 0.02, yaw, pitch, 0xFFCC22DD, 0xFFCC22DD, 0xFFCC22DD)

		torsoUV := makeUniformFaceUV(64, 32, 20, 16, 28, 28)
		c.renderTextured3DBox(pos.X, pos.Y+1.85, pos.Z, 0.16, 0.45, 0.10, yaw, 0, mobTex, torsoUV, fbEnder, tint)

		armOffX := 0.22 * cosY
		armOffZ := -0.22 * sinY
		armBob := math.Sin(float64(m.WalkDistance)*1.2) * 15.0
		limbUV := makeUniformFaceUV(64, 32, 56, 0, 60, 30)
		c.renderTextured3DBox(pos.X-armOffX, pos.Y+1.60, pos.Z-armOffZ, 0.05, 0.65, 0.05, yaw, -armBob, mobTex, limbUV, fbEnder, tint)
		c.renderTextured3DBox(pos.X+armOffX, pos.Y+1.60, pos.Z+armOffZ, 0.05, 0.65, 0.05, yaw, armBob, mobTex, limbUV, fbEnder, tint)

		legOffX := 0.08 * cosY
		legOffZ := -0.08 * sinY
		c.renderTextured3DBox(pos.X-legOffX, pos.Y+0.75, pos.Z-legOffZ, 0.05, 0.75, 0.05, yaw, -walkSwing*0.8, mobTex, limbUV, fbEnder, tint)
		c.renderTextured3DBox(pos.X+legOffX, pos.Y+0.75, pos.Z+legOffZ, 0.05, 0.75, 0.05, yaw, walkSwing*0.8, mobTex, limbUV, fbEnder, tint)

	case sim.MobSpider:
		fbSpider := [6]uint32{0xFF242020, 0xFF1C1818, 0xFF221E1E, 0xFF221E1E, 0xFF221E1E, 0xFF221E1E}
		// Thorax (central body)
		bodyUV := makeUniformFaceUV(64, 32, 0, 0, 16, 12)
		c.renderTextured3DBox(pos.X, pos.Y+0.35, pos.Z, 0.18, 0.16, 0.18, yaw, 0, mobTex, bodyUV, fbSpider, tint)

		// Head (front)
		headX := 0.32 * sinY
		headZ := 0.32 * cosY
		headUV := makeUniformFaceUV(64, 32, 32, 4, 48, 16)
		c.renderTextured3DBox(pos.X+headX, pos.Y+0.35, pos.Z+headZ, 0.16, 0.14, 0.16, yaw, pitch, mobTex, headUV, fbSpider, tint)

		// Red Spider Eyes
		eyeOffX := 0.06 * cosY
		eyeOffZ := -0.06 * sinY
		eyeFwdX := 0.47 * sinY
		eyeFwdZ := 0.47 * cosY
		c.render3DBox(pos.X-eyeOffX+eyeFwdX, pos.Y+0.37, pos.Z-eyeOffZ+eyeFwdZ, 0.03, 0.02, 0.02, yaw, pitch, 0xFFFF1515, 0xFFFF1515, 0xFFFF1515)
		c.render3DBox(pos.X+eyeOffX+eyeFwdX, pos.Y+0.37, pos.Z+eyeOffZ+eyeFwdZ, 0.03, 0.02, 0.02, yaw, pitch, 0xFFFF1515, 0xFFFF1515, 0xFFFF1515)

		// Abdomen (rear)
		abX := -0.36 * sinY
		abZ := -0.36 * cosY
		abUV := makeUniformFaceUV(64, 32, 0, 12, 24, 32)
		c.renderTextured3DBox(pos.X+abX, pos.Y+0.42, pos.Z+abZ, 0.28, 0.22, 0.34, yaw, 0, mobTex, abUV, fbSpider, tint)

		// 8 Angled Legs (4 per side)
		legUV := makeUniformFaceUV(64, 32, 18, 0, 24, 16)
		legSpacings := []float64{-0.18, -0.06, 0.06, 0.18}
		for i, off := range legSpacings {
			sideYaw := 55.0 - float64(i)*22.0
			legSwing := math.Sin(float64(m.WalkDistance)*2.5+float64(i)*0.8) * 16.0

			// Left leg
			lx := -0.28*cosY + off*sinY
			lz := 0.28*sinY + off*cosY
			c.renderTextured3DBox(pos.X+lx, pos.Y+0.28, pos.Z+lz, 0.20, 0.04, 0.04, yaw+sideYaw, legSwing, mobTex, legUV, fbSpider, tint)

			// Right leg
			rx := 0.28*cosY + off*sinY
			rz := -0.28*sinY + off*cosY
			c.renderTextured3DBox(pos.X+rx, pos.Y+0.28, pos.Z+rz, 0.20, 0.04, 0.04, yaw-sideYaw, -legSwing, mobTex, legUV, fbSpider, tint)
		}
	}

	// Health bar above mob if damaged
	if m.Health < m.MaxHealth {
		headH := 1.9
		if m.Type == sim.MobEnderman {
			headH = 3.1
		} else if m.Type == sim.MobSpider {
			headH = 1.0
		} else if m.Type == sim.MobChicken {
			headH = 0.9
		}
		sp, ok := c.camera.ProjectToScreen(pos.X, pos.Y+headH, pos.Z, c.framebuf.Width, c.framebuf.Height)
		if ok && sp.Depth > 0.2 {
			barW := 36
			barH := 3
			barX := int(sp.X) - barW/2
			barY := int(sp.Y)
			c.framebuf.DrawRect(barX, barY, barW, barH, 0xFF000000)
			hpFrac := m.Health / m.MaxHealth
			if hpFrac < 0 {
				hpFrac = 0
			}
			fillW := int(float32(barW-2) * hpFrac)
			c.framebuf.DrawRect(barX+1, barY+1, fillW, barH-2, 0xFF00FF00)
		}
	}
}

func (c *Client) drawEndEntities() {
	if c.engine == nil {
		return
	}

	crystals := c.engine.GetEndCrystals()
	dragon := c.engine.GetEnderDragon()

	// 1. Draw End Crystals atop obsidian pillars
	crystalTex, _ := render.GetOfficialImage("entity/end_crystal/end_crystal.png")
	t := float64(time.Now().UnixNano()%10000000000) / 1e9
	spinYaw := t * 90.0

	for _, cry := range crystals {
		if !cry.IsAlive {
			continue
		}
		cPos := cry.Position
		// Base plate (bedrock/obsidian slab)
		c.render3DBox(cPos.X, cPos.Y-0.1, cPos.Z, 0.40, 0.08, 0.40, 0, 0, 0xFF2A2A2A, 0xFF181818, 0xFF222222)

		// Inner luminous white core
		c.render3DBox(cPos.X, cPos.Y+0.6, cPos.Z, 0.15, 0.15, 0.15, -spinYaw*1.5, 20, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFEEEEEE)

		// Outer rotating crystal cube
		cryUV := makeUniformFaceUV(64, 64, 0, 0, 32, 32)
		fbCry := [6]uint32{0xFFDA70D6, 0xFF8A2BE2, 0xFFBA55D3, 0xFFBA55D3, 0xFF9932CC, 0xFF9932CC}
		c.renderTextured3DBox(cPos.X, cPos.Y+0.6, cPos.Z, 0.28, 0.28, 0.28, spinYaw, 15, crystalTex, cryUV, fbCry, 0)

		// Healing beam to dragon if tethered
		if dragon != nil && (dragon.IsAlive || dragon.IsDying) && dragon.HealingCrystalID == cry.ID {
			c.drawHealingBeam(cPos, dragon.Position)
		}
	}

	// 2. Draw Ender Dragon
	if dragon != nil && (dragon.IsAlive || dragon.IsDying) {
		c.drawEnderDragon(dragon)
	}
}

func (c *Client) drawHealingBeam(start, end sim.Vec3) {
	dx := end.X - start.X
	dy := end.Y - start.Y
	dz := end.Z - start.Z
	steps := 12
	for i := 0; i <= steps; i++ {
		frac := float64(i) / float64(steps)
		bx := start.X + dx*frac
		by := start.Y + dy*frac + 0.6
		bz := start.Z + dz*frac
		c.render3DBox(bx, by, bz, 0.08, 0.08, 0.08, 0, 0, 0xFFE050FF, 0xFFD030EE, 0xFFE050FF)
	}
}

func (c *Client) drawEnderDragon(dragon *sim.EnderDragon) {
	pos := dragon.Position
	yaw := float64(dragon.Yaw)
	pitch := float64(dragon.Pitch)

	radY := yaw * math.Pi / 180.0
	cosY, sinY := math.Cos(radY), math.Sin(radY)

	dragonTex, _ := render.GetOfficialImage("entity/enderdragon/dragon.png")
	tint := uint32(0)
	if dragon.IsDying {
		tint = 0xFFFFAAAA
	}

	// 1. Torso / Main Body
	torsoUV := makeUniformFaceUV(256, 256, 0, 0, 64, 64)
	fbDragon := [6]uint32{0xFF1A1A1A, 0xFF141414, 0xFF181818, 0xFF181818, 0xFF181818, 0xFF181818}
	c.renderTextured3DBox(pos.X, pos.Y, pos.Z, 0.75, 0.65, 1.25, yaw, pitch, dragonTex, torsoUV, fbDragon, tint)

	// Dorsal Spine plates on Torso
	c.render3DBox(pos.X, pos.Y+0.75, pos.Z, 0.12, 0.20, 1.10, yaw, pitch, 0xFF353535, 0xFF2A2A2A, 0xFF2F2F2F)

	// 2. Neck & Head extending forward (+Z in local coords -> +sinY, +cosY in world)
	for i := 1; i <= 4; i++ {
		dist := float64(i) * 0.75
		nx := pos.X + dist*sinY
		ny := pos.Y + float64(i)*0.25 - math.Sin(pitch*math.Pi/180.0)*dist*0.4
		nz := pos.Z + dist*cosY
		c.renderTextured3DBox(nx, ny, nz, 0.30, 0.30, 0.35, yaw, pitch*0.7, dragonTex, torsoUV, fbDragon, tint)
		c.render3DBox(nx, ny+0.35, nz, 0.08, 0.12, 0.25, yaw, pitch*0.7, 0xFF353535, 0xFF2A2A2A, 0xFF2F2F2F)
	}

	// Head at tip of neck
	headDist := 3.6
	hx := pos.X + headDist*sinY
	hy := pos.Y + 1.1 - math.Sin(pitch*math.Pi/180.0)*headDist*0.4
	hz := pos.Z + headDist*cosY
	headUV := makeUniformFaceUV(256, 256, 112, 30, 144, 62)
	c.renderTextured3DBox(hx, hy, hz, 0.45, 0.40, 0.55, yaw, pitch, dragonTex, headUV, fbDragon, tint)

	// Snout & Jaw
	snoutDist := headDist + 0.60
	c.render3DBox(pos.X+snoutDist*sinY, hy-0.10, pos.Z+snoutDist*cosY, 0.30, 0.20, 0.35, yaw, pitch, 0xFF141414, 0xFF141414, 0xFF141414)

	// Purple Glowing Eyes
	eyeOff := 0.32
	c.render3DBox(hx-eyeOff*cosY+0.25*sinY, hy+0.12, hz+eyeOff*sinY+0.25*cosY, 0.08, 0.08, 0.08, yaw, pitch, 0xFFCC22DD, 0xFFCC22DD, 0xFFCC22DD)
	c.render3DBox(hx+eyeOff*cosY+0.25*sinY, hy+0.12, hz-eyeOff*sinY+0.25*cosY, 0.08, 0.08, 0.08, yaw, pitch, 0xFFCC22DD, 0xFFCC22DD, 0xFFCC22DD)

	// 3. Tail extending backwards (-Z local)
	for i := 1; i <= 5; i++ {
		dist := float64(i) * 0.85
		tx := pos.X - dist*sinY
		ty := pos.Y - float64(i)*0.15
		tz := pos.Z - dist*cosY
		w := 0.40 - float64(i)*0.05
		c.renderTextured3DBox(tx, ty, tz, w, w, 0.42, yaw, 0, dragonTex, torsoUV, fbDragon, tint)
		c.render3DBox(tx, ty+w+0.08, tz, 0.06, 0.12, 0.30, yaw, 0, 0xFF353535, 0xFF2A2A2A, 0xFF2F2F2F)
	}

	// 4. Wings flapping dynamically
	wingFlap := math.Sin(float64(dragon.WingAngle)) * 36.0
	// Left wing
	lwOffX := -1.3 * cosY
	lwOffZ := 1.3 * sinY
	c.renderTextured3DBox(pos.X+lwOffX, pos.Y+0.3, pos.Z+lwOffZ, 1.10, 0.08, 0.70, yaw-15.0, wingFlap, dragonTex, torsoUV, fbDragon, tint)
	// Right wing
	rwOffX := 1.3 * cosY
	rwOffZ := -1.3 * sinY
	c.renderTextured3DBox(pos.X+rwOffX, pos.Y+0.3, pos.Z+rwOffZ, 1.10, 0.08, 0.70, yaw+15.0, -wingFlap, dragonTex, torsoUV, fbDragon, tint)
}

// BoxFaceUV stores the UV bounds for one face of a 3D box.
type BoxFaceUV struct {
	U0, V0, U1, V1 float32
}

func makeBoxFaceUVs(texW, texH float32, top, btm, north, south, west, east [4]float32) [6]BoxFaceUV {
	uv := func(r [4]float32) BoxFaceUV {
		return BoxFaceUV{
			U0: r[0] / texW,
			V0: r[1] / texH,
			U1: r[2] / texW,
			V1: r[3] / texH,
		}
	}
	return [6]BoxFaceUV{
		uv(top),   // 0: +Y (Top)
		uv(btm),   // 1: -Y (Bottom)
		uv(north), // 2: -Z (North/Back)
		uv(south), // 3: +Z (South/Front)
		uv(west),  // 4: -X (West/Right)
		uv(east),  // 5: +X (East/Left)
	}
}

func makeUniformFaceUV(texW, texH float32, x0, y0, x1, y1 float32) [6]BoxFaceUV {
	r := [4]float32{x0, y0, x1, y1}
	return makeBoxFaceUVs(texW, texH, r, r, r, r, r, r)
}

// renderTextured3DBox renders a volumetric 3D cuboid with authentic UV texture mapping, lighting and depth testing.
func (c *Client) renderTextured3DBox(
	cx, cy, cz, hx, hy, hz, yawDeg, pitchDeg float64,
	tex *render.RawImage,
	faceUVs [6]BoxFaceUV,
	fallbackCols [6]uint32,
	tint uint32,
) {
	radY := yawDeg * math.Pi / 180.0
	radP := pitchDeg * math.Pi / 180.0
	cosY, sinY := math.Cos(radY), math.Sin(radY)
	cosP, sinP := math.Cos(radP), math.Sin(radP)

	rot := func(lx, ly, lz float64) (float64, float64, float64) {
		ry := ly*cosP - lz*sinP
		rz := ly*sinP + lz*cosP
		rx := lx*cosY + rz*sinY
		rz2 := -lx*sinY + rz*cosY
		return cx + rx, cy + ry, cz + rz2
	}

	rotNormal := func(nx, ny, nz float64) (float64, float64, float64) {
		ry := ny*cosP - nz*sinP
		rz := ny*sinP + nz*cosP
		rx := nx*cosY + rz*sinY
		rz2 := -nx*sinY + rz*cosY
		return rx, ry, rz2
	}

	corners := [8][3]float64{
		{-hx, -hy, -hz}, // 0: -X, -Y, -Z
		{hx, -hy, -hz},  // 1: +X, -Y, -Z
		{hx, hy, -hz},   // 2: +X, +Y, -Z
		{-hx, hy, -hz},  // 3: -X, +Y, -Z
		{-hx, -hy, hz},  // 4: -X, -Y, +Z
		{hx, -hy, hz},   // 5: +X, -Y, +Z
		{hx, hy, hz},    // 6: +X, +Y, +Z
		{-hx, hy, hz},   // 7: -X, +Y, +Z
	}

	var screenPts [8]render.ScreenPoint
	var valid [8]bool
	for i := 0; i < 8; i++ {
		wx, wy, wz := rot(corners[i][0], corners[i][1], corners[i][2])
		screenPts[i], valid[i] = c.camera.ProjectToScreen(wx, wy, wz, c.framebuf.Width, c.framebuf.Height)
	}

	type boxFaceDef struct {
		indices [4]int
		normal  [3]float64
		center  [3]float64
	}

	// 6 Box faces with canonical Minecraft UV corner alignment:
	// p0: Top-Left (U0, V0), p1: Top-Right (U1, V0), p2: Bottom-Right (U1, V1), p3: Bottom-Left (U0, V1)
	faces := [6]boxFaceDef{
		{indices: [4]int{3, 2, 6, 7}, normal: [3]float64{0, 1, 0}, center: [3]float64{0, hy, 0}},   // 0: Top (+Y)
		{indices: [4]int{4, 5, 1, 0}, normal: [3]float64{0, -1, 0}, center: [3]float64{0, -hy, 0}}, // 1: Bottom (-Y)
		{indices: [4]int{2, 3, 0, 1}, normal: [3]float64{0, 0, -1}, center: [3]float64{0, 0, -hz}}, // 2: North (-Z, Back)
		{indices: [4]int{7, 6, 5, 4}, normal: [3]float64{0, 0, 1}, center: [3]float64{0, 0, hz}},   // 3: South (+Z, Front / Face)
		{indices: [4]int{3, 7, 4, 0}, normal: [3]float64{-1, 0, 0}, center: [3]float64{-hx, 0, 0}}, // 4: West (-X, Right)
		{indices: [4]int{6, 2, 1, 5}, normal: [3]float64{1, 0, 0}, center: [3]float64{hx, 0, 0}},   // 5: East (+X, Left)
	}

	for fIdx, f := range faces {
		if !valid[f.indices[0]] || !valid[f.indices[1]] || !valid[f.indices[2]] || !valid[f.indices[3]] {
			continue
		}

		// 3D back-face normal culling: test if face normal points towards camera
		rnx, rny, rnz := rotNormal(f.normal[0], f.normal[1], f.normal[2])
		fcx, fcy, fcz := rot(f.center[0], f.center[1], f.center[2])
		vx := fcx - c.camera.X
		vy := fcy - c.camera.Y
		vz := fcz - c.camera.Z
		if rnx*vx+rny*vy+rnz*vz >= 0 {
			continue // Points away from camera -> culled
		}

		p0 := screenPts[f.indices[0]]
		p1 := screenPts[f.indices[1]]
		p2 := screenPts[f.indices[2]]
		p3 := screenPts[f.indices[3]]

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
		if c.isCreative && c.showInventoryGUI && !c.isCraftingTableOpen && c.creativeTab == inventory.CreativeHotbar && vkCode >= 0x31 && vkCode <= 0x39 && c.player != nil && c.player.Inventory != nil {
			preset := vkCode - 0x31
			if wnd.IsKeyDown(0x11) { // Ctrl+1..9 saves the current hotbar preset.
				copy(c.creativeHotbars[preset][:], c.player.Inventory.Slots[:9])
				if err := saveCreativeHotbarPresets(c.creativeHotbars); err != nil {
					log.Printf("[Creative] Failed to save hotbar presets: %v", err)
				}
				return
			}
			if wnd.IsKeyDown(0x10) { // Shift+1..9 loads a saved preset.
				copy(c.player.Inventory.Slots[:9], c.creativeHotbars[preset][:])
				return
			}
		}
		if c.isCreative && c.showInventoryGUI && !c.isCraftingTableOpen &&
			vkCode >= 0x31 && vkCode <= 0x39 && c.player != nil && c.player.Inventory != nil && c.framebuf != nil {
			mx, my := wnd.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
			hit := render.HitTestCreativeInventory(c.framebuf.Width, c.framebuf.Height, c.creativeTab, c.creativeSearch, c.creativeScrollRow, mx, my, c.creativeHotbars)
			items := inventory.SearchCreativeCatalog(c.creativeTab, c.creativeSearch)
			if c.creativeTab == inventory.CreativeHotbar {
				if applyCreativeHotbarKey(c.player.Inventory, hit, items, vkCode-0x31, c.creativeHotbars) {
					return
				}
			} else if applyCreativeHotbarKey(c.player.Inventory, hit, items, vkCode-0x31) {
				return
			}
		}
		if c.showAdvancementsGUI && (vkCode == 0x1B || vkCode == 0x4C) {
			c.showAdvancementsGUI = false
			wnd.SetMouseCaptured(true)
			return
		}
		if c.isCreative && c.showInventoryGUI && c.creativeSearchActive {
			switch vkCode {
			case 0x08: // Backspace
				runes := []rune(c.creativeSearch)
				if len(runes) > 0 {
					c.creativeSearch = string(runes[:len(runes)-1])
					c.creativeScrollRow = 0
				}
				return
			case 0x0D: // Enter ends search editing
				c.creativeSearchActive = false
				return
			case 0x1B, 0x45: // Escape or E closes the inventory below
				c.creativeSearchActive = false
			default:
				return
			}
		}
		switch vkCode {
		case 0x46: // F: Swap main hand and offhand
			if c.screenState == ScreenPlaying && !c.showPauseMenu && !c.showInventoryGUI && !c.isFurnaceOpen && c.player != nil && c.player.Inventory != nil {
				c.player.Inventory.SwapSelectedWithOffhand()
			}
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
					c.creativeSearch = ""
					c.creativeSearchActive = false
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
		if c.showInventoryGUI && c.isCreative && !c.isCraftingTableOpen {
			items := inventory.SearchCreativeCatalog(c.creativeTab, c.creativeSearch)
			maxScroll := render.CreativeCatalogMaxScroll(len(items))
			c.creativeScrollRow, c.creativeWheelRemainder = scrollCreativeCatalog(c.creativeScrollRow, delta, maxScroll, c.creativeWheelRemainder)
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
	wnd.OnMouseMove = func() {
		if c.creativeDrag.Active {
			if !c.showInventoryGUI || !c.isCreative || c.isCraftingTableOpen || c.framebuf == nil {
				c.creativeDrag.reset()
			} else {
				mx, my := wnd.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
				hit := render.HitTestCreativeInventory(c.framebuf.Width, c.framebuf.Height, c.creativeTab, c.creativeSearch, c.creativeScrollRow, mx, my, c.creativeHotbars)
				if hit.Type == render.SlotInventory {
					c.creativeDrag.addTarget(hit.Index)
				}
			}
		}
		if !c.creativeScrollDrag {
			return
		}
		if !wnd.IsMouseDown(1) || !c.showInventoryGUI || !c.isCreative || c.isCraftingTableOpen || c.framebuf == nil {
			c.creativeScrollDrag = false
			return
		}
		items := inventory.SearchCreativeCatalog(c.creativeTab, c.creativeSearch)
		maxScroll := render.CreativeCatalogMaxScroll(len(items))
		_, my := wnd.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
		c.creativeScrollRow = render.CreativeScrollRowAtPointer(c.framebuf.Width, c.framebuf.Height, my, maxScroll, c.creativeScrollOffset)
	}
	wnd.OnMouseCancel = func() {
		c.creativeDrag.reset()
		c.creativeScrollDrag = false
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
		if c.isCreative && c.showInventoryGUI && !c.isCraftingTableOpen && c.creativeSearchActive {
			if ch >= 32 && ch != 127 && len([]rune(c.creativeSearch)) < 64 {
				c.creativeSearch += string(ch)
				c.creativeScrollRow = 0
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
		if btn == 1 {
			c.creativeScrollDrag = false
		}
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
			if c.isCreative && !c.isCraftingTableOpen {
				hit := render.HitTestCreativeInventory(c.framebuf.Width, c.framebuf.Height, c.creativeTab, c.creativeSearch, c.creativeScrollRow, mx, my, c.creativeHotbars)
				switch hit.Type {
				case render.SlotCreativeSearch:
					c.creativeSearchActive = true
				case render.SlotCreativeTab:
					previousTab := c.creativeTab
					c.creativeTab = inventory.CreativeCategory(hit.Index)
					if previousTab != c.creativeTab {
						c.creativeSearch = ""
					}
					c.creativeSearchActive = c.creativeTab == inventory.CreativeAll
					c.creativeScrollRow = 0
				case render.SlotCreativeScroll:
					c.creativeSearchActive = false
					items := inventory.SearchCreativeCatalog(c.creativeTab, c.creativeSearch)
					maxScroll := render.CreativeCatalogMaxScroll(len(items))
					if maxScroll > 0 {
						thumbY := render.CreativeScrollThumbY(c.framebuf.Width, c.framebuf.Height, c.creativeScrollRow, maxScroll)
						scale := render.CalculateGUIScaleFromDimensions(c.framebuf.Width, c.framebuf.Height)
						thumbHeight := 17 * scale
						if my >= thumbY && my < thumbY+thumbHeight {
							c.creativeScrollOffset = my - thumbY
						} else {
							c.creativeScrollOffset = thumbHeight / 2
						}
						c.creativeScrollRow = render.CreativeScrollRowAtPointer(c.framebuf.Width, c.framebuf.Height, my, maxScroll, c.creativeScrollOffset)
						c.creativeScrollDrag = true
					}
				case render.SlotCreativeItem:
					c.creativeSearchActive = false
					items := inventory.SearchCreativeCatalog(c.creativeTab, c.creativeSearch)
					if c.creativeTab == inventory.CreativeHotbar {
						if hit.Index >= 0 && hit.Index < 81 {
							stack := c.creativeHotbars[hit.Index/9][hit.Index%9]
							if !stack.IsEmpty() {
								c.cursorItem = inventory.CreativeItemStack(stack.ItemID, btn)
								if btn == 1 || btn == 2 {
									c.creativeDrag = creativeInventoryDrag{Active: true, Button: btn, StartIndex: -1}
								}
							}
						}
						break
					}
					if hit.Index >= 0 && hit.Index < len(items) {
						item := items[hit.Index]
						c.cursorItem = inventory.CreativeItemStack(item.ID, btn)
						if btn == 1 || btn == 2 {
							c.creativeDrag = creativeInventoryDrag{Active: true, Button: btn, StartIndex: -1}
						}
					}
				case render.SlotInventory:
					if hit.Index >= 0 && hit.Index < len(c.player.Inventory.Slots) {
						if (btn == 1 || btn == 2) && !c.cursorItem.IsEmpty() && !wnd.IsKeyDown(0x10) {
							c.creativeDrag = creativeInventoryDrag{Active: true, Button: btn, StartIndex: hit.Index, TargetSlots: []int{hit.Index}}
							return
						}
						slot := &c.player.Inventory.Slots[hit.Index]
						if btn == 1 && wnd.IsKeyDown(0x10) && !slot.IsEmpty() {
							dstStart, dstEnd := 0, 9
							if hit.Index < 9 {
								dstStart, dstEnd = 9, 36
							}
							for dst := dstStart; dst < dstEnd; dst++ {
								if c.player.Inventory.Slots[dst].IsEmpty() {
									c.player.Inventory.Slots[dst] = *slot
									*slot = inventory.ItemStack{}
									break
								}
							}
						} else if btn == 1 {
							handleSlotLeftClick(&c.cursorItem, slot)
						} else if btn == 2 {
							handleSlotRightClick(&c.cursorItem, slot)
						} else if btn == 3 && !slot.IsEmpty() {
							c.cursorItem = inventory.CreativeItemStack(slot.ItemID, 3)
						}
					}
				case render.SlotCreativeTrash:
					if btn == 1 {
						c.cursorItem = inventory.ItemStack{}
						if wnd.IsKeyDown(0x10) {
							for i := range c.player.Inventory.Slots {
								c.player.Inventory.Slots[i] = inventory.ItemStack{}
							}
							for i := range c.player.Inventory.Armor {
								c.player.Inventory.Armor[i] = inventory.ItemStack{}
							}
							c.player.Inventory.Offhand = inventory.ItemStack{}
							c.syncPlayerArmorProtection()
						}
					}
				case render.SlotCreativeArmor:
					if hit.Index >= 0 && hit.Index < len(c.player.Inventory.Armor) {
						slot := &c.player.Inventory.Armor[hit.Index]
						if btn == 1 && wnd.IsKeyDown(0x10) && !slot.IsEmpty() {
							for dst := range c.player.Inventory.Slots {
								if c.player.Inventory.Slots[dst].IsEmpty() {
									c.player.Inventory.Slots[dst] = *slot
									*slot = inventory.ItemStack{}
									break
								}
							}
						} else {
							inventory.HandleArmorSlotClick(&c.cursorItem, slot, hit.Index, btn)
						}
						c.syncPlayerArmorProtection()
					}
				case render.SlotNone:
					if btn == 1 || btn == 2 {
						c.cursorItem = inventory.ItemStack{}
					}
				}
				return
			}
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
	wnd.OnMouseRelease = func(btn int) {
		if btn == 1 {
			c.creativeScrollDrag = false
		}
		if !c.creativeDrag.Active || btn != c.creativeDrag.Button {
			return
		}
		drag := c.creativeDrag
		c.creativeDrag.reset()
		if c.player == nil || c.player.Inventory == nil || c.framebuf == nil || !c.showInventoryGUI || !c.isCreative || c.isCraftingTableOpen {
			return
		}
		if drag.StartIndex < 0 && len(drag.TargetSlots) == 0 {
			return
		}
		mx, my := wnd.GetCursorClientPos(c.framebuf.Width, c.framebuf.Height)
		hit := render.HitTestCreativeInventory(c.framebuf.Width, c.framebuf.Height, c.creativeTab, c.creativeSearch, c.creativeScrollRow, mx, my, c.creativeHotbars)
		if hit.Type == render.SlotInventory {
			drag.addTarget(hit.Index)
		}
		if len(drag.TargetSlots) >= 2 && distributeCreativeCursor(c.player.Inventory, &c.cursorItem, drag.TargetSlots, btn) > 0 {
			return
		}
		index := drag.StartIndex
		if len(drag.TargetSlots) == 1 {
			index = drag.TargetSlots[0]
		}
		if index < 0 || index >= len(c.player.Inventory.Slots) {
			return
		}
		slot := &c.player.Inventory.Slots[index]
		if btn == 1 {
			handleSlotLeftClick(&c.cursorItem, slot)
		} else if btn == 2 {
			handleSlotRightClick(&c.cursorItem, slot)
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
