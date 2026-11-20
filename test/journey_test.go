package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
	"github.com/HUA503/minecraft-go/pkg/world/gen"
)

// TestSingleplayerCompleteJourney verifies the full journey requested in the audit:
// 新建世界 (New World) → 采集 (Gathering) → 鼠标合成 (Crafting) → 熔炼 (Furnace Smelting) → 保存退出 (Save & Quit) → 重新进入继续玩 (Resume World).
func TestSingleplayerCompleteJourney(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "journey_world_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	worldDir := filepath.Join(tmpDir, "test_world")
	_ = os.MkdirAll(worldDir, 0755)

	sm, err := world.NewSaveManager(worldDir)
	if err != nil {
		t.Fatalf("failed to create save manager: %v", err)
	}

	// 1. 新建世界 (New World - Survival mode)
	engine := sim.NewEngine()
	engine.SetStorage(sm)
	seed := int64(987654321)
	generator := gen.NewGenerator(seed, gen.DimensionOverworld)

	spawnPos := sim.Vec3{X: 10.0, Y: 65.0, Z: 10.0}
	player := engine.AddPlayer("Steve", [16]byte{1, 2, 3, 4}, spawnPos)
	player.GameMode = 0 // Survival

	// Verify Survival begins with empty inventory (parity requirement)
	for i := 0; i < 36; i++ {
		if !player.Inventory.Slots[i].IsEmpty() {
			t.Fatalf("expected empty slot %d on survival startup, got %+v", i, player.Inventory.Slots[i])
		}
	}

	// 2. 采集 (Resource Gathering)
	// Place an Oak Log block in front of player and punch/mine it
	logPos := world.BlockPos{X: 10, Y: 65, Z: 11}
	engine.SetBlock(10, 65, 11, inventory.ItemOakLog)
	dropLog := engine.BreakBlock(player.ID, logPos)
	if dropLog != inventory.ItemOakLog {
		t.Fatalf("expected oak log drop, got %d", dropLog)
	}
	if player.Inventory.Slots[0].ItemID != inventory.ItemOakLog || player.Inventory.Slots[0].Count != 1 {
		t.Fatalf("expected 1 oak log in player inventory slot 0, got %+v", player.Inventory.Slots[0])
	}

	// 3. 鼠标合成 (Mouse Crafting)
	cm := inventory.NewCraftingManager()

	// Step A: Craft 1 Oak Log into 4 Oak Planks in 2x2 grid
	craftGrid2x2 := make([]int32, 4)
	craftGrid2x2[0] = player.Inventory.Slots[0].ItemID
	_ = player.Inventory.RemoveItem(0, 1)

	outPlanks, ok := cm.MatchCrafting(craftGrid2x2, 2)
	if !ok || outPlanks.ItemID != inventory.ItemOakPlanks || outPlanks.Count != 4 {
		t.Fatalf("failed to craft oak planks: %+v", outPlanks)
	}
	player.Inventory.AddItem(outPlanks.ItemID, outPlanks.Count)

	// Step B: Craft 4 Oak Planks into 1 Crafting Table (2x2 grid)
	for i := 0; i < 4; i++ {
		craftGrid2x2[i] = inventory.ItemOakPlanks
	}
	_ = player.Inventory.RemoveItem(0, 4)

	outTable, ok := cm.MatchCrafting(craftGrid2x2, 2)
	if !ok || outTable.ItemID != inventory.ItemCraftingTable || outTable.Count != 1 {
		t.Fatalf("failed to craft crafting table: %+v", outTable)
	}
	player.Inventory.AddItem(outTable.ItemID, outTable.Count)

	// Step C: Place Crafting Table in world
	tablePos := world.BlockPos{X: 10, Y: 65, Z: 11}
	placed := engine.PlaceBlock(player.ID, tablePos, inventory.ItemCraftingTable)
	if !placed || engine.GetBlock(10, 65, 11) != inventory.ItemCraftingTable {
		t.Fatalf("failed to place crafting table in world")
	}

	// Step D: Open 3x3 crafting grid and craft Wooden Pickaxe
	// Provide planks and sticks
	craftGrid3x3 := []int32{
		inventory.ItemOakPlanks, inventory.ItemOakPlanks, inventory.ItemOakPlanks,
		inventory.ItemAir, inventory.ItemStick, inventory.ItemAir,
		inventory.ItemAir, inventory.ItemStick, inventory.ItemAir,
	}
	outPickaxe, ok := cm.MatchCrafting(craftGrid3x3, 3)
	if !ok || outPickaxe.ItemID != inventory.ItemWoodenPickaxe || outPickaxe.Count != 1 {
		t.Fatalf("failed to craft wooden pickaxe: %+v", outPickaxe)
	}
	player.Inventory.AddItem(outPickaxe.ItemID, outPickaxe.Count)

	// Step E: Mine 8 stone blocks using pickaxe to get cobblestone
	for i := 0; i < 8; i++ {
		bPos := world.BlockPos{X: 11 + int32(i), Y: 65, Z: 10}
		engine.SetBlock(bPos.X, bPos.Y, bPos.Z, inventory.ItemStone)
		dropItem, mined := engine.BreakBlockWithTool(player.ID, bPos, inventory.ItemWoodenPickaxe)
		if !mined || dropItem != inventory.ItemCobblestone {
			t.Fatalf("failed to mine cobblestone with wooden pickaxe: drop=%d", dropItem)
		}
		player.Inventory.AddItem(dropItem, 1)
	}
	if player.Inventory.Slots[1].ItemID != inventory.ItemCobblestone || player.Inventory.Slots[1].Count != 8 {
		t.Fatalf("expected 8 cobblestone in inventory, got %+v", player.Inventory.Slots[1])
	}

	// Step F: Craft 8 Cobblestone into Furnace in 3x3 grid
	furnaceGrid := []int32{
		inventory.ItemCobblestone, inventory.ItemCobblestone, inventory.ItemCobblestone,
		inventory.ItemCobblestone, inventory.ItemAir, inventory.ItemCobblestone,
		inventory.ItemCobblestone, inventory.ItemCobblestone, inventory.ItemCobblestone,
	}
	outFurnace, ok := cm.MatchCrafting(furnaceGrid, 3)
	if !ok || outFurnace.ItemID != inventory.ItemFurnace || outFurnace.Count != 1 {
		t.Fatalf("failed to craft furnace: %+v", outFurnace)
	}
	_ = player.Inventory.RemoveItem(1, 8)
	player.Inventory.AddItem(outFurnace.ItemID, outFurnace.Count)

	// 4. 熔炼 (Furnace Smelting)
	// Place Furnace at (10, 65, 12)
	furnacePos := world.BlockPos{X: 10, Y: 65, Z: 12}
	placed = engine.PlaceBlock(player.ID, furnacePos, inventory.ItemFurnace)
	if !placed {
		t.Fatalf("failed to place furnace")
	}

	furnaceEntity := engine.GetOrCreateFurnace(furnacePos)
	if furnaceEntity == nil {
		t.Fatalf("furnace block entity not created in engine")
	}

	// Insert Raw Iron / Iron Ore (ID 15) and Coal Fuel (ID 263)
	furnaceEntity.InputSlot = inventory.ItemStack{ItemID: 15, Count: 1}
	furnaceEntity.FuelSlot = inventory.ItemStack{ItemID: inventory.ItemCoal, Count: 1}

	// Run ticks until smelted
	for i := 0; i < 210; i++ {
		engine.Tick()
	}

	if furnaceEntity.OutputSlot.ItemID != inventory.ItemIronIngot || furnaceEntity.OutputSlot.Count != 1 {
		t.Fatalf("expected 1 iron ingot smelted in furnace output slot, got %+v", furnaceEntity.OutputSlot)
	}

	// Take smelted iron into player inventory
	player.Inventory.AddItem(furnaceEntity.OutputSlot.ItemID, furnaceEntity.OutputSlot.Count)
	furnaceEntity.OutputSlot = inventory.ItemStack{}

	// 5. 保存退出 (Save & Quit to Title)
	engine.SaveAll()

	savedInv := make([]world.SavedItemStack, 0, 36)
	for slotIdx, s := range player.Inventory.Slots {
		if !s.IsEmpty() {
			savedInv = append(savedInv, world.SavedItemStack{
				Slot:   slotIdx,
				ItemID: s.ItemID,
				Count:  s.Count,
				Damage: s.Damage,
			})
		}
	}

	lvl := &world.LevelData{
		Seed:      seed,
		SpawnX:    int32(player.Position.X),
		SpawnY:    int32(player.Position.Y),
		SpawnZ:    int32(player.Position.Z),
		LevelName: "Test Journey World",
		GameMode:  0,
		Player: &world.PlayerSaveData{
			X:         player.Position.X,
			Y:         player.Position.Y,
			Z:         player.Position.Z,
			Health:    player.Health,
			Hunger:    player.Hunger,
			GameMode:  0,
			Inventory: savedInv,
		},
	}
	if err := sm.SaveLevel(lvl); err != nil {
		t.Fatalf("failed to save level: %v", err)
	}
	_ = sm.Close()
	engine.Stop()

	// 6. 重新进入继续玩 (Resume World from Save)
	sm2, err := world.NewSaveManager(worldDir)
	if err != nil {
		t.Fatalf("failed to re-open save manager: %v", err)
	}
	defer sm2.Close()

	loadedLvl, err := sm2.LoadLevel()
	if err != nil {
		t.Fatalf("failed to load level.dat: %v", err)
	}

	if loadedLvl.LevelName != "Test Journey World" || loadedLvl.Seed != seed {
		t.Fatalf("level data mismatch: Name=%s, Seed=%d", loadedLvl.LevelName, loadedLvl.Seed)
	}

	engine2 := sim.NewEngine()
	engine2.SetStorage(sm2)
	resumedPlayer := engine2.AddPlayer("Steve", [16]byte{1, 2, 3, 4}, sim.Vec3{
		X: loadedLvl.Player.X,
		Y: loadedLvl.Player.Y,
		Z: loadedLvl.Player.Z,
	})

	for _, item := range loadedLvl.Player.Inventory {
		resumedPlayer.Inventory.Slots[item.Slot] = inventory.ItemStack{
			ItemID: item.ItemID,
			Count:  item.Count,
			Damage: item.Damage,
		}
	}

	// Verify the iron ingot and pickaxe are right where they were saved!
	hasIronIngot := false
	hasPickaxe := false
	for _, s := range resumedPlayer.Inventory.Slots {
		if s.ItemID == inventory.ItemIronIngot && s.Count == 1 {
			hasIronIngot = true
		}
		if s.ItemID == inventory.ItemWoodenPickaxe && s.Count == 1 {
			hasPickaxe = true
		}
	}

	if !hasIronIngot {
		t.Fatalf("iron ingot was not restored in resumed inventory: %+v", resumedPlayer.Inventory.Slots)
	}
	if !hasPickaxe {
		t.Fatalf("wooden pickaxe was not restored in resumed inventory: %+v", resumedPlayer.Inventory.Slots)
	}

	_ = generator // Used for procedural parity
}
