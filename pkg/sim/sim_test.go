package sim

import (
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestAABBIntersection(t *testing.T) {
	b1 := AABB{Min: Vec3{0, 0, 0}, Max: Vec3{1, 1, 1}}
	b2 := AABB{Min: Vec3{0.5, 0.5, 0.5}, Max: Vec3{1.5, 1.5, 1.5}}
	b3 := AABB{Min: Vec3{2, 2, 2}, Max: Vec3{3, 3, 3}}

	if !b1.Intersects(b2) {
		t.Fatalf("b1 and b2 should intersect")
	}
	if b1.Intersects(b3) {
		t.Fatalf("b1 and b3 should not intersect")
	}
}

func TestTickNaturalMobSpawnDoesNotDeadlock(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("test", [16]byte{}, Vec3{X: 0.5, Y: 64, Z: 0.5})
	if player == nil {
		t.Fatal("expected player to be added")
	}
	eng.TickCount = 59

	// Natural spawning runs at tick 60 and selects a position 28–47 blocks from
	// the player. Make every candidate a valid grass surface so this regression
	// reliably exercises the spawning path that previously re-locked Engine.mu.
	for chunkX := int32(-4); chunkX <= 3; chunkX++ {
		for chunkZ := int32(-4); chunkZ <= 3; chunkZ++ {
			pos := world.ChunkPos{X: chunkX, Z: chunkZ}
			chunk := world.NewChunk(pos)
			for localX := int32(0); localX < world.ChunkWidth; localX++ {
				for localZ := int32(0); localZ < world.ChunkWidth; localZ++ {
					chunk.SetBlock(chunkX*world.ChunkWidth+localX, 63, chunkZ*world.ChunkWidth+localZ, 2)
				}
			}
			eng.World[pos] = chunk
		}
	}

	done := make(chan struct{})
	go func() {
		eng.Tick()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Engine.Tick deadlocked during natural mob spawning at tick 60")
	}
	eng.mu.RLock()
	mobCount := len(eng.Mobs)
	eng.mu.RUnlock()
	if mobCount == 0 {
		t.Fatal("tick 60 completed without spawning on a valid daytime grass surface")
	}
}

func TestMaceSmashKnocksBackNearbyMobs(t *testing.T) {
	eng := NewEngine()
	target := eng.SpawnMob(MobZombie, Vec3{X: 4, Y: 64, Z: 4})
	nearby := eng.SpawnMob(MobCow, Vec3{X: 5, Y: 64, Z: 4})
	far := eng.SpawnMob(MobPig, Vec3{X: 12, Y: 64, Z: 4})

	eng.ApplyMaceSmashKnockback(target.ID, target.Position, 4, 1.2)
	if nearby.Velocity.X <= 0 || nearby.Velocity.Y <= 0 {
		t.Fatalf("nearby mob was not launched away from mace target: velocity=%+v", nearby.Velocity)
	}
	if far.Velocity != (Vec3{}) {
		t.Fatalf("far mob was affected by mace smash: velocity=%+v", far.Velocity)
	}
	if target.Velocity != (Vec3{}) {
		t.Fatalf("primary target received area knockback in addition to hit knockback: velocity=%+v", target.Velocity)
	}
}

func TestPoplarSaplingRandomTickGrowth(t *testing.T) {
	eng := NewEngine()
	for chunkX := int32(-1); chunkX <= 1; chunkX++ {
		for chunkZ := int32(-1); chunkZ <= 1; chunkZ++ {
			pos := world.ChunkPos{X: chunkX, Z: chunkZ}
			eng.World[pos] = world.NewChunk(pos)
		}
	}
	eng.SetBlock(15, 63, 15, 2)
	eng.SetBlock(15, 64, 15, 10063)

	eng.mu.Lock()
	grown := eng.growPoplarSaplingLocked(15, 64, 15)
	eng.mu.Unlock()
	if !grown {
		t.Fatal("supported poplar sapling with clear sky should grow")
	}
	for dy := int32(0); dy < 5; dy++ {
		if got := eng.GetBlock(15, 64+dy, 15); got != 10010 {
			t.Fatalf("poplar trunk at y=%d = %d, want block 10010", 64+dy, got)
		}
	}
	leafCount := 0
	for x := int32(13); x <= 17; x++ {
		for y := int32(67); y <= 69; y++ {
			for z := int32(13); z <= 17; z++ {
				id := eng.GetBlock(x, y, z)
				if id >= 10060 && id <= 10062 {
					leafCount++
				}
			}
		}
	}
	if leafCount < 30 {
		t.Fatalf("grown poplar has only %d leaves", leafCount)
	}
}

func TestPoplarSaplingGrowsThroughEngineRandomTickLoop(t *testing.T) {
	seed, saplingPos, ok := findPoplarGrowthRandomTick()
	if !ok {
		t.Fatal("failed to find a deterministic random-tick sample for poplar growth")
	}
	eng := NewEngine()
	eng.randomTickState = seed
	eng.World[world.ChunkPos{}] = world.NewChunk(world.ChunkPos{})
	eng.World[world.ChunkPos{}].SetBlock(saplingPos.X, saplingPos.Y-1, saplingPos.Z, 2)
	eng.World[world.ChunkPos{}].SetBlock(saplingPos.X, saplingPos.Y, saplingPos.Z, poplarSaplingID)
	eng.Players[1] = &PlayerEntity{
		ID: 1, IsAlive: true, ClientDriven: true,
		Position: Vec3{X: 0.5, Y: 64, Z: 0.5}, Inventory: inventory.NewPlayerInventory(),
	}

	eng.Tick()
	if got := eng.GetBlock(saplingPos.X, saplingPos.Y, saplingPos.Z); got != poplarLogID {
		t.Fatalf("engine random tick left block %d at sapling position; want poplar log %d", got, poplarLogID)
	}
}

func findPoplarGrowthRandomTick() (uint64, world.BlockPos, bool) {
	next := func(state *uint64) uint64 {
		value := *state
		value ^= value << 13
		value ^= value >> 7
		value ^= value << 17
		*state = value
		return value
	}
	for seed := uint64(1); seed < 10_000; seed++ {
		state := seed
		random := next(&state)
		pos := world.BlockPos{
			X: int32(random & 15), Y: world.MinWorldY + int32((random>>4)&15), Z: int32((random >> 8) & 15),
		}
		if next(&state)%7 == 0 && pos.X >= 2 && pos.X <= 13 && pos.Z >= 2 && pos.Z <= 13 && pos.Y > world.MinWorldY && pos.Y+5 <= world.MaxWorldY {
			return seed, pos, true
		}
	}
	return 0, world.BlockPos{}, false
}

func TestPoplarSaplingGrowthWaitsForObstructionToClear(t *testing.T) {
	eng := NewEngine()
	for chunkX := int32(-1); chunkX <= 1; chunkX++ {
		for chunkZ := int32(-1); chunkZ <= 1; chunkZ++ {
			pos := world.ChunkPos{X: chunkX, Z: chunkZ}
			eng.World[pos] = world.NewChunk(pos)
		}
	}
	eng.SetBlock(15, 63, 15, 2)
	eng.SetBlock(15, 64, 15, 10063)
	eng.SetBlock(17, 67, 15, 1)

	eng.mu.Lock()
	grown := eng.growPoplarSaplingLocked(15, 64, 15)
	eng.mu.Unlock()
	if grown || eng.GetBlock(15, 64, 15) != 10063 {
		t.Fatal("obstructed poplar growth must leave the sapling and terrain untouched")
	}
}

func TestPoplarLeavesBreakAndShearDrops(t *testing.T) {
	makeEngine := func() *Engine {
		eng := NewEngine()
		pos := world.ChunkPos{X: 0, Z: 0}
		eng.World[pos] = world.NewChunk(pos)
		eng.World[pos].SetBlock(2, 64, 2, 10060)
		eng.Players[1] = &PlayerEntity{ID: 1, IsAlive: true, Inventory: inventory.NewPlayerInventory()}
		eng.randomTickState = 1 // deterministic roll that does not drop a sapling
		return eng
	}

	hand := makeEngine()
	if dropped := hand.BreakBlock(1, world.BlockPos{X: 2, Y: 64, Z: 2}); dropped != inventory.ItemAir {
		t.Fatalf("hand-breaking poplar leaves dropped %d, want no drop", dropped)
	}
	if got := hand.Players[1].Inventory.Slots[0]; !got.IsEmpty() {
		t.Fatalf("hand-breaking leaves unexpectedly gave an item: %+v", got)
	}

	shears := makeEngine()
	drop, harvested := shears.BreakBlockWithTool(1, world.BlockPos{X: 2, Y: 64, Z: 2}, inventory.ItemShears)
	if !harvested || drop != 10060 {
		t.Fatalf("shears breaking poplar leaves = (drop %d, harvested %t), want (10060, true)", drop, harvested)
	}
}

func TestModernDeepslateHarvestAndDropRules(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("test", [16]byte{}, Vec3{X: 0.5, Y: 64, Z: 0.5})
	pos := world.BlockPos{X: 2, Y: 64, Z: 2}
	eng.SetBlock(pos.X, pos.Y, pos.Z, 12006) // Deepslate
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemWoodenPickaxe); harvested || drop != inventory.ItemAir {
		t.Fatalf("wood pickaxe deepslate break = (%d, %v), want no drop", drop, harvested)
	}
	eng.SetBlock(pos.X, pos.Y, pos.Z, 12006)
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemStonePickaxe); !harvested || drop != 12007 {
		t.Fatalf("stone pickaxe deepslate break = (%d, %v), want cobbled deepslate (12007)", drop, harvested)
	}

	eng.SetBlock(pos.X, pos.Y, pos.Z, 12006)
	player.Inventory.Slots[0] = inventory.ItemStack{}
	if drop := eng.BreakBlock(player.ID, pos); drop != inventory.ItemAir {
		t.Fatalf("hand-mined deepslate drop = %d, want air/no item", drop)
	}
}

func TestStainedGlassBreaksWithoutCreatingDrops(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("glass miner", [16]byte{0x17}, Vec3{X: 0.5, Y: 64, Z: 0.5})
	pos := world.BlockPos{X: 2, Y: 64, Z: 2}
	const whiteStainedGlass int32 = 17100
	if !eng.SetBlockChecked(pos.X, pos.Y, pos.Z, whiteStainedGlass) {
		t.Fatal("failed to place stained glass test block")
	}
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemDiamondPickaxe); !harvested || drop != inventory.ItemAir {
		t.Fatalf("tool break stained glass = (drop %d, harvested %t), want harvested with no item", drop, harvested)
	}
	if len(eng.ItemDrops) != 0 {
		t.Fatalf("tool-breaking stained glass spawned item entities: %+v", eng.ItemDrops)
	}
	if !eng.SetBlockChecked(pos.X, pos.Y, pos.Z, whiteStainedGlass) {
		t.Fatal("failed to restore stained glass test block")
	}
	if drop := eng.BreakBlock(player.ID, pos); drop != inventory.ItemAir {
		t.Fatalf("hand-breaking stained glass drop = %d, want no item", drop)
	}
	if len(eng.ItemDrops) != 0 {
		t.Fatalf("hand-breaking stained glass spawned item entities: %+v", eng.ItemDrops)
	}
	const whiteGlassPane int32 = 17116
	if !eng.SetBlockChecked(pos.X, pos.Y, pos.Z, whiteGlassPane) {
		t.Fatal("failed to place glass pane test block")
	}
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemAir); !harvested || drop != inventory.ItemAir {
		t.Fatalf("hand-breaking glass pane = (drop %d, harvested %t), want harvested with no item", drop, harvested)
	}
	if len(eng.ItemDrops) != 0 {
		t.Fatalf("breaking glass pane spawned item entities: %+v", eng.ItemDrops)
	}
}

func TestCandlesRequireSupportAndDropTheirItem(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("candle placer", [16]byte{0x18}, Vec3{X: 0.5, Y: 64, Z: 0.5})
	const candleID int32 = 17133
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: candleID, Count: 2}
	if !eng.SetBlockChecked(10, 63, 10, inventory.ItemStone) {
		t.Fatal("failed to place candle support block")
	}
	if eng.PlaceBlock(player.ID, world.BlockPos{X: 11, Y: 64, Z: 10}, candleID) {
		t.Fatal("candle was placed without a supporting block")
	}
	if !eng.PlaceBlock(player.ID, world.BlockPos{X: 10, Y: 64, Z: 10}, candleID) {
		t.Fatal("candle was not placed on a solid top face")
	}
	if !eng.PlaceBlock(player.ID, world.BlockPos{X: 10, Y: 64, Z: 10}, candleID) {
		t.Fatal("matching candle did not stack on the clicked candle block")
	}
	if got := eng.GetBlock(10, 64, 10); got != inventory.CandleBlockState(candleID, 2, false) {
		t.Fatalf("placed candle state = %d, want two unlit candles", got)
	}
	if !eng.SetCandleLit(world.BlockPos{X: 10, Y: 64, Z: 10}, true) {
		t.Fatal("failed to light candle cluster")
	}
	if itemID, count, lit, ok := inventory.CandleStateProperties(eng.GetBlock(10, 64, 10)); !ok || itemID != candleID || count != 2 || !lit {
		t.Fatalf("lit candle properties = (%d,%d,%t,%t)", itemID, count, lit, ok)
	}
	drop, harvested := eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 10, Y: 64, Z: 10}, inventory.ItemAir)
	if !harvested || drop != candleID {
		t.Fatalf("breaking candle = (drop %d, harvested %t), want candle item %d", drop, harvested, candleID)
	}
	if drop := eng.ItemDrops[eng.nextItemDropID]; drop == nil || drop.ItemID != candleID || drop.Count != 2 {
		t.Fatalf("breaking two-candle cluster spawned %+v, want two candle items", drop)
	}
}

func TestWoodBlockVariantsDropTheirOwnItems(t *testing.T) {
	for _, name := range []string{
		"oak_wood", "stripped_oak_wood", "spruce_wood", "stripped_spruce_wood",
		"birch_wood", "stripped_birch_wood", "jungle_wood", "stripped_jungle_wood",
		"acacia_wood", "stripped_acacia_wood", "dark_oak_wood", "stripped_dark_oak_wood",
		"mangrove_wood", "stripped_mangrove_wood", "cherry_wood", "stripped_cherry_wood",
		"pale_oak_wood", "stripped_pale_oak_wood", "crimson_hyphae", "stripped_crimson_hyphae",
		"warped_hyphae", "stripped_warped_hyphae",
	} {
		t.Run(name, func(t *testing.T) {
			block, ok := inventory.ExtendedBlockByName("minecraft:" + name)
			if !ok {
				t.Fatalf("wood block %s is not registered", name)
			}
			eng := NewEngine()
			player := eng.AddPlayer("wood miner", [16]byte{}, Vec3{X: 0.5, Y: 64, Z: 0.5})
			pos := world.BlockPos{X: 1, Y: 64, Z: 1}
			eng.SetBlock(pos.X, pos.Y, pos.Z, block.RuntimeID)
			if got, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemWoodenAxe); !harvested || got != block.RuntimeID {
				t.Fatalf("breaking %s returned (%d, %t), want itself (%d, true)", name, got, harvested, block.RuntimeID)
			}
			drops := eng.GetItemDrops()
			if len(drops) != 1 || drops[0].ItemID != block.RuntimeID || drops[0].Count != 1 {
				t.Fatalf("breaking %s produced drops %+v, want one of block ID %d", name, drops, block.RuntimeID)
			}
		})
	}
}

func TestCommonVanillaCubeBlocksBreakAndDropTheirOwnItems(t *testing.T) {
	blocks := []struct {
		name string
		tool int32
	}{
		{"bone_block", inventory.ItemAir},
		{"chiseled_quartz_block", inventory.ItemAir},
		{"chiseled_red_sandstone", inventory.ItemAir},
		{"chiseled_sandstone", inventory.ItemAir},
		{"coarse_dirt", inventory.ItemAir},
		{"end_stone_bricks", inventory.ItemAir},
		{"honeycomb_block", inventory.ItemAir},
		{"lodestone", inventory.ItemAir},
		{"muddy_mangrove_roots", inventory.ItemAir},
		{"purpur_pillar", inventory.ItemAir},
		{"quartz_pillar", inventory.ItemAir},
		{"rooted_dirt", inventory.ItemAir},
		{"shroomlight", inventory.ItemAir},
		{"smooth_quartz", inventory.ItemAir},
		{"smooth_red_sandstone", inventory.ItemAir},
		{"smooth_sandstone", inventory.ItemAir},
		{"smooth_stone", inventory.ItemAir},
		{"soul_soil", inventory.ItemAir},
		{"wet_sponge", inventory.ItemAir},
		{"red_sand", inventory.ItemAir},
	}
	eng := NewEngine()
	player := eng.AddPlayer("cube-breaker", [16]byte{0xC3}, Vec3{X: 0, Y: 64, Z: 0})
	for i, block := range blocks {
		t.Run(block.name, func(t *testing.T) {
			definition, ok := inventory.ExtendedBlockByName("minecraft:" + block.name)
			if !ok {
				t.Fatalf("missing block definition for %s", block.name)
			}
			pos := world.BlockPos{X: int32(i + 1), Y: 64, Z: 0}
			eng.SetBlock(pos.X, pos.Y, pos.Z, definition.RuntimeID)
			got, harvested := eng.BreakBlockWithTool(player.ID, pos, block.tool)
			if !harvested || got != definition.RuntimeID {
				t.Fatalf("breaking %s returned (%d, %t), want (%d, true)", block.name, got, harvested, definition.RuntimeID)
			}
		})
	}
}

func TestNetheriteBlockRequiresDiamondTierToDrop(t *testing.T) {
	block, ok := inventory.ExtendedBlockByName("minecraft:netherite_block")
	if !ok {
		t.Fatal("netherite block definition is missing")
	}
	eng := NewEngine()
	player := eng.AddPlayer("netherite miner", [16]byte{0xC4}, Vec3{X: 0.5, Y: 64, Z: 0.5})
	pos := world.BlockPos{X: 2, Y: 64, Z: 2}
	eng.SetBlock(pos.X, pos.Y, pos.Z, block.RuntimeID)
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemIronPickaxe); harvested || drop != inventory.ItemAir {
		t.Fatalf("iron pickaxe netherite block break = (%d, %t), want no drop", drop, harvested)
	}
	eng.SetBlock(pos.X, pos.Y, pos.Z, block.RuntimeID)
	if drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemDiamondPickaxe); !harvested || drop != block.RuntimeID {
		t.Fatalf("diamond pickaxe netherite block break = (%d, %t), want self-drop %d", drop, harvested, block.RuntimeID)
	}
}

func TestAddedOresRequireTheRightPickaxeAndDropTheirMaterials(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("ore miner", [16]byte{}, Vec3{X: 0.5, Y: 64, Z: 0.5})
	cases := []struct {
		name    string
		tool    int32
		drop    int32
		minDrop int
		maxDrop int
	}{
		{"minecraft:copper_ore", inventory.ItemStonePickaxe, inventory.ItemRawCopper, 2, 5},
		{"minecraft:deepslate_copper_ore", inventory.ItemStonePickaxe, inventory.ItemRawCopper, 2, 5},
		{"minecraft:deepslate_coal_ore", inventory.ItemStonePickaxe, inventory.ItemCoal, 1, 1},
		{"minecraft:deepslate_iron_ore", inventory.ItemStonePickaxe, inventory.ItemRawIron, 1, 1},
		{"minecraft:deepslate_gold_ore", inventory.ItemIronPickaxe, inventory.ItemRawGold, 1, 1},
		{"minecraft:deepslate_diamond_ore", inventory.ItemIronPickaxe, inventory.ItemDiamond, 1, 1},
		{"minecraft:deepslate_emerald_ore", inventory.ItemIronPickaxe, inventory.ItemEmerald, 1, 1},
		{"minecraft:deepslate_lapis_ore", inventory.ItemStonePickaxe, inventory.ItemLapisLazuli, 4, 9},
		{"minecraft:deepslate_redstone_ore", inventory.ItemIronPickaxe, 331, 4, 5},
		{"minecraft:nether_quartz_ore", inventory.ItemStonePickaxe, inventory.ItemNetherQuartz, 1, 1},
	}
	for i, tc := range cases {
		block, ok := inventory.ExtendedBlockByName(tc.name)
		if !ok {
			t.Fatalf("missing %s", tc.name)
		}
		pos := world.BlockPos{X: int32(i + 1), Y: 64, Z: 0}
		eng.SetBlock(pos.X, pos.Y, pos.Z, block.RuntimeID)
		if got, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemAir); harvested || got != inventory.ItemAir {
			t.Fatalf("hand mined %s as (%d, %v), want no drop", tc.name, got, harvested)
		}
		eng.SetBlock(pos.X, pos.Y, pos.Z, block.RuntimeID)
		previousDrops := len(eng.GetItemDrops())
		if got, harvested := eng.BreakBlockWithTool(player.ID, pos, tc.tool); !harvested || got != tc.drop {
			t.Fatalf("proper pickaxe mined %s as (%d, %v), want drop %d", tc.name, got, harvested, tc.drop)
		}
		drops := eng.GetItemDrops()
		if len(drops) != previousDrops+1 {
			t.Fatalf("mining %s created %d drops, want one", tc.name, len(drops)-previousDrops)
		}
		gotCount := 0
		for _, drop := range drops {
			if drop.ItemID == tc.drop && drop.Position.X == float64(pos.X)+0.5 {
				gotCount = drop.Count
				break
			}
		}
		if gotCount < tc.minDrop || gotCount > tc.maxDrop {
			t.Fatalf("mining %s dropped %d items, want range %d..%d", tc.name, gotCount, tc.minDrop, tc.maxDrop)
		}
	}
	ancientDebris, _ := inventory.ExtendedBlockByName("minecraft:ancient_debris")
	pos := world.BlockPos{X: 20, Y: 64, Z: 0}
	eng.SetBlock(pos.X, pos.Y, pos.Z, ancientDebris.RuntimeID)
	if got, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemIronPickaxe); harvested || got != inventory.ItemAir {
		t.Fatalf("iron pickaxe mined ancient debris as (%d, %v), want no drop", got, harvested)
	}
	eng.SetBlock(pos.X, pos.Y, pos.Z, ancientDebris.RuntimeID)
	if got, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemDiamondPickaxe); !harvested || got != ancientDebris.RuntimeID {
		t.Fatalf("diamond pickaxe mined ancient debris as (%d, %v), want the block item %d", got, harvested, ancientDebris.RuntimeID)
	}
}

func TestLegacyOreMiningUsesVanillaToolAndDropRules(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("ore miner", [16]byte{}, Vec3{X: 0.5, Y: 64, Z: 0.5})
	pos := world.BlockPos{X: 2, Y: 64, Z: 0}
	eng.SetBlock(pos.X, pos.Y, pos.Z, 73) // Redstone ore
	if got := eng.BreakBlock(player.ID, pos); got != inventory.ItemAir {
		t.Fatalf("hand-mined redstone ore drop=%d, want no drop", got)
	}
	eng.SetBlock(pos.X, pos.Y, pos.Z, 73)
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemIronPickaxe, Count: 1}
	if got := eng.BreakBlock(player.ID, pos); got != 331 {
		t.Fatalf("iron-pickaxe redstone drop=%d, want redstone dust", got)
	}
	count := 0
	for _, stack := range player.Inventory.Slots {
		if stack.ItemID == 331 {
			count += stack.Count
		}
	}
	if count < 4 || count > 5 {
		t.Fatalf("redstone drop count=%d, want 4..5", count)
	}
}

func TestRaycastHitBlock(t *testing.T) {
	solidBlocks := map[[3]int32]bool{
		{5, 64, 0}: true,
	}

	isSolid := func(x, y, z int32) bool {
		return solidBlocks[[3]int32{x, y, z}]
	}

	origin := Vec3{0.5, 64.5, 0.5}
	dir := Vec3{1.0, 0.0, 0.0}

	res := Raycast(origin, dir, 10.0, isSolid)
	if !res.Hit {
		t.Fatalf("raycast should hit solid block at (5, 64, 0)")
	}
	if res.BlockX != 5 || res.BlockY != 64 || res.BlockZ != 0 {
		t.Fatalf("hit block mismatch: (%d, %d, %d)", res.BlockX, res.BlockY, res.BlockZ)
	}
	if res.FaceX != -1 || res.FaceY != 0 || res.FaceZ != 0 {
		t.Fatalf("hit face mismatch: (%d, %d, %d)", res.FaceX, res.FaceY, res.FaceZ)
	}
}

func TestMiningDropsToInventory(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Miner", [16]byte{1}, Vec3{0, 64, 0})

	// 1. Break Stone (1) -> drops Cobblestone (4)
	eng.SetBlock(1, 64, 0, 1)
	drop := eng.BreakBlock(p.ID, world.BlockPos{X: 1, Y: 64, Z: 0})
	if drop != inventory.ItemCobblestone {
		t.Fatalf("expected cobblestone drop, got %d", drop)
	}
	if p.Inventory.Slots[0].ItemID != inventory.ItemCobblestone || p.Inventory.Slots[0].Count != 1 {
		t.Fatalf("item not added to inventory: %+v", p.Inventory.Slots[0])
	}

	// 2. Break Diamond Ore (56) -> drops Diamond (264)
	p.Inventory.Slots[1] = inventory.ItemStack{ItemID: inventory.ItemIronPickaxe, Count: 1}
	p.Inventory.SelectedHotbar = 1
	eng.SetBlock(2, 64, 0, 56)
	drop = eng.BreakBlock(p.ID, world.BlockPos{X: 2, Y: 64, Z: 0})
	if drop != inventory.ItemDiamond {
		t.Fatalf("expected diamond drop, got %d", drop)
	}
}

func TestCannotPlaceBlockInsidePlayer(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{1}, Vec3{10.5, 64.0, 10.5})

	// Attempt to place block right at (10, 64, 10) which intersects player
	success := eng.PlaceBlock(p.ID, world.BlockPos{X: 10, Y: 64, Z: 10}, 1)
	if success {
		t.Fatalf("should not be able to place block inside player")
	}

	// Place block next to player at (12, 64, 10) -> should succeed
	success = eng.PlaceBlock(p.ID, world.BlockPos{X: 12, Y: 64, Z: 10}, 1)
	if !success {
		t.Fatalf("should be able to place block adjacent to player")
	}
}

func TestSlabPlacementMergeCollisionAndDrops(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{9}, Vec3{X: 20.5, Y: 64, Z: 20.5})
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemStoneSlab, Count: 2}
	pos := world.BlockPos{X: 10, Y: 64, Z: 10}
	if !eng.PlaceBlock(player.ID, pos, inventory.ItemStoneSlab) {
		t.Fatal("bottom stone slab placement failed")
	}
	if got := eng.GetBlock(pos.X, pos.Y, pos.Z); got != inventory.ItemStoneSlab {
		t.Fatalf("bottom slab state = %d, want item state %d", got, inventory.ItemStoneSlab)
	}
	if got := player.Inventory.Slots[0].Count; got != 1 {
		t.Fatalf("survival placement left %d slabs, want 1", got)
	}
	if !eng.PlaceBlock(player.ID, pos, inventory.BlockStoneSlabDouble) {
		t.Fatal("combining a second slab failed")
	}
	if got := eng.GetBlock(pos.X, pos.Y, pos.Z); got != inventory.BlockStoneSlabDouble {
		t.Fatalf("merged slab state = %d, want double state %d", got, inventory.BlockStoneSlabDouble)
	}
	if got := player.Inventory.Slots[0].Count; got != 0 {
		t.Fatalf("merge left %d slabs, want 0", got)
	}
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemStonePickaxe, Count: 1}
	drop, harvested := eng.BreakBlockWithTool(player.ID, pos, inventory.ItemStonePickaxe)
	if !harvested || drop != inventory.ItemStoneSlab {
		t.Fatalf("breaking double slab returned (%d,%t), want canonical stone slab", drop, harvested)
	}
	foundTwoSlabs := false
	for _, itemDrop := range eng.ItemDrops {
		if itemDrop.ItemID == inventory.ItemStoneSlab && itemDrop.Count == 2 {
			foundTwoSlabs = true
			break
		}
	}
	if !foundTwoSlabs {
		t.Fatal("breaking a double stone slab should drop two canonical slab items")
	}

	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemOakSlab, Count: 1}
	topPos := world.BlockPos{X: 12, Y: 64, Z: 10}
	if !eng.PlaceBlock(player.ID, topPos, inventory.BlockOakSlabTop) {
		t.Fatal("top oak slab placement failed")
	}
	if got := eng.GetBlock(topPos.X, topPos.Y, topPos.Z); got != inventory.BlockOakSlabTop {
		t.Fatalf("top oak slab state = %d, want %d", got, inventory.BlockOakSlabTop)
	}

	// Exercise a modern wood family through the real placement, merge, and
	// harvest path rather than only testing the shared state helpers.
	spruce := inventory.FirstWoodSlabItem
	spruceDouble := inventory.FirstWoodSlabDouble
	woodPos := world.BlockPos{X: 14, Y: 64, Z: 10}
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: spruce, Count: 2}
	if !eng.PlaceBlock(player.ID, woodPos, spruce) || !eng.PlaceBlock(player.ID, woodPos, spruceDouble) {
		t.Fatal("spruce slab placement and merge failed")
	}
	if got := eng.GetBlock(woodPos.X, woodPos.Y, woodPos.Z); got != spruceDouble {
		t.Fatalf("merged spruce slab state = %d, want %d", got, spruceDouble)
	}
	if got, harvested := eng.BreakBlockWithTool(player.ID, woodPos, inventory.ItemWoodenAxe); !harvested || got != spruce {
		t.Fatalf("breaking double spruce slab returned (%d,%t), want canonical spruce slab", got, harvested)
	}
	foundTwoWoodSlabs := false
	for _, itemDrop := range eng.ItemDrops {
		if itemDrop.ItemID == spruce && itemDrop.Count == 2 {
			foundTwoWoodSlabs = true
			break
		}
	}
	if !foundTwoWoodSlabs {
		t.Fatal("breaking a double spruce slab should drop two spruce slab items")
	}

	whiteConcreteSlab := inventory.FirstColorSlabItem + 1
	whiteConcreteDouble := inventory.FirstColorSlabDouble + 1
	concretePos := world.BlockPos{X: 16, Y: 64, Z: 10}
	if !eng.SetBlockChecked(concretePos.X, concretePos.Y, concretePos.Z, whiteConcreteDouble) {
		t.Fatal("failed to install double concrete slab test state")
	}
	if _, harvested := eng.BreakBlockWithTool(player.ID, concretePos, inventory.ItemAir); harvested {
		t.Fatal("breaking a double concrete slab by hand should not harvest its drop")
	}
	for _, itemDrop := range eng.ItemDrops {
		if itemDrop.ItemID == whiteConcreteSlab {
			t.Fatal("double concrete slab should not drop without a pickaxe")
		}
	}
	if !eng.SetBlockChecked(concretePos.X, concretePos.Y, concretePos.Z, whiteConcreteDouble) {
		t.Fatal("failed to reinstall double concrete slab test state")
	}
	if drop, broken := eng.BreakBlockWithTool(player.ID, concretePos, inventory.ItemWoodenPickaxe); !broken || drop != whiteConcreteSlab {
		t.Fatalf("breaking a double concrete slab with a wooden pickaxe returned (%d,%t), want %d", drop, broken, whiteConcreteSlab)
	}

	cobblestoneSlab := inventory.FirstMasonrySlabItem
	cobblestoneDouble := inventory.FirstMasonrySlabDouble
	masonryPos := world.BlockPos{X: 18, Y: 64, Z: 10}
	if !eng.SetBlockChecked(masonryPos.X, masonryPos.Y, masonryPos.Z, cobblestoneDouble) {
		t.Fatal("failed to install double cobblestone slab test state")
	}
	if _, harvested := eng.BreakBlockWithTool(player.ID, masonryPos, inventory.ItemAir); harvested {
		t.Fatal("double cobblestone slab should not drop without a pickaxe")
	}
	if !eng.SetBlockChecked(masonryPos.X, masonryPos.Y, masonryPos.Z, cobblestoneDouble) {
		t.Fatal("failed to reinstall double cobblestone slab test state")
	}
	if drop, harvested := eng.BreakBlockWithTool(player.ID, masonryPos, inventory.ItemWoodenPickaxe); !harvested || drop != cobblestoneSlab {
		t.Fatalf("breaking double cobblestone slab with a wooden pickaxe returned (%d,%t), want %d", drop, harvested, cobblestoneSlab)
	}
}

func TestSlabCollisionAndShapeAwareRaycast(t *testing.T) {
	getBlock := func(x, y, z int32) int32 {
		if x == 2 && y == 0 && z == 0 {
			return inventory.ItemStoneSlab
		}
		return 0
	}
	resolved, onGround, _ := ResolveMovementCollisionForBlocks(Vec3{X: 2.5, Y: 1, Z: .5}, Vec3{Y: -1}, getBlock)
	if !onGround || math.Abs(resolved.Y-.5) > 1e-8 {
		t.Fatalf("fall onto bottom slab resolved to %+v, onGround=%t; want Y=.5", resolved, onGround)
	}
	miss := RaycastBlocks(Vec3{X: .5, Y: .75, Z: .5}, Vec3{X: 1}, 5, getBlock)
	if miss.Hit {
		t.Fatalf("ray above bottom slab incorrectly hit %+v", miss)
	}
	hit := RaycastBlocks(Vec3{X: .5, Y: .25, Z: .5}, Vec3{X: 1}, 5, getBlock)
	if !hit.Hit || hit.FaceX != -1 || math.Abs(hit.HitPos.Y-.25) > 1e-8 {
		t.Fatalf("ray through slab side returned %+v; want a lower-half side hit", hit)
	}
}

func TestJava263ColoredStairPlacementCollisionAndDrops(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Builder", [16]byte{43}, Vec3{X: 20.5, Y: 64, Z: 20.5})
	woolStair := inventory.FirstColorStairItem
	woolState, ok := inventory.StairBlockState(woolStair, inventory.StairFacing(0), false)
	if !ok {
		t.Fatal("failed to create white wool stair state")
	}
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: woolStair, Count: 2}
	woolPos := world.BlockPos{X: 10, Y: 64, Z: 10}
	if !eng.PlaceBlock(player.ID, woolPos, woolState) {
		t.Fatal("white wool stair placement failed")
	}
	if got := eng.GetBlock(woolPos.X, woolPos.Y, woolPos.Z); got != woolState {
		t.Fatalf("placed wool stair state = %d, want %d", got, woolState)
	}
	if got := player.Inventory.Slots[0].Count; got != 1 {
		t.Fatalf("wool stair placement left %d items, want 1", got)
	}
	if drop, harvested := eng.BreakBlockWithTool(player.ID, woolPos, inventory.ItemAir); !harvested || drop != woolStair {
		t.Fatalf("breaking wool stair by hand returned (%d,%t), want item %d", drop, harvested, woolStair)
	}

	concreteStair := woolStair + 1
	concreteState, ok := inventory.StairBlockState(concreteStair, inventory.StairFacing(1), true)
	if !ok {
		t.Fatal("failed to create white concrete stair state")
	}
	concretePos := world.BlockPos{X: 12, Y: 64, Z: 10}
	if !eng.SetBlockChecked(concretePos.X, concretePos.Y, concretePos.Z, concreteState) {
		t.Fatal("failed to install concrete stair test state")
	}
	if drop, harvested := eng.BreakBlockWithTool(player.ID, concretePos, inventory.ItemAir); harvested || drop != 0 {
		t.Fatalf("breaking concrete stair by hand returned (%d,%t), want no drop", drop, harvested)
	}
	if !eng.SetBlockChecked(concretePos.X, concretePos.Y, concretePos.Z, concreteState) {
		t.Fatal("failed to reinstall concrete stair test state")
	}
	if drop, harvested := eng.BreakBlockWithTool(player.ID, concretePos, inventory.ItemWoodenPickaxe); !harvested || drop != concreteStair {
		t.Fatalf("breaking concrete stair with wooden pickaxe returned (%d,%t), want item %d", drop, harvested, concreteStair)
	}

	lower, ok := inventory.StairBlockState(woolStair, inventory.StairFacing(2), false)
	if !ok {
		t.Fatal("failed to construct lower stair state")
	}
	boxes := BlockCollisionAABBs(lower, 0, 0, 0)
	if len(boxes) != 2 || boxes[0].Max.Y != .5 || boxes[1].Min.Y != .5 || boxes[1].Max.Y != 1 {
		t.Fatalf("lower stair collision boxes = %+v", boxes)
	}
	upper, _ := inventory.StairBlockState(woolStair, inventory.StairFacing(2), true)
	upperBoxes := BlockCollisionAABBs(upper, 0, 0, 0)
	if len(upperBoxes) != 2 || upperBoxes[0].Min.Y != .5 || upperBoxes[1].Max.Y != .5 {
		t.Fatalf("upper stair collision boxes = %+v", upperBoxes)
	}
}

func TestWoodStairPlacementAndDrop(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{98}, Vec3{X: 10.5, Y: 64, Z: 10.5})
	eng.SetBlock(11, 63, 10, inventory.ItemStone)
	state, ok := inventory.StairBlockState(inventory.FirstWoodStairItem, inventory.StairFacing(1), true)
	if !ok {
		t.Fatal("could not construct wood stair state")
	}
	pos := world.BlockPos{X: 11, Y: 64, Z: 10}
	if !eng.PlaceBlock(player.ID, pos, state) {
		t.Fatal("wood stair placement failed")
	}
	if got := eng.GetBlock(pos.X, pos.Y, pos.Z); got != state {
		t.Fatalf("placed block state = %d, want %d", got, state)
	}
	if got := eng.BreakBlock(player.ID, pos); got != inventory.FirstWoodStairItem {
		t.Fatalf("wood stair drop = %d, want item %d", got, inventory.FirstWoodStairItem)
	}
}

func TestRedShrubPlacementCollisionAndDrop(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{1}, Vec3{X: 10.5, Y: 64, Z: 10.5})
	if IsSolidBlock(10100) {
		t.Fatal("red shrub should not collide with players")
	}
	if eng.PlaceBlock(player.ID, world.BlockPos{X: 11, Y: 64, Z: 10}, 10100) {
		t.Fatal("red shrub should not place without a supporting block")
	}
	eng.SetBlock(11, 63, 10, inventory.ItemStone)
	if !eng.PlaceBlock(player.ID, world.BlockPos{X: 11, Y: 64, Z: 10}, 10100) {
		t.Fatal("red shrub should place on a solid block")
	}
	if got := eng.GetBlock(11, 64, 10); got != 10100 {
		t.Fatalf("placed red shrub ID = %d, want 10100", got)
	}
	if got := eng.BreakBlock(player.ID, world.BlockPos{X: 11, Y: 64, Z: 10}); got != 10100 {
		t.Fatalf("breaking red shrub dropped item %d, want 10100", got)
	}
	if !inventory.IsReplaceableBlock(10100) {
		t.Fatal("red shrub should be replaceable by another block")
	}
}

func TestShelfMushroomPlacementGrowthAndDrops(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{21}, Vec3{X: 20.5, Y: 64, Z: 20.5})
	pos := world.BlockPos{X: 21, Y: 64, Z: 20}
	state := inventory.ShelfMushroomBlockState(1, 0) // East facing, supported by the west block.
	if eng.PlaceBlock(player.ID, pos, state) {
		t.Fatal("shelf mushroom placed without a sturdy side support")
	}
	eng.SetBlock(pos.X-1, pos.Y, pos.Z, inventory.ItemStone)
	if !eng.PlaceBlock(player.ID, pos, state) {
		t.Fatal("shelf mushroom failed to attach to a sturdy side")
	}
	if got := eng.GetBlock(pos.X, pos.Y, pos.Z); got != state {
		t.Fatalf("placed state=%d want %d", got, state)
	}
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemBoneMeal, Count: 1}
	if !eng.UseBoneMealOnShelfMushroom(player.ID, pos) {
		t.Fatal("bone meal failed to grow small shelf mushroom")
	}
	large := inventory.ShelfMushroomBlockState(1, 1)
	if got := eng.GetBlock(pos.X, pos.Y, pos.Z); got != large {
		t.Fatalf("grown state=%d want %d", got, large)
	}
	if dropped := eng.BreakBlock(player.ID, pos); dropped != inventory.ItemShelfMushroom {
		t.Fatalf("drop ID=%d want %d", dropped, inventory.ItemShelfMushroom)
	}
	count := 0
	for _, stack := range player.Inventory.Slots {
		if stack.ItemID == inventory.ItemShelfMushroom {
			count += stack.Count
		}
	}
	if count != 2 {
		t.Fatalf("large shelf mushroom drop count=%d want 2", count)
	}
	if !eng.PlaceBlock(player.ID, pos, inventory.ShelfMushroomBlockState(1, 0)) {
		t.Fatal("shelf mushroom could not be placed for support-removal check")
	}
	eng.BreakBlock(player.ID, world.BlockPos{X: pos.X - 1, Y: pos.Y, Z: pos.Z})
	if got := eng.GetBlock(pos.X, pos.Y, pos.Z); got != 0 {
		t.Fatalf("shelf mushroom remained after its support broke: state=%d", got)
	}
	count = 0
	for _, stack := range player.Inventory.Slots {
		if stack.ItemID == inventory.ItemShelfMushroom {
			count += stack.Count
		}
	}
	if count != 2 {
		t.Fatalf("support break shelf mushroom drop count=%d want total 2", count)
	}
}

func TestShelfMushroomLandingBouncesAndCancelsFallDamage(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("mushroom-bounce", [16]byte{}, Vec3{X: 8.7, Y: 70, Z: 8.5})
	if !eng.SetBlockChecked(7, 64, 8, 1) {
		t.Fatal("failed to place shelf mushroom support")
	}
	if !eng.SetBlockChecked(8, 64, 8, inventory.ShelfMushroomBlockState(1, 1)) {
		t.Fatal("failed to place large shelf mushroom")
	}
	player.Velocity.Y = -8
	player.FallDistance = 8
	position, onGround, hitCeiling := eng.MovePlayerRelativeWithSneaking(player.ID, Vec3{Y: -8}, 0, 0, false)
	if hitCeiling || onGround || math.Abs(position.Y-64.6875) > 1e-4 {
		t.Fatalf("shelf landing resolved to position=%+v ground=%v ceiling=%v", position, onGround, hitCeiling)
	}
	if math.Abs(player.Velocity.Y-5.28) > 1e-4 {
		t.Fatalf("bounce velocity Y = %.4f, want 5.28", player.Velocity.Y)
	}
	if player.FallDistance != 0 || player.Health != 20 {
		t.Fatalf("shelf bounce left fall distance %.2f and health %.1f", player.FallDistance, player.Health)
	}

	sneakEngine := NewEngine()
	sneakingPlayer := sneakEngine.AddPlayer("mushroom-sneak", [16]byte{}, Vec3{X: 8.7, Y: 70, Z: 8.5})
	if !sneakEngine.SetBlockChecked(7, 64, 8, 1) ||
		!sneakEngine.SetBlockChecked(8, 64, 8, inventory.ShelfMushroomBlockState(1, 1)) {
		t.Fatal("failed to prepare sneaking shelf landing")
	}
	sneakingPlayer.Velocity.Y = -8
	sneakingPlayer.FallDistance = 8
	_, sneakingGrounded, _ := sneakEngine.MovePlayerRelativeWithSneaking(sneakingPlayer.ID, Vec3{Y: -8}, 0, 0, true)
	if !sneakingGrounded || sneakingPlayer.Velocity.Y != 0 || sneakingPlayer.Health != 15 {
		t.Fatalf("sneaking landing grounded=%v velocity=%v health=%.1f, want grounded without bounce and normal fall damage", sneakingGrounded, sneakingPlayer.Velocity, sneakingPlayer.Health)
	}
}

func TestAdditionalTreeSaplingPlacementAndShearedLeafDrop(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{22}, Vec3{X: 32.5, Y: 64, Z: 32.5})
	sapling, ok := inventory.ExtendedBlockByName("minecraft:cherry_sapling")
	if !ok {
		t.Fatal("cherry sapling definition missing")
	}
	pos := world.BlockPos{X: 33, Y: 64, Z: 32}
	if eng.PlaceBlock(player.ID, pos, sapling.RuntimeID) {
		t.Fatal("sapling placed without soil")
	}
	eng.SetBlock(pos.X, pos.Y-1, pos.Z, inventory.ItemDirt)
	if !eng.PlaceBlock(player.ID, pos, sapling.RuntimeID) {
		t.Fatal("sapling failed to place on dirt")
	}
	if got := eng.GetBlock(pos.X, pos.Y, pos.Z); got != sapling.RuntimeID {
		t.Fatalf("sapling state=%d want %d", got, sapling.RuntimeID)
	}
	eng.BreakBlock(player.ID, world.BlockPos{X: pos.X, Y: pos.Y - 1, Z: pos.Z})
	if got := eng.GetBlock(pos.X, pos.Y, pos.Z); got != 0 {
		t.Fatalf("sapling remained after its soil was removed: state=%d", got)
	}
	if player.Inventory.Slots[0].ItemID != sapling.RuntimeID || player.Inventory.Slots[0].Count != 1 {
		t.Fatalf("unsupported sapling drop=%+v, want one item %d", player.Inventory.Slots[0], sapling.RuntimeID)
	}

	leaves, ok := inventory.ExtendedBlockByName("minecraft:cherry_leaves")
	if !ok {
		t.Fatal("cherry leaves definition missing")
	}
	leafPos := world.BlockPos{X: 34, Y: 64, Z: 32}
	eng.SetBlock(leafPos.X, leafPos.Y, leafPos.Z, leaves.RuntimeID)
	if dropped, harvested := eng.BreakBlockWithTool(player.ID, leafPos, inventory.ItemShears); !harvested || dropped != leaves.RuntimeID {
		t.Fatalf("shears leaf drop=(%d,%v), want (%d,true)", dropped, harvested, leaves.RuntimeID)
	}
}

func TestColoredCushionPlacementCollisionAndDrop(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{77}, Vec3{X: 10.5, Y: 64, Z: 10.5})
	stone := world.BlockPos{X: 11, Y: 63, Z: 10}
	pos := world.BlockPos{X: 11, Y: 64, Z: 10}
	if eng.PlaceBlock(player.ID, pos, inventory.FirstCushionBlock) {
		t.Fatal("cushion should not place without support")
	}
	eng.SetBlock(stone.X, stone.Y, stone.Z, inventory.ItemStone)
	if !eng.PlaceBlock(player.ID, pos, inventory.FirstCushionBlock) {
		t.Fatal("cushion should place on a solid surface")
	}
	if IsSolidBlock(inventory.FirstCushionBlock) || len(BlockCollisionAABBs(inventory.FirstCushionBlock, pos.X, pos.Y, pos.Z)) != 0 {
		t.Fatal("cushion must not collide with players")
	}
	if boxes, ok := inventory.BlockCollisionBoxes(inventory.FirstCushionBlock); !ok || len(boxes) != 1 || boxes[0].MaxY != .25 {
		t.Fatalf("cushion selection/render shape = %+v, %t", boxes, ok)
	}
	if got := eng.BreakBlock(player.ID, pos); got != inventory.FirstCushionBlock {
		t.Fatalf("breaking cushion dropped %d, want %d", got, inventory.FirstCushionBlock)
	}
	if !eng.PlaceBlock(player.ID, pos, inventory.FirstCushionBlock) {
		t.Fatal("cushion should be placeable again on the supported surface")
	}
	if got := eng.BreakBlock(player.ID, stone); got != inventory.ItemCobblestone {
		t.Fatalf("breaking cushion support dropped %d, want cobblestone", got)
	}
	if got := eng.GetBlock(pos.X, pos.Y, pos.Z); got != inventory.ItemAir {
		t.Fatalf("cushion remained after support was broken: block ID %d", got)
	}
	count := 0
	for _, stack := range player.Inventory.Slots {
		if stack.ItemID == inventory.FirstCushionBlock {
			count += stack.Count
		}
	}
	if count != 1 {
		t.Fatalf("cushion inventory count after replacing one cushion and losing support = %d, want 1", count)
	}
}

func TestWoodFenceGatePlacementToggleCollisionAndDrop(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{91}, Vec3{X: 10.5, Y: 64, Z: 10.5})
	itemID := inventory.FirstWoodFenceGateItem + 5
	pos := world.BlockPos{X: 12, Y: 64, Z: 10}
	state, ok := inventory.FenceGatePlacementState(itemID, 90)
	if !ok {
		t.Fatal("could not create fence gate placement state")
	}
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: itemID, Count: 2}
	if !eng.PlaceBlock(player.ID, pos, state) {
		t.Fatal("could not place fence gate")
	}
	if got := eng.GetBlock(pos.X, pos.Y, pos.Z); got != state {
		t.Fatalf("placed gate state = %d, want %d", got, state)
	}
	if got := player.Inventory.Slots[0].Count; got != 1 {
		t.Fatalf("gate stack after placing = %d, want 1", got)
	}
	closedBoxes := BlockCollisionAABBsAt(state, pos.X, pos.Y, pos.Z, eng.GetBlock)
	if len(closedBoxes) != 4 {
		t.Fatalf("closed gate collision boxes = %d, want 4", len(closedBoxes))
	}
	if !eng.SetFenceGateOpen(pos, true) {
		t.Fatal("could not open fence gate")
	}
	openState := eng.GetBlock(pos.X, pos.Y, pos.Z)
	_, _, open, _, valid := inventory.FenceGateStateProperties(openState)
	if !valid || !open {
		t.Fatalf("open gate state %d decoded as open=%v valid=%v", openState, open, valid)
	}
	openBoxes := BlockCollisionAABBsAt(openState, pos.X, pos.Y, pos.Z, eng.GetBlock)
	if len(openBoxes) != 4 || openBoxes[0].Min.Z == closedBoxes[0].Min.Z {
		t.Fatalf("open gate did not rotate its collision model: closed=%+v open=%+v", closedBoxes[0], openBoxes[0])
	}
	if !eng.SetFenceGateOpen(pos, false) {
		t.Fatal("could not close fence gate")
	}
	if got := eng.BreakBlock(player.ID, pos); got != itemID {
		t.Fatalf("breaking gate dropped %d, want %d", got, itemID)
	}
}

func TestWoodDoorAndTrapdoorPlacementToggleAndDrop(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{92}, Vec3{X: 10.5, Y: 64, Z: 10.5})
	doorItem := inventory.FirstWoodDoorItem + 2
	doorPos := world.BlockPos{X: 14, Y: 64, Z: 10}
	eng.SetBlock(doorPos.X, doorPos.Y-1, doorPos.Z, inventory.ItemStone)
	doorState, ok := inventory.WoodDoorPlacementState(doorItem, 90)
	if !ok {
		t.Fatal("could not create wood door placement state")
	}
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: doorItem, Count: 1}
	if !eng.PlaceBlock(player.ID, doorPos, doorState) {
		t.Fatal("could not place two-block wood door")
	}
	lower := eng.GetBlock(doorPos.X, doorPos.Y, doorPos.Z)
	upper := eng.GetBlock(doorPos.X, doorPos.Y+1, doorPos.Z)
	if _, _, _, _, isUpper, valid := inventory.WoodDoorStateProperties(lower); !valid || isUpper {
		t.Fatalf("lower door half state %d is invalid or marked upper", lower)
	}
	if id, _, _, _, isUpper, valid := inventory.WoodDoorStateProperties(upper); !valid || id != doorItem || !isUpper {
		t.Fatalf("upper door half state %d is invalid", upper)
	}
	if !eng.SetWoodDoorOpen(doorPos, true) {
		t.Fatal("could not open paired door")
	}
	for _, state := range []int32{eng.GetBlock(doorPos.X, doorPos.Y, doorPos.Z), eng.GetBlock(doorPos.X, doorPos.Y+1, doorPos.Z)} {
		_, _, open, _, _, valid := inventory.WoodDoorStateProperties(state)
		if !valid || !open {
			t.Fatalf("paired door state %d did not open", state)
		}
	}
	if got := eng.BreakBlock(player.ID, world.BlockPos{X: doorPos.X, Y: doorPos.Y + 1, Z: doorPos.Z}); got != doorItem {
		t.Fatalf("breaking upper door half dropped %d, want %d", got, doorItem)
	}
	if eng.GetBlock(doorPos.X, doorPos.Y, doorPos.Z) != inventory.ItemAir {
		t.Fatal("lower door half remained after breaking the upper half")
	}

	trapdoorItem := inventory.FirstWoodTrapdoorItem + 4
	trapdoorPos := world.BlockPos{X: 16, Y: 64, Z: 10}
	trapdoorState := inventory.WoodTrapdoorBlockState(trapdoorItem, 0, false, false)
	player.Inventory.Slots[1] = inventory.ItemStack{ItemID: trapdoorItem, Count: 1}
	if !eng.PlaceBlock(player.ID, trapdoorPos, trapdoorState) {
		t.Fatal("could not place wood trapdoor")
	}
	if !eng.SetWoodTrapdoorOpen(trapdoorPos, true) {
		t.Fatal("could not open wood trapdoor")
	}
	if _, _, open, _, valid := inventory.WoodTrapdoorStateProperties(eng.GetBlock(trapdoorPos.X, trapdoorPos.Y, trapdoorPos.Z)); !valid || !open {
		t.Fatal("trapdoor open state was not saved")
	}
	if got := eng.BreakBlock(player.ID, trapdoorPos); got != trapdoorItem {
		t.Fatalf("breaking trapdoor dropped %d, want %d", got, trapdoorItem)
	}
}

func TestIronTrapdoorUsesRedstoneControlledState(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{93}, Vec3{X: 10.5, Y: 64, Z: 10.5})
	pos := world.BlockPos{X: 14, Y: 64, Z: 10}
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemIronTrapdoor, Count: 1}
	state := inventory.IronTrapdoorPlacementState(90, -1, .2)
	if !eng.PlaceBlock(player.ID, pos, state) {
		t.Fatal("could not place iron trapdoor")
	}
	if eng.SetIronTrapdoorPowered(pos, false) {
		t.Fatal("unpowered iron trapdoor changed state")
	}
	if !eng.SetIronTrapdoorPowered(pos, true) {
		t.Fatal("powered iron trapdoor did not open")
	}
	if _, open, _, ok := inventory.IronTrapdoorStateProperties(eng.GetBlock(pos.X, pos.Y, pos.Z)); !ok || !open {
		t.Fatal("powered iron trapdoor state is not open")
	}
	if !eng.SetIronTrapdoorPowered(pos, false) {
		t.Fatal("unpowered iron trapdoor did not close")
	}
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemWoodenPickaxe, Count: 1}
	if got := eng.BreakBlock(player.ID, pos); got != inventory.ItemIronTrapdoor {
		t.Fatalf("breaking iron trapdoor dropped %d, want %d", got, inventory.ItemIronTrapdoor)
	}
}

func TestBoneMealGrowsRedShrubAndConsumesOnlyInSurvival(t *testing.T) {
	for _, gameMode := range []int{0, 1} {
		eng := NewEngine()
		player := eng.AddPlayer("Steve", [16]byte{byte(gameMode + 1)}, Vec3{X: 20.5, Y: 64, Z: 20.5})
		player.GameMode = gameMode
		pos := world.BlockPos{X: 10, Y: 64, Z: 10}
		eng.SetBlock(pos.X, pos.Y-1, pos.Z, inventory.ItemStone)
		eng.SetBlock(pos.X, pos.Y, pos.Z, 10100)
		for _, direction := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			eng.SetBlock(pos.X+direction[0], pos.Y-1, pos.Z+direction[1], inventory.ItemStone)
		}
		player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemBoneMeal, Count: 2}

		grown, ok := eng.UseBoneMealOnRedShrub(player.ID, pos)
		if !ok {
			t.Fatalf("game mode %d: bone meal did not grow a supported red shrub", gameMode)
		}
		if got := eng.GetBlock(grown.X, grown.Y, grown.Z); got != 10100 {
			t.Fatalf("game mode %d: grown block ID = %d, want red shrub", gameMode, got)
		}
		wantCount := 1
		if gameMode == 1 {
			wantCount = 2
		}
		if got := player.Inventory.Slots[0].Count; got != wantCount {
			t.Fatalf("game mode %d: bone meal count = %d, want %d", gameMode, got, wantCount)
		}
	}
}

func TestBoneMealDoesNotConsumeWhenRedShrubCannotSpread(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{1}, Vec3{X: 20.5, Y: 64, Z: 20.5})
	pos := world.BlockPos{X: 10, Y: 64, Z: 10}
	eng.SetBlock(pos.X, pos.Y-1, pos.Z, inventory.ItemStone)
	eng.SetBlock(pos.X, pos.Y, pos.Z, 10100)
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemBoneMeal, Count: 2}

	if _, ok := eng.UseBoneMealOnRedShrub(player.ID, pos); ok {
		t.Fatal("red shrub unexpectedly spread without adjacent supporting ground")
	}
	if got := player.Inventory.Slots[0].Count; got != 2 {
		t.Fatalf("failed bone meal use consumed an item: count = %d", got)
	}
}

func TestConcurrentRedShrubBoneMealUseConsumesAndPlacesAtomically(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Steve", [16]byte{2}, Vec3{X: 20.5, Y: 64, Z: 20.5})
	pos := world.BlockPos{X: 10, Y: 64, Z: 10}
	eng.SetBlock(pos.X, pos.Y-1, pos.Z, inventory.ItemStone)
	eng.SetBlock(pos.X, pos.Y, pos.Z, 10100)
	for _, direction := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		eng.SetBlock(pos.X+direction[0], pos.Y-1, pos.Z+direction[1], inventory.ItemStone)
	}
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemBoneMeal, Count: 1}

	start := make(chan struct{})
	results := make(chan bool, 2)
	var workers sync.WaitGroup
	workers.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer workers.Done()
			<-start
			_, grown := eng.UseBoneMealOnRedShrub(player.ID, pos)
			results <- grown
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	successes := 0
	for grown := range results {
		if grown {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("two concurrent bone meal uses succeeded %d times, want once", successes)
	}
	if player.Inventory.Slots[0] != (inventory.ItemStack{}) {
		t.Fatalf("bone meal after concurrent use = %+v, want empty", player.Inventory.Slots[0])
	}
	spreadShrubs := 0
	for _, direction := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if eng.GetBlock(pos.X+direction[0], pos.Y, pos.Z+direction[1]) == 10100 {
			spreadShrubs++
		}
	}
	if spreadShrubs != 1 {
		t.Fatalf("concurrent bone meal placed %d shrubs, want exactly one", spreadShrubs)
	}
}

func TestBlockPlacementAllFacesAndPillaring(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{1}, Vec3{8.5, 65.0, 8.5})
	p.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemOakPlanks, Count: 64}

	// 1. Place on top of ground block (adjacent at 9, 65, 8)
	ok := eng.PlaceBlock(p.ID, world.BlockPos{X: 9, Y: 65, Z: 8}, inventory.ItemOakPlanks)
	if !ok {
		t.Fatalf("failed to place block on adjacent ground face")
	}
	if eng.GetBlock(9, 65, 8) != inventory.ItemOakPlanks {
		t.Fatalf("block was not placed at (9, 65, 8)")
	}

	// 2. Place on +X face of newly placed block (10, 65, 8)
	ok = eng.PlaceBlock(p.ID, world.BlockPos{X: 10, Y: 65, Z: 8}, inventory.ItemCobblestone)
	if !ok {
		t.Fatalf("failed to place block on +X face")
	}

	// 3. Place on +Z face (9, 65, 9)
	ok = eng.PlaceBlock(p.ID, world.BlockPos{X: 9, Y: 65, Z: 9}, inventory.ItemDirt)
	if !ok {
		t.Fatalf("failed to place block on +Z face")
	}

	// 4. Pillaring: player jumps up to Y = 66.2, places block underneath feet at (8, 65, 8)
	p.Position.Y = 66.2
	ok = eng.PlaceBlock(p.ID, world.BlockPos{X: 8, Y: 65, Z: 8}, inventory.ItemStone)
	if !ok {
		t.Fatalf("pillaring failed: should place block underneath feet when jumping")
	}
	if eng.GetBlock(8, 65, 8) != inventory.ItemStone {
		t.Fatalf("pillared block not set at (8, 65, 8)")
	}
}

func TestFallDamageAndDeathRespawn(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{1}, Vec3{0, 64, 0})

	// Damage player by 15 HP
	eng.DamagePlayer(p.ID, 15.0, "attack")
	if p.Health != 5.0 {
		t.Fatalf("health mismatch: %f", p.Health)
	}

	// Fatal damage
	eng.DamagePlayer(p.ID, 10.0, "fatal")
	if p.IsAlive || p.Health != 0 {
		t.Fatalf("player should be dead: alive=%v, health=%f", p.IsAlive, p.Health)
	}

	// Respawn
	eng.RespawnPlayer(p.ID, Vec3{0, 64, 0})
	if !p.IsAlive || p.Health != 20.0 {
		t.Fatalf("player should be respawned with 20 HP: alive=%v, health=%f", p.IsAlive, p.Health)
	}
}

func TestArmorReducesPhysicalDamageButNotFallDamage(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Armored", [16]byte{0xA1}, Vec3{X: 1, Y: 64, Z: 1})
	protection := inventory.ArmorProtection{Defense: 20, Toughness: 8}
	eng.SetPlayerArmorProtection(player.ID, protection)
	eng.DamagePlayer(player.ID, 10, "explosion")
	if player.Health != 17 {
		t.Fatalf("full diamond armor took 10 explosion damage down to %v HP, want 17", player.Health)
	}

	falling := eng.AddPlayer("Falling", [16]byte{0xA2}, Vec3{X: 4, Y: 64, Z: 4})
	eng.SetPlayerArmorProtection(falling.ID, protection)
	eng.DamagePlayer(falling.ID, 10, "fall")
	if falling.Health != 10 {
		t.Fatalf("armor reduced fall damage: health=%v, want 10", falling.Health)
	}
}

func TestZombieAIAndPlayerCombat(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Player", [16]byte{2}, Vec3{X: 10, Y: 64, Z: 10})

	// Spawn Zombie 5 blocks away
	zombie := eng.SpawnMob(MobZombie, Vec3{X: 10, Y: 64, Z: 15})
	if zombie.Health != 20.0 || !zombie.IsAlive {
		t.Fatalf("zombie should spawn with 20 HP")
	}

	// Tick simulation: zombie should detect player and walk towards player (-Z)
	eng.Tick()
	if zombie.Velocity.Z >= 0 {
		t.Fatalf("zombie should move towards player (-Z), got vz=%f", zombie.Velocity.Z)
	}

	// Move zombie directly next to player (dist < 1.5 blocks)
	zombie.Position = Vec3{X: 10, Y: 64, Z: 11}
	zombie.AttackCooldown = 0
	eng.Tick()

	// Zombie should have attacked player for 3.0 damage (20 - 3 = 17)
	if player.Health != 17.0 {
		t.Fatalf("expected player health 17.0 after zombie attack, got %f", player.Health)
	}

	// Player strikes back and kills zombie
	drops, killed := eng.DamageMob(zombie.ID, 20.0)
	if !killed || len(drops) == 0 {
		t.Fatalf("zombie should be killed and drop items: killed=%v, drops=%+v", killed, drops)
	}
	if drops[0].ItemID != 367 { // Rotten flesh
		t.Fatalf("expected rotten flesh drop (367), got %d", drops[0].ItemID)
	}
}

func TestCreeperExplosionDestruction(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Player", [16]byte{3}, Vec3{X: 0, Y: 64, Z: 0})

	// Place dirt blocks around player
	eng.SetBlock(1, 64, 1, 3)
	eng.SetBlock(2, 64, 1, 3)

	// Spawn Creeper 2 blocks away
	creeper := eng.SpawnMob(MobCreeper, Vec3{X: 1, Y: 64, Z: 1})

	// Tick 30 times to let fuse count down to explosion
	for i := 0; i < 30; i++ {
		eng.Tick()
	}

	// Creeper should have exploded
	if creeper.IsAlive {
		t.Fatalf("creeper should have exploded")
	}
	// Blocks in radius should have been blasted to Air (0)
	if eng.GetBlock(1, 64, 1) != 0 {
		t.Fatalf("dirt block at (1, 64, 1) should be destroyed by explosion")
	}
	// Player should have taken blast damage
	if player.Health >= 20.0 {
		t.Fatalf("player should have taken explosion damage, health=%f", player.Health)
	}
}

func TestMovementCollisionNoClipping(t *testing.T) {
	eng := NewEngine()
	// Solid floor at Y=63
	for x := int32(8); x <= 12; x++ {
		for z := int32(8); z <= 12; z++ {
			eng.SetBlock(x, 63, z, 1) // Stone
		}
	}
	// Solid stone wall at X=11 (blocks at (11, 64, 10) and (11, 65, 10))
	eng.SetBlock(11, 64, 10, 1)
	eng.SetBlock(11, 65, 10, 1)

	player := eng.AddPlayer("Steve", [16]byte{5}, Vec3{X: 10.0, Y: 64.0, Z: 10.0})

	// 1. Try to walk into the stone wall by delta X = +2.0
	resPos, _, _ := eng.MovePlayerRelative(player.ID, Vec3{X: 2.0, Y: 0, Z: 0}, 0, 0)

	// Wall is at X=11. Player has half-width 0.3. Max X allowed is 11.0 - 0.3 = 10.7
	if resPos.X > 10.701 {
		t.Fatalf("player clipped into wall! Expected X <= 10.7, got %f", resPos.X)
	}
	if resPos.X < 10.69 {
		t.Fatalf("player stopped too far from wall, got X=%f", resPos.X)
	}

	// 2. Try to fall through solid stone floor (Y=63)
	resPos, ground, _ := eng.MovePlayerRelative(player.ID, Vec3{X: 0, Y: -2.0, Z: 0}, 0, 0)
	if !ground {
		t.Fatalf("player should be flagged onGround after landing on floor")
	}
	// Floor top is Y=63+1 = 64.0
	if math.Abs(resPos.Y-64.0) > 1e-3 {
		t.Fatalf("player feet should rest exactly on floor Y=64.0, got %f", resPos.Y)
	}

	// 3. High-velocity fall from height (e.g. falling 10 blocks in a lag spike)
	highPlayer := eng.AddPlayer("FallenSteve", [16]byte{6}, Vec3{X: 10.0, Y: 75.0, Z: 10.0})
	resHighPos, highGround, _ := eng.MovePlayerRelative(highPlayer.ID, Vec3{X: 0, Y: -15.0, Z: 0}, 0, 0)
	if !highGround {
		t.Fatalf("high velocity fall should be caught by floor and flagged onGround")
	}
	if math.Abs(resHighPos.Y-64.0) > 1e-3 {
		t.Fatalf("high velocity fall should stop exactly at floor Y=64.0, got %f", resHighPos.Y)
	}

	// 4. Sprinting diagonally into wall corner
	cornerPlayer := eng.AddPlayer("CornerSteve", [16]byte{7}, Vec3{X: 10.0, Y: 64.0, Z: 9.0})
	resCornerPos, _, _ := eng.MovePlayerRelative(cornerPlayer.ID, Vec3{X: 2.0, Y: 0, Z: 2.0}, 0, 0)
	// Block at (11, 64, 10) prevents passing (X <= 10.7, Z <= 9.7)
	if resCornerPos.X > 10.701 && resCornerPos.Z > 9.701 {
		t.Fatalf("corner penetration occurred! Pos=%v", resCornerPos)
	}
}

func TestAuthenticMiningAndToolProgression(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Miner", [16]byte{9}, Vec3{X: 0, Y: 64, Z: 0})

	// 1. Bare hand breaking Stone (ID: 1): cannot harvest, yields no item
	eng.SetBlock(1, 64, 0, 1) // Stone
	drop, harvested := eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 1, Y: 64, Z: 0}, inventory.ItemAir)
	if harvested || drop != 0 {
		t.Fatalf("bare hands should NOT harvest stone, got drop=%d, harvested=%v", drop, harvested)
	}

	// 2. Wooden Pickaxe breaking Stone (ID: 1): can harvest, yields Cobblestone (4)
	eng.SetBlock(2, 64, 0, 1) // Stone
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemWoodenPickaxe, Count: 1, Damage: 0}
	player.Inventory.SelectedHotbar = 0

	drop, harvested = eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 2, Y: 64, Z: 0}, inventory.ItemWoodenPickaxe)
	if !harvested || drop != inventory.ItemCobblestone {
		t.Fatalf("wooden pickaxe must harvest stone into cobblestone, got drop=%d, harvested=%v", drop, harvested)
	}
	// Tool durability must decrease by 1
	if player.Inventory.Slots[0].Damage != 1 {
		t.Fatalf("wooden pickaxe should have taken 1 damage, got %d", player.Inventory.Slots[0].Damage)
	}

	// 3. Diamond Ore (56) with Stone Pickaxe (Tier 1): cannot harvest (needs Iron Tier 2)
	eng.SetBlock(3, 64, 0, 56)
	drop, harvested = eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 3, Y: 64, Z: 0}, inventory.ItemStonePickaxe)
	if harvested || drop != 0 {
		t.Fatalf("stone pickaxe cannot harvest diamond ore, got drop=%d, harvested=%v", drop, harvested)
	}

	// 4. Diamond Ore (56) with Iron Pickaxe (Tier 2): yields Diamond (264)
	eng.SetBlock(4, 64, 0, 56)
	drop, harvested = eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 4, Y: 64, Z: 0}, inventory.ItemIronPickaxe)
	if !harvested || drop != inventory.ItemDiamond {
		t.Fatalf("iron pickaxe must harvest diamond ore into diamond, got drop=%d, harvested=%v", drop, harvested)
	}

	// 5. Bedrock (7) cannot be broken
	eng.SetBlock(5, 64, 0, 7)
	drop, harvested = eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 5, Y: 64, Z: 0}, inventory.ItemDiamondPickaxe)
	if harvested || eng.GetBlock(5, 64, 0) != 7 {
		t.Fatalf("bedrock must be unbreakable")
	}

	// 6. Mining break speed calculations
	dpsDirtHand := inventory.CalculateBlockDamagePerSecond(inventory.ItemAir, inventory.ItemDirt)
	// Time = 1.0 / dpsDirtHand ~ 0.75 seconds
	timeDirtHand := 1.0 / dpsDirtHand
	if math.Abs(timeDirtHand-0.75) > 0.05 {
		t.Fatalf("dirt hand break time should be ~0.75s, got %f", timeDirtHand)
	}

	dpsStonePick := inventory.CalculateBlockDamagePerSecond(inventory.ItemWoodenPickaxe, inventory.ItemStone)
	// Time = 1.0 / dpsStonePick ~ 1.125 seconds
	timeStonePick := 1.0 / dpsStonePick
	if math.Abs(timeStonePick-1.125) > 0.05 {
		t.Fatalf("stone wooden pick break time should be ~1.125s, got %f", timeStonePick)
	}
}

func TestDropSpawningAndAutoRespawn(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("Miner", [16]byte{5}, Vec3{X: 8, Y: 64, Z: 8})

	// 1. Mine a dirt block: verify physical ItemDrop entity is spawned
	eng.SetBlock(8, 64, 9, 3) // Dirt
	dropID, ok := eng.BreakBlockWithTool(player.ID, world.BlockPos{X: 8, Y: 64, Z: 9}, inventory.ItemAir)
	if !ok || dropID != inventory.ItemDirt {
		t.Fatalf("expected dirt drop, got %d, %v", dropID, ok)
	}

	drops := eng.GetItemDrops()
	if len(drops) == 0 {
		t.Fatalf("expected 3D ItemDrop entity in world after mining block")
	}
	drop := drops[0]
	if drop.ItemID != inventory.ItemDirt {
		t.Fatalf("expected ItemDrop to be dirt, got %d", drop.ItemID)
	}

	// Fast forward past pickup delay (0.4s = 8 ticks)
	for i := 0; i < 9; i++ {
		eng.Tick()
	}

	// ItemDrop should now be picked up into player inventory
	remainingDrops := eng.GetItemDrops()
	if len(remainingDrops) != 0 {
		t.Fatalf("expected ItemDrop to be collected by player after pickup delay")
	}
	foundDirt := 0
	for _, s := range player.Inventory.Slots {
		if s.ItemID == inventory.ItemDirt {
			foundDirt += s.Count
		}
	}
	if foundDirt != 1 {
		t.Fatalf("expected 1 dirt in inventory, got %d", foundDirt)
	}

	// 2. Fatal damage: test automatic safe respawn prevention of void teleportation
	eng.DamagePlayer(player.ID, 100.0, "explosion")
	if player.IsAlive {
		t.Fatalf("player should be dead after 100 damage")
	}

	// Next client movement must auto-respawn and return valid world position, never {0, 0, 0}
	newPos, ok, _ := eng.MovePlayerRelative(player.ID, Vec3{X: 0, Y: 0, Z: 0}, 0, 0)
	if !ok || newPos.Y < 60.0 {
		t.Fatalf("expected safe auto-respawn position, got %+v, ok=%v", newPos, ok)
	}
	if !player.IsAlive || player.Health != 20.0 {
		t.Fatalf("player should be fully respawned with 20 HP")
	}
}

func TestFoodConsumptionAndRegeneration(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("HungrySteve", [16]byte{9}, Vec3{X: 8, Y: 64, Z: 8})
	player.Hunger = 10
	player.Health = 15.0

	// Give Cooked Porkchop (320)
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemCookedPorkchop, Count: 2}

	// Consume one porkchop (+8 hunger -> 10 + 8 = 18)
	ok, newHunger := eng.ConsumeFood(player.ID, 0)
	if !ok || newHunger != 18 {
		t.Fatalf("expected food to be consumed and hunger restored to 18, got %v, %d", ok, newHunger)
	}
	if player.Inventory.Slots[0].Count != 1 {
		t.Fatalf("expected remaining count 1, got %d", player.Inventory.Slots[0].Count)
	}

	// Run 80 ticks: natural health regeneration should trigger when hunger >= 18 (+1.0 HP)
	for i := 0; i < 80; i++ {
		eng.Tick()
	}

	if player.Health != 16.0 {
		t.Fatalf("expected health to regenerate from 15.0 to 16.0 at tick 80, got %f", player.Health)
	}

	// Eat second porkchop (+8 hunger -> capped at 20)
	ok, newHunger = eng.ConsumeFood(player.ID, 0)
	if !ok || newHunger != 20 {
		t.Fatalf("expected hunger to cap at 20, got %v, %d", ok, newHunger)
	}

	// Stomach is full: further food consumption should be rejected
	player.Inventory.Slots[1] = inventory.ItemStack{ItemID: inventory.ItemApple, Count: 1}
	ok, _ = eng.ConsumeFood(player.ID, 1)
	if ok {
		t.Fatalf("should not be able to eat when hunger is already full at 20")
	}
}

func TestModernFoodRemaindersAndCreativeAppleEating(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("FoodTester", [16]byte{17}, Vec3{X: 8, Y: 64, Z: 8})
	player.Hunger = 10
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemMushroomStew, Count: 1}
	for {
		select {
		case <-eng.Events:
		default:
			goto drained
		}
	}
drained:
	if ok, hunger := eng.ConsumeFood(player.ID, 0); !ok || hunger != 16 {
		t.Fatalf("mushroom stew consume = (%v, %d), want (true, 16)", ok, hunger)
	}
	if got := player.Inventory.Slots[0]; got.ItemID != 281 || got.Count != 1 {
		t.Fatalf("consuming the last stew should leave its bowl in the slot, got %+v", got)
	}
	select {
	case event := <-eng.Events:
		if event.Type != "player_eat" || event.BlockID != inventory.ItemMushroomStew {
			t.Fatalf("food event after consuming final item = %+v, want original stew ID", event)
		}
	default:
		t.Fatal("food consumption did not emit an event")
	}

	player.Hunger = 20
	player.Inventory.Slots[1] = inventory.ItemStack{ItemID: inventory.ItemEnchantedGoldenApple, Count: 1}
	if ok, hunger := eng.ConsumeFood(player.ID, 1); !ok || hunger != 20 {
		t.Fatalf("enchanted golden apple at full hunger = (%v, %d), want (true, 20)", ok, hunger)
	}
	if got := player.Inventory.Slots[1]; !got.IsEmpty() {
		t.Fatalf("enchanted golden apple was not consumed at full hunger: %+v", got)
	}
	player.Inventory.Slots[2] = inventory.ItemStack{ItemID: inventory.ItemHoneyBottle, Count: 1}
	if ok, hunger := eng.ConsumeFood(player.ID, 2); !ok || hunger != 20 {
		t.Fatalf("honey bottle at full hunger = (%v, %d), want (true, 20)", ok, hunger)
	}
	if got := player.Inventory.Slots[2]; got.ItemID != 374 || got.Count != 1 {
		t.Fatalf("honey bottle did not return a glass bottle: %+v", got)
	}
}

func TestModernFoodStatusEffectsAndMilkClear(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("EffectTester", [16]byte{18}, Vec3{X: 8, Y: 64, Z: 8})
	player.Health = 10
	player.Hunger = 10
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemGoldenApple, Count: 1}
	if ok, _ := eng.ConsumeFood(player.ID, 0); !ok {
		t.Fatal("golden apple was not consumed")
	}
	if player.Absorption != 4 || activeEffectIndex(player, inventory.EffectRegeneration) < 0 {
		t.Fatalf("golden apple effects missing: absorption=%v effects=%+v", player.Absorption, player.ActiveEffects[:player.EffectCount])
	}
	for range 25 {
		eng.Tick()
	}
	if player.Health != 11 {
		t.Fatalf("Regeneration II did not heal at the vanilla 25-tick interval: health=%v", player.Health)
	}

	player.Inventory.Slots[1] = inventory.ItemStack{ItemID: inventory.ItemSpiderEye, Count: 1}
	if ok, _ := eng.ConsumeFood(player.ID, 1); !ok || activeEffectIndex(player, inventory.EffectPoison) < 0 {
		t.Fatal("spider eye did not apply poison")
	}
	player.Inventory.Slots[2] = inventory.ItemStack{ItemID: inventory.ItemHoneyBottle, Count: 1}
	if ok, _ := eng.ConsumeFood(player.ID, 2); !ok || activeEffectIndex(player, inventory.EffectPoison) >= 0 {
		t.Fatal("honey bottle did not remove poison")
	}
	if got := player.Inventory.Slots[2]; got.ItemID != 374 || got.Count != 1 {
		t.Fatalf("honey bottle remainder = %+v, want glass bottle", got)
	}

	player.Inventory.Slots[3] = inventory.ItemStack{ItemID: inventory.ItemMilkBucket, Count: 1}
	if ok, _ := eng.ConsumeFood(player.ID, 3); !ok || player.EffectCount != 0 || player.Absorption != 0 {
		t.Fatalf("milk did not clear active effects: ok=%v absorption=%v effects=%+v", ok, player.Absorption, player.ActiveEffects[:player.EffectCount])
	}
	if got := player.Inventory.Slots[3]; got.ItemID != 325 || got.Count != 1 {
		t.Fatalf("milk bucket remainder = %+v, want empty bucket", got)
	}
}

func TestMobWanderingAndObstacleJump(t *testing.T) {
	eng := NewEngine()
	// Place flat floor at Y=64, with 1-block step at X=6, Y=65, Z=5
	for x := int32(0); x <= 10; x++ {
		for z := int32(0); z <= 10; z++ {
			eng.SetBlock(x, 64, z, 1) // Stone floor
		}
	}
	eng.SetBlock(6, 65, 5, 1) // 1-block elevation step obstacle

	cow := eng.SpawnMob(MobCow, Vec3{X: 5.5, Y: 65.0, Z: 5.5})
	cow.IsIdle = false
	cow.WanderTicks = 50
	cow.TargetYaw = 90.0 // heading in +X direction towards the step at X=6
	cow.Yaw = 90.0

	// Tick the cow simulation
	eng.Tick()

	// Since block at X=6, Y=65 is solid stone and Y=66 is air, cow must trigger 1-block jump impulse
	if cow.Velocity.Y <= 0 {
		t.Fatalf("expected mob to trigger 1-block jump over obstacle, got Y velocity: %f", cow.Velocity.Y)
	}

	if cow.WalkDistance <= 0 {
		t.Fatalf("expected mob to accumulate walk distance for leg swing, got: %f", cow.WalkDistance)
	}
}

func TestSheepAndChickenDrops(t *testing.T) {
	eng := NewEngine()
	sheep := eng.SpawnMob(MobSheep, Vec3{X: 10, Y: 65, Z: 10})
	chicken := eng.SpawnMob(MobChicken, Vec3{X: 12, Y: 65, Z: 12})

	dropsSheep, killedSheep := eng.DamageMob(sheep.ID, 20.0)
	if !killedSheep || len(dropsSheep) == 0 {
		t.Fatalf("expected sheep to be killed and drop wool and mutton, got %v", dropsSheep)
	}
	hasWool := false
	for _, d := range dropsSheep {
		if d.ItemID == 35 { // White wool
			hasWool = true
		}
	}
	if !hasWool {
		t.Fatalf("expected sheep drops to include White Wool (ID 35)")
	}

	dropsChicken, killedChicken := eng.DamageMob(chicken.ID, 10.0)
	if !killedChicken || len(dropsChicken) == 0 {
		t.Fatalf("expected chicken to be killed and drop feathers, got %v", dropsChicken)
	}
	hasFeather := false
	for _, d := range dropsChicken {
		if d.ItemID == 288 { // Feather
			hasFeather = true
		}
	}
	if !hasFeather {
		t.Fatalf("expected chicken drops to include Feathers (ID 288)")
	}
}

func TestItemPickupFullInventoryAndPartial(t *testing.T) {
	eng := NewEngine()
	player := eng.AddPlayer("TestSteve", [16]byte{1}, Vec3{X: 10, Y: 65, Z: 10})

	// Fill all 36 slots completely with stone (stack of 64 each)
	for i := 0; i < len(player.Inventory.Slots); i++ {
		player.Inventory.Slots[i] = inventory.ItemStack{ItemID: inventory.ItemStone, Count: 64}
	}

	// Spawn a diamond drop near the player (within 1.5 blocks) with 0 pickup delay
	drop := eng.SpawnItemDropWithDelay(inventory.ItemDiamond, 5, Vec3{X: 10.5, Y: 65, Z: 10.5}, Vec3{}, 0.0)

	// Step simulation ticks
	for i := 0; i < 5; i++ {
		eng.Tick()
	}

	// Because inventory was completely full of stone, diamond drop must NOT be swallowed or deleted!
	eng.mu.RLock()
	d, exists := eng.ItemDrops[drop.ID]
	eng.mu.RUnlock()
	if !exists || d == nil {
		t.Fatalf("ground item drop was swallowed when inventory was full!")
	}
	if d.Count != 5 {
		t.Fatalf("expected ground drop count to remain 5, got %d", d.Count)
	}

	// Free up slot 0 but put 62 stone in it (so it has room for only 2 more stone)
	player.Inventory.Slots[0] = inventory.ItemStack{ItemID: inventory.ItemStone, Count: 62}

	// Spawn a stone drop with 10 stone
	stoneDrop := eng.SpawnItemDropWithDelay(inventory.ItemStone, 10, Vec3{X: 10.2, Y: 65, Z: 10.2}, Vec3{}, 0.0)
	for i := 0; i < 5; i++ {
		eng.Tick()
	}

	// Slot 0 should now have 64 stone (+2 picked up)
	if player.Inventory.Slots[0].Count != 64 {
		t.Fatalf("expected slot 0 to fill to 64, got %d", player.Inventory.Slots[0].Count)
	}

	// Ground drop must have 8 remaining stone (10 - 2)
	eng.mu.RLock()
	sd, exists := eng.ItemDrops[stoneDrop.ID]
	eng.mu.RUnlock()
	if !exists || sd == nil {
		t.Fatalf("partial drop was completely deleted!")
	}
	if sd.Count != 8 {
		t.Fatalf("expected 8 stone remaining on ground, got %d", sd.Count)
	}
}

func TestEngineChunkPersistenceAndUnloadSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sim_save_test")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sm, err := world.NewSaveManager(tmpDir)
	if err != nil {
		t.Fatalf("NewSaveManager failed: %v", err)
	}
	defer sm.Close()

	eng := NewEngine()
	eng.SetStorage(sm)

	// Set a block (Diamond Block ID=57) at (5, 65, 5) which is in chunk (0, 0)
	eng.SetBlock(5, 65, 5, 57)
	if b := eng.GetBlock(5, 65, 5); b != 57 {
		t.Fatalf("expected block 57, got %d", b)
	}

	// Unload the chunk (simulating player walking away)
	cp := world.ChunkPos{X: 0, Z: 0}
	eng.UnloadChunk(cp)

	// Ensure chunk is no longer in memory
	eng.mu.RLock()
	_, inMem := eng.World[cp]
	eng.mu.RUnlock()
	if inMem {
		t.Fatalf("chunk should have been evicted from memory")
	}

	// Read block back: GetBlock should auto-load the chunk from region file!
	loadedBlock := eng.GetBlock(5, 65, 5)
	if loadedBlock != 57 {
		t.Fatalf("expected loaded block to be 57, got %d", loadedBlock)
	}

	// Test cross-process restart: create brand new Engine with the same save directory
	eng2 := NewEngine()
	eng2.SetStorage(sm)
	restartBlock := eng2.GetBlock(5, 65, 5)
	if restartBlock != 57 {
		t.Fatalf("expected block across engine restart to be 57, got %d", restartBlock)
	}
}

func TestMobHurtFlashDecay(t *testing.T) {
	eng := NewEngine()
	zombie := eng.SpawnMob(MobZombie, Vec3{X: 10, Y: 65, Z: 10})
	if zombie.HurtTime != 0 {
		t.Fatalf("new mob should start with HurtTime 0, got %d", zombie.HurtTime)
	}

	// Player attacks the zombie with 3.0 damage
	eng.DamageMob(zombie.ID, 3.0)
	if zombie.HurtTime != 10 {
		t.Fatalf("mob should have HurtTime 10 after taking damage, got %d", zombie.HurtTime)
	}
	if zombie.Health != 17.0 {
		t.Fatalf("zombie health should be 17.0, got %f", zombie.Health)
	}

	// Tick the engine for 5 ticks
	for i := 0; i < 5; i++ {
		eng.Tick()
	}
	if zombie.HurtTime != 5 {
		t.Fatalf("mob HurtTime should decay to 5 after 5 ticks, got %d", zombie.HurtTime)
	}

	// Tick 5 more ticks (total 10 ticks = 0.5s)
	for i := 0; i < 5; i++ {
		eng.Tick()
	}
	if zombie.HurtTime != 0 {
		t.Fatalf("mob HurtTime should decay to 0 and not stay red indefinitely, got %d", zombie.HurtTime)
	}

	// Test AttackMobInFront
	camPos := Vec3{X: zombie.Position.X - 1.5, Y: zombie.Position.Y + 0.9, Z: zombie.Position.Z}
	look := Vec3{X: 1, Y: 0, Z: 0}
	mobID, killed, hit, _ := eng.AttackMobInFront(camPos, look, 3.0, 4.0)
	if !hit || killed || mobID != zombie.ID {
		t.Fatalf("expected hit on zombie, got hit=%v killed=%v mobID=%d", hit, killed, mobID)
	}
	if zombie.HurtTime != 10 {
		t.Fatalf("mob should reset HurtTime to 10 on new attack, got %d", zombie.HurtTime)
	}

	// Step 10 ticks again
	for i := 0; i < 10; i++ {
		eng.Tick()
	}
	if zombie.HurtTime != 0 {
		t.Fatalf("mob HurtTime should decay back to 0, got %d", zombie.HurtTime)
	}
}

func TestHostileMobLineOfSightThroughWall(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{1}, Vec3{X: 10, Y: 64, Z: 10})
	p.Health = 20.0

	// Build a 3-high solid stone wall between player and skeleton
	eng.SetBlock(10, 64, 11, 1) // Stone
	eng.SetBlock(10, 65, 11, 1) // Stone
	eng.SetBlock(10, 66, 11, 1) // Stone

	// Spawn skeleton on other side of wall
	skeleton := eng.SpawnMob(MobSkeleton, Vec3{X: 10, Y: 64, Z: 12})
	skeleton.AttackCooldown = 0

	// Tick AI - Skeleton should NOT be able to damage player through solid wall!
	skeleton.tickSkeletonAI(eng, p, 2.0)
	if p.Health < 20.0 {
		t.Fatalf("player took damage through solid wall from skeleton: health = %f", p.Health)
	}

	// Remove wall - now line of sight is clear
	eng.SetBlock(10, 64, 11, 0)
	eng.SetBlock(10, 65, 11, 0)
	eng.SetBlock(10, 66, 11, 0)

	skeleton.AttackCooldown = 0
	skeleton.tickSkeletonAI(eng, p, 2.0)
	if p.Health >= 20.0 {
		t.Fatalf("player should have taken damage after wall removed, health = %f", p.Health)
	}
}

func TestFallDamageResetInWater(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{2}, Vec3{X: 5, Y: 70, Z: 5})
	p.Health = 20.0
	p.FallDistance = 8.0 // Accumulated high fall

	// Place water block at player destination
	eng.SetBlock(5, 64, 5, 9) // Water block
	eng.SetBlock(5, 63, 5, 1) // Riverbed stone floor

	// Step down into water
	_, ground, _ := eng.MovePlayerRelative(p.ID, Vec3{X: 0, Y: -6.0, Z: 0}, 0, 0)
	_ = ground

	// Fall distance should be completely reset in water!
	if p.FallDistance != 0 {
		t.Fatalf("fall distance in water should be 0, got %f", p.FallDistance)
	}
	if p.Health != 20.0 {
		t.Fatalf("player took fall damage entering water: health = %f", p.Health)
	}
}

func TestVanillaHungerSaturationAndFastRegen(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{3}, Vec3{X: 0, Y: 64, Z: 0})
	p.Health = 15.0
	p.Hunger = 20
	p.Saturation = 5.0
	p.Exhaustion = 0

	// In 10 ticks, fast regen should restore 1 HP and consume saturation via exhaustion
	for i := 0; i < 10; i++ {
		eng.Tick()
	}

	if p.Health != 16.0 {
		t.Fatalf("fast natural regen expected health 16.0, got %f", p.Health)
	}

	// Step another 50 ticks to drain exhaustion
	for i := 0; i < 50; i++ {
		eng.Tick()
	}
	if p.Saturation >= 5.0 {
		t.Fatalf("expected saturation to decrease from fast regen exhaustion, got %f", p.Saturation)
	}
}

func TestEndermanStareAggroAndTeleport(t *testing.T) {
	eng := NewEngine()
	p := eng.AddPlayer("Steve", [16]byte{4}, Vec3{X: 10, Y: 64, Z: 10})
	p.Yaw = 0   // looking south (+Z)
	p.Pitch = 0 // looking horizontal

	// Spawn Enderman directly in front along +Z
	enderman := eng.SpawnMob(MobEnderman, Vec3{X: 10, Y: 64, Z: 18})
	if enderman.IsAggro {
		t.Fatalf("enderman should start unprovoked")
	}

	// Tick mob AI: player is staring directly at enderman
	enderman.TickMob(eng)
	if !enderman.IsAggro {
		t.Fatalf("enderman should become aggressive when stared at")
	}

	// Test drops
	drops := enderman.DeathDrops()
	if len(drops) == 0 || drops[0].ItemID != 368 {
		t.Fatalf("expected ender pearl drop from enderman, got %v", drops)
	}
}

func TestEndDimensionBossAndCrystalExplosion(t *testing.T) {
	eng := NewEngine()
	eng.SwitchDimension(2) // The End

	if eng.GetDimension() != 2 {
		t.Fatalf("expected active dimension 2 (The End)")
	}

	dragon := eng.GetEnderDragon()
	if dragon == nil {
		t.Fatalf("expected Ender Dragon boss spawned in The End")
	}
	if dragon.Health != 200.0 {
		t.Fatalf("expected 200.0 HP for Ender Dragon, got %f", dragon.Health)
	}

	crystals := eng.GetEndCrystals()
	if len(crystals) != 8 {
		t.Fatalf("expected 8 End Crystals on towers, got %d", len(crystals))
	}

	// Tether dragon to crystal 1 and explode crystal
	eng.SetDragonHealingCrystal(crystals[0].ID)
	eng.AttackEndCrystal(crystals[0].ID)

	dragonAfter := eng.GetEnderDragon()
	if dragonAfter.Health >= 200.0 {
		t.Fatalf("exploding tethered End Crystal should damage dragon, got health %f", dragonAfter.Health)
	}
}

func TestEnderDragonDeathSettlementNoDeadlock(t *testing.T) {
	eng := NewEngine()
	eng.SwitchDimension(2) // The End

	dragon := eng.GetEnderDragon()
	if dragon == nil {
		t.Fatalf("expected Ender Dragon boss spawned in The End")
	}

	// Deal fatal damage
	killed := eng.DamageEnderDragon(200.0)
	if !killed {
		t.Fatalf("expected DamageEnderDragon to trigger death state")
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 110; i++ {
			eng.Tick()
		}
		close(done)
	}()

	select {
	case <-done:
		// Completed 110 ticks successfully without deadlocking!
	case <-time.After(2 * time.Second):
		t.Fatalf("FATAL: Engine.Tick deadlocked during Ender Dragon death settlement!")
	}

	finalDragon := eng.GetEnderDragon()
	if finalDragon.IsAlive || finalDragon.IsDying {
		t.Fatalf("expected dragon to be fully defeated after 110 death ticks, got alive=%v dying=%v", finalDragon.IsAlive, finalDragon.IsDying)
	}

	drops := eng.GetItemDrops()
	if len(drops) == 0 {
		t.Fatalf("expected celebration drops spawned after dragon death")
	}
}

func TestDimensionSwitchRetainsModifiedBlocks(t *testing.T) {
	eng := NewEngine()
	// 1. Overworld block placement
	eng.SetBlock(-85, 110, 25, 57) // Diamond block
	if b := eng.GetBlock(-85, 110, 25); b != 57 {
		t.Fatalf("expected diamond block 57 in Overworld, got %d", b)
	}

	// 2. Switch to Nether
	eng.SwitchDimension(1)
	if b := eng.GetBlock(-85, 110, 25); b == 57 {
		t.Fatalf("overworld block should not be present at nether coordinates")
	}
	eng.SetBlock(10, 66, 10, 89) // Glowstone
	if b := eng.GetBlock(10, 66, 10); b != 89 {
		t.Fatalf("expected glowstone 89 in Nether, got %d", b)
	}

	// 3. Switch back to Overworld
	eng.SwitchDimension(0)
	if b := eng.GetBlock(-85, 110, 25); b != 57 {
		t.Fatalf("re-entering Overworld lost placed block! Expected 57, got %d", b)
	}

	// 4. Switch back to Nether
	eng.SwitchDimension(1)
	if b := eng.GetBlock(10, 66, 10); b != 89 {
		t.Fatalf("re-entering Nether lost placed block! Expected 89, got %d", b)
	}
}

func TestFurnaceSmeltingAndDrop(t *testing.T) {
	eng := NewEngine()
	furnacePos := world.BlockPos{X: 10, Y: 64, Z: 10}

	// 1. Place Furnace (ID 61)
	eng.PlaceBlock(1, furnacePos, 61)
	if b := eng.GetBlock(10, 64, 10); b != 61 {
		t.Fatalf("expected furnace block at (10, 64, 10), got %d", b)
	}

	furnace := eng.GetOrCreateFurnace(furnacePos)
	if furnace == nil {
		t.Fatalf("expected furnace entity created")
	}

	// 2. Add Coal (263) and Iron Ore (15)
	furnace.FuelSlot = inventory.ItemStack{ItemID: inventory.ItemCoal, Count: 1}
	furnace.InputSlot = inventory.ItemStack{ItemID: 15, Count: 1} // Iron Ore

	// 3. Run ticks until smelting completes (200 ticks = 1 item)
	for i := 0; i < 205; i++ {
		eng.Tick()
	}

	// Verify OutputSlot has Iron Ingot (ID 265)
	if furnace.OutputSlot.ItemID != inventory.ItemIronIngot || furnace.OutputSlot.Count != 1 {
		t.Fatalf("expected 1 Iron Ingot smelted in furnace, got ItemID=%d Count=%d", furnace.OutputSlot.ItemID, furnace.OutputSlot.Count)
	}

	// 4. Break furnace with pickaxe (tool ID 270: Wooden Pickaxe)
	dropItem, ok := eng.BreakBlockWithTool(1, furnacePos, 270)
	if !ok || dropItem != 61 {
		t.Fatalf("expected furnace block to drop when mined, got drop=%d, ok=%v", dropItem, ok)
	}

	// Verify furnace dropped its smelted iron ingot
	drops := eng.GetItemDrops()
	foundIngot := false
	for _, d := range drops {
		if d.ItemID == inventory.ItemIronIngot {
			foundIngot = true
			break
		}
	}
	if !foundIngot {
		t.Fatalf("expected furnace contents (iron ingot) dropped when broken")
	}
}

func TestRedstoneControlPlacementAndButtonReset(t *testing.T) {
	eng := NewEngine()
	platePos := world.BlockPos{X: 10, Y: 64, Z: 10}
	if !eng.SetBlockChecked(platePos.X, platePos.Y-1, platePos.Z, inventory.ItemStone) {
		t.Fatal("could not place pressure plate support")
	}
	if !eng.PlaceBlock(0, platePos, inventory.ItemOakPressurePlate) {
		t.Fatal("supported oak pressure plate placement failed")
	}
	if got := eng.GetBlock(platePos.X, platePos.Y, platePos.Z); got != inventory.PressurePlateBlockState(inventory.ItemOakPressurePlate, false) {
		t.Fatalf("placed pressure plate state = %d", got)
	}
	if got := BlockCollisionAABBs(eng.GetBlock(platePos.X, platePos.Y, platePos.Z), platePos.X, platePos.Y, platePos.Z); len(got) != 1 || got[0].Max.Y-got[0].Min.Y > .07 {
		t.Fatalf("pressure plate collision = %+v", got)
	}
	player := eng.AddPlayer("plate-tester", [16]byte{}, Vec3{X: 10.5, Y: 64.0625, Z: 10.5})
	eng.Tick()
	if _, powered, ok := inventory.PressurePlateStateProperties(eng.GetBlock(platePos.X, platePos.Y, platePos.Z)); !ok || !powered {
		t.Fatal("player standing on pressure plate did not activate it")
	}
	player.Position.X = 14.5
	eng.Tick()
	if _, powered, ok := inventory.PressurePlateStateProperties(eng.GetBlock(platePos.X, platePos.Y, platePos.Z)); !ok || powered {
		t.Fatal("pressure plate did not reset after player left")
	}

	buttonPos := world.BlockPos{X: 11, Y: 64, Z: 10}
	buttonState := inventory.ButtonBlockState(inventory.ItemOakButton, 5, false)
	if !eng.PlaceBlock(0, buttonPos, buttonState) {
		t.Fatal("button with west-side support failed to place")
	}
	if eng.PlaceBlock(0, world.BlockPos{X: 13, Y: 64, Z: 10}, inventory.ButtonBlockState(inventory.ItemStoneButton, 5, false)) {
		t.Fatal("unsupported button placement succeeded")
	}
	if !eng.PressButton(buttonPos) {
		t.Fatal("button did not activate")
	}
	if _, _, powered, ok := inventory.ButtonStateProperties(eng.GetBlock(buttonPos.X, buttonPos.Y, buttonPos.Z)); !ok || !powered {
		t.Fatal("button did not enter powered state")
	}
	for range 9 {
		eng.Tick()
	}
	if _, _, powered, _ := inventory.ButtonStateProperties(eng.GetBlock(buttonPos.X, buttonPos.Y, buttonPos.Z)); !powered {
		t.Fatal("button released before ten ticks")
	}
	eng.Tick()
	if _, _, powered, ok := inventory.ButtonStateProperties(eng.GetBlock(buttonPos.X, buttonPos.Y, buttonPos.Z)); !ok || powered {
		t.Fatal("button did not reset after ten ticks")
	}
}
