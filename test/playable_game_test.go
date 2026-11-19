package test

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/render"
	"github.com/HUA503/minecraft-go/pkg/sim"
	"github.com/HUA503/minecraft-go/pkg/world"
	"github.com/HUA503/minecraft-go/pkg/world/gen"
)

// TestFullPlayableSurvivalLoop verifies the entire gameplay loop:
// World gen -> Player spawn -> Movement -> Mining & drops -> Crafting -> Building -> 3D Rendering -> Damage.
func TestFullPlayableSurvivalLoop(t *testing.T) {
	// 1. World Generation
	generator := gen.NewGenerator(98765, gen.DimensionOverworld)
	chunk := generator.GenerateChunk(world.ChunkPos{X: 0, Z: 0})

	engine := sim.NewEngine()
	engine.LoadChunk(world.ChunkPos{X: 0, Z: 0}, chunk)
	engine.StartTickLoop()
	defer engine.Stop()

	// 2. Player Spawn with 20 HP, 20 Hunger
	player := engine.AddPlayer("Alex", [16]byte{0xAA, 0xBB}, sim.Vec3{X: 8.0, Y: 65.0, Z: 8.0})
	if player.Health != 20.0 || player.Hunger != 20 {
		t.Fatalf("initial player stats invalid: Health=%.1f, Hunger=%d", player.Health, player.Hunger)
	}

	// 3. Movement
	engine.MovePlayer(player.ID, sim.Vec3{X: 9.0, Y: 65.0, Z: 8.0}, 0, 0, true)
	if player.Position.X != 9.0 {
		t.Fatalf("player did not move: X=%.2f", player.Position.X)
	}

	// 4. Mining and Drops
	// Set stone block in front of player (+Z direction) at (9, 65, 11)
	engine.SetBlock(9, 65, 11, 1) // Stone
	drop := engine.BreakBlock(player.ID, world.BlockPos{X: 9, Y: 65, Z: 11})
	if drop != inventory.ItemCobblestone {
		t.Fatalf("expected cobblestone drop from stone, got %d", drop)
	}
	if engine.GetBlock(9, 65, 11) != 0 {
		t.Fatalf("mined block should become Air (0)")
	}
	if player.Inventory.Slots[0].ItemID != inventory.ItemCobblestone || player.Inventory.Slots[0].Count != 1 {
		t.Fatalf("drop was not placed in player inventory: %+v", player.Inventory.Slots[0])
	}

	// 5. Crafting
	cm := inventory.NewCraftingManager()
	// Craft 1 Oak Log -> 4 Oak Planks
	craftGrid := []int32{
		inventory.ItemOakLog, inventory.ItemAir,
		inventory.ItemAir, inventory.ItemAir,
	}
	output, ok := cm.MatchCrafting(craftGrid, 2)
	if !ok || output.ItemID != inventory.ItemOakPlanks || output.Count != 4 {
		t.Fatalf("crafting oak planks failed: %+v (ok=%v)", output, ok)
	}
	player.Inventory.AddItem(output.ItemID, output.Count)

	// 6. Block Placement
	player.Inventory.SelectedHotbar = 1 // slot 1 holds oak planks
	placed := engine.PlaceBlock(player.ID, world.BlockPos{X: 9, Y: 65, Z: 11}, inventory.ItemOakPlanks)
	if !placed {
		t.Fatalf("failed to place oak plank at (9, 65, 11)")
	}
	if engine.GetBlock(9, 65, 11) != inventory.ItemOakPlanks {
		t.Fatalf("placed block not found in world: got %d", engine.GetBlock(9, 65, 11))
	}

	// 7. 3D Camera Projection & Rendering
	cam := render.NewCamera(player.Position.X, player.Position.Y+1.62, player.Position.Z)
	fb := render.NewFramebuffer(640, 480)
	fb.Clear(0xFF87CEEB) // Sky blue

	// Project placed block point (9, 65, 11) into screen space
	sp, visible := cam.ProjectToScreen(9.0, 65.0, 11.0, fb.Width, fb.Height)
	if !visible {
		t.Fatalf("block placed in front of player should be visible in camera")
	}
	if sp.X <= 0 || sp.X >= float32(fb.Width) || sp.Y <= 0 || sp.Y >= float32(fb.Height) {
		t.Fatalf("projected screen coordinate out of bounds: (%f, %f)", sp.X, sp.Y)
	}

	// Render HUD (Crosshair, Hearts, Hunger, Hotbar)
	fb.DrawHUD(player.Health, player.Hunger, player.Inventory.SelectedHotbar)
	centerPixel := fb.Color[240*640+320]
	if centerPixel != 0xFFFFFFFF {
		t.Fatalf("crosshair center pixel should be white, got 0x%X", centerPixel)
	}

	// 8. Damage & Respawn
	engine.DamagePlayer(player.ID, 8.0, "fall")
	if player.Health != 12.0 {
		t.Fatalf("player health mismatch after damage: %.1f", player.Health)
	}

	engine.DamagePlayer(player.ID, 15.0, "fatal")
	if player.IsAlive {
		t.Fatalf("player should be dead after lethal damage")
	}

	engine.RespawnPlayer(player.ID, sim.Vec3{X: 8.0, Y: 65.0, Z: 8.0})
	if !player.IsAlive || player.Health != 20.0 {
		t.Fatalf("player should be respawned with 20 HP")
	}

	// 9. Mob Combat & Loot Dropping
	zombie := engine.SpawnMob(sim.MobZombie, sim.Vec3{X: 12.0, Y: 65.0, Z: 8.0})
	drops, killed := engine.DamageMob(zombie.ID, 20.0)
	if !killed || len(drops) == 0 {
		t.Fatalf("zombie should be defeated and drop loot")
	}
	player.Inventory.AddItem(drops[0].ItemID, drops[0].Count)

	// 10. Tool Durability Wear
	tool := &player.Inventory.Slots[player.Inventory.SelectedHotbar]
	tool.ItemID = inventory.ItemDiamondPickaxe
	tool.Count = 1
	tool.Damage = 0
	inventory.ApplyToolWear(tool)
	if tool.Damage != 1 {
		t.Fatalf("tool should have 1 point of durability wear, got %d", tool.Damage)
	}

	// 11. Furnace Smelting Loop
	furnace := inventory.NewFurnaceBlockEntity()
	furnace.InputSlot = inventory.ItemStack{ItemID: 15, Count: 1} // Iron ore
	furnace.FuelSlot = inventory.ItemStack{ItemID: inventory.ItemCoal, Count: 1}
	for i := 0; i < 200; i++ {
		furnace.Tick()
	}
	if furnace.OutputSlot.ItemID != inventory.ItemIronIngot {
		t.Fatalf("furnace should produce iron ingot, got %+v", furnace.OutputSlot)
	}

	// 12. Celestial Day/Night Sky Simulation
	fb.ClearSkyWithTime(18000) // Midnight
	midnightCenter := fb.Color[10*640+320]
	// Midnight sky should be dark navy
	if midnightCenter == 0xFF87CEEB {
		t.Fatalf("midnight sky should not be daytime blue")
	}
}

// TestItemDropAndPickupDelay verifies authentic Minecraft item dropping with initial velocity
// and pickup delay to prevent immediate self-pickup.
func TestItemDropAndPickupDelay(t *testing.T) {
	engine := sim.NewEngine()
	drop := engine.SpawnItemDropWithDelay(inventory.ItemOakPlanks, 5, sim.Vec3{X: 10, Y: 65, Z: 10}, sim.Vec3{X: 0.3, Y: 0.2, Z: 0}, 1.2)
	if drop.PickupDelay != 1.2 {
		t.Fatalf("expected 1.2s pickup delay, got %.2f", drop.PickupDelay)
	}
	if drop.Count != 5 || drop.ItemID != inventory.ItemOakPlanks {
		t.Fatalf("item drop parameters mismatch: %+v", drop)
	}

	drops := engine.GetItemDrops()
	if len(drops) != 1 {
		t.Fatalf("expected 1 item drop in world, got %d", len(drops))
	}
}

// TestWorldTimeCycle verifies standard 24,000 tick day-night cycle and command set.
func TestWorldTimeCycle(t *testing.T) {
	engine := sim.NewEngine()
	if engine.GetWorldTime() != 6000 {
		t.Fatalf("expected default world time 6000 (noon), got %d", engine.GetWorldTime())
	}

	engine.SetWorldTime(1000)
	if engine.GetWorldTime() != 1000 {
		t.Fatalf("expected 1000 after set, got %d", engine.GetWorldTime())
	}

	engine.SetWorldTime(13000)
	if engine.GetWorldTime() != 13000 {
		t.Fatalf("expected 13000 after set, got %d", engine.GetWorldTime())
	}

	// Test 24000 tick wraparound
	engine.SetWorldTime(24500)
	if engine.GetWorldTime() != 500 {
		t.Fatalf("expected 500 after modulo wraparound, got %d", engine.GetWorldTime())
	}
}

// TestReplaceableBlockPlacement verifies blocks can be placed over grass foliage and flowers.
func TestReplaceableBlockPlacement(t *testing.T) {
	engine := sim.NewEngine()
	player := engine.AddPlayer("Tester", [16]byte{}, sim.Vec3{X: 10.0, Y: 65.0, Z: 10.0})

	// Plant tall grass at (15, 65, 15)
	engine.SetBlock(15, 65, 15, 31) // Tall Grass
	if engine.GetBlock(15, 65, 15) != 31 {
		t.Fatalf("expected tall grass at (15, 65, 15)")
	}

	// Place Cobblestone directly over tall grass without mining it first
	ok := engine.PlaceBlock(player.ID, world.BlockPos{X: 15, Y: 65, Z: 15}, inventory.ItemCobblestone)
	if !ok {
		t.Fatalf("placing block over replaceable tall grass should succeed")
	}
	if engine.GetBlock(15, 65, 15) != inventory.ItemCobblestone {
		t.Fatalf("block should now be Cobblestone, got %d", engine.GetBlock(15, 65, 15))
	}
}

// TestCameraPitchSkyDomeRotation verifies the sky dome rotates when looking up vs down.
func TestCameraPitchSkyDomeRotation(t *testing.T) {
	fb := render.NewFramebuffer(320, 240)
	camUp := render.NewCamera(0, 65, 0)
	camUp.Pitch = 60.0 // Looking up at sky dome

	fb.ClearSkyWithCamera(6000, camUp)
	centerColorUp := fb.Color[120*320+160]

	camDown := render.NewCamera(0, 65, 0)
	camDown.Pitch = -45.0 // Looking down at ground

	fb.ClearSkyWithCamera(6000, camDown)
	centerColorDown := fb.Color[120*320+160]

	if centerColorUp == centerColorDown {
		t.Fatalf("sky dome pixel color should differ when camera is pitched up vs down")
	}
}
