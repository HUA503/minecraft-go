package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestSmallFlowerPlacementRequiresFlowerSoil(t *testing.T) {
	engine := NewEngine()
	engine.SetBlock(0, 63, 0, inventory.ItemGrass)
	engine.SetBlock(2, 63, 0, inventory.ItemStone)
	flower := inventory.FirstSmallFlower
	if !engine.PlaceBlock(0, world.BlockPos{X: 0, Y: 64, Z: 0}, flower) {
		t.Fatal("flower placement on grass block failed")
	}
	if engine.PlaceBlock(0, world.BlockPos{X: 2, Y: 64, Z: 0}, flower) {
		t.Fatal("flower placement on stone succeeded")
	}
	if got := engine.GetBlock(0, 64, 0); got != flower {
		t.Fatalf("placed flower state = %d, want %d", got, flower)
	}
}

func TestMushroomsPlaceAndDropAsNaturalPlants(t *testing.T) {
	plants := []struct {
		id        int32
		supportID int32
	}{
		{inventory.ItemBrownMushroom, inventory.ItemGrass},
		{inventory.ItemRedMushroom, inventory.ItemGrass},
		{inventory.ItemCrimsonFungus, mustPlantBlockID(t, "minecraft:crimson_nylium")},
		{inventory.ItemWarpedFungus, mustPlantBlockID(t, "minecraft:warped_nylium")},
		{inventory.ItemCrimsonRoots, 87},
		{inventory.ItemWarpedRoots, mustPlantBlockID(t, "minecraft:soul_soil")},
		{inventory.ItemNetherSprouts, mustPlantBlockID(t, "minecraft:warped_nylium")},
	}
	for _, plant := range plants {
		engine := NewEngine()
		engine.SetBlock(0, 63, 0, plant.supportID)
		pos := world.BlockPos{X: 0, Y: 64, Z: 0}
		if !engine.PlaceBlock(0, pos, plant.id) {
			t.Fatalf("plant %d placement on block %d failed", plant.id, plant.supportID)
		}
		player := engine.AddPlayer("mushroom-test", [16]byte{}, Vec3{X: 8.5, Y: 64, Z: 8.5})
		if player == nil {
			t.Fatal("failed to add player")
		}
		if got := engine.BreakBlock(player.ID, pos); got != plant.id {
			t.Fatalf("plant %d break drop = %d", plant.id, got)
		}
	}
}

func mustPlantBlockID(t *testing.T, name string) int32 {
	t.Helper()
	block, ok := inventory.ExtendedBlockByName(name)
	if !ok {
		t.Fatalf("missing support block %s", name)
	}
	return block.RuntimeID
}

func TestBreakingFlowerSupportDropsFlower(t *testing.T) {
	engine := NewEngine()
	player := engine.AddPlayer("flower-test", [16]byte{}, Vec3{X: 8.5, Y: 64, Z: 8.5})
	if player == nil {
		t.Fatal("failed to add player")
	}
	engine.SetBlock(0, 63, 0, inventory.ItemGrass)
	flower := inventory.FirstSmallFlower
	engine.SetBlock(0, 64, 0, flower)
	if got := engine.BreakBlock(player.ID, world.BlockPos{X: 0, Y: 63, Z: 0}); got != inventory.ItemDirt {
		t.Fatalf("support break drop = %d, want dirt (%d)", got, inventory.ItemDirt)
	}
	if got := engine.GetBlock(0, 64, 0); got != 0 {
		t.Fatalf("unsupported flower remained in world with block id %d", got)
	}
	flowerCount := 0
	for _, stack := range player.Inventory.Slots {
		if stack.ItemID == flower {
			flowerCount += stack.Count
		}
	}
	if flowerCount != 1 {
		t.Fatalf("support break returned %d flower items, want 1", flowerCount)
	}
}

func TestWitherRoseContactAppliesWitherAndDamagesPlayer(t *testing.T) {
	engine := NewEngine()
	player := engine.AddPlayer("wither-rose-test", [16]byte{}, Vec3{X: 0.5, Y: 64, Z: 0.5})
	if player == nil {
		t.Fatal("failed to add player")
	}
	engine.SetBlock(0, 63, 0, inventory.ItemGrass)
	engine.SetBlock(0, 64, 0, inventory.ItemWitherRose)
	engine.Tick()
	if activeEffectIndex(player, inventory.EffectWither) < 0 {
		t.Fatal("touching a wither rose did not apply the wither effect")
	}
	engine.Tick()
	if player.Health != 19 {
		t.Fatalf("withering damage left player at %.1f health, want 19", player.Health)
	}
}

func TestWitherRoseDamagesNonUndeadMobsButNotUndead(t *testing.T) {
	engine := NewEngine()
	engine.SetBlock(0, 63, 0, inventory.ItemGrass)
	engine.SetBlock(0, 64, 0, inventory.ItemWitherRose)
	cow := engine.SpawnMob(MobCow, Vec3{X: 0.5, Y: 64, Z: 0.5})
	zombie := engine.SpawnMob(MobZombie, Vec3{X: 0.5, Y: 64, Z: 0.5})
	engine.Tick()
	if cow.Health != 9 {
		t.Fatalf("cow health after touching wither rose = %.1f, want 9", cow.Health)
	}
	if zombie.Health != 20 {
		t.Fatalf("undead zombie health after touching wither rose = %.1f, want 20", zombie.Health)
	}
}
