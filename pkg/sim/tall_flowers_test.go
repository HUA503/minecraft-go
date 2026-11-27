package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestTallFlowerPlacementAndPairedBreaking(t *testing.T) {
	engine := NewEngine()
	player := engine.AddPlayer("tall-flower-test", [16]byte{}, Vec3{X: 8.5, Y: 64, Z: 8.5})
	if player == nil {
		t.Fatal("failed to add player")
	}
	base := world.BlockPos{X: 0, Y: 64, Z: 0}
	engine.SetBlock(base.X, base.Y-1, base.Z, inventory.ItemGrass)
	flower := inventory.FirstTallFlower
	upper, _ := inventory.TallFlowerBlockState(flower, true)
	if !engine.PlaceBlock(player.ID, base, flower) {
		t.Fatal("placing tall flower on grass failed")
	}
	if got := engine.GetBlock(base.X, base.Y, base.Z); got != flower {
		t.Fatalf("lower half = %d, want %d", got, flower)
	}
	if got := engine.GetBlock(base.X, base.Y+1, base.Z); got != upper {
		t.Fatalf("upper half = %d, want %d", got, upper)
	}
	if got := engine.BreakBlock(player.ID, world.BlockPos{X: base.X, Y: base.Y + 1, Z: base.Z}); got != flower {
		t.Fatalf("breaking upper half dropped %d, want item %d", got, flower)
	}
	if engine.GetBlock(base.X, base.Y, base.Z) != 0 || engine.GetBlock(base.X, base.Y+1, base.Z) != 0 {
		t.Fatal("breaking one half left the other half in the world")
	}
}

func TestBreakingTallFlowerSupportRemovesBothHalves(t *testing.T) {
	engine := NewEngine()
	player := engine.AddPlayer("tall-flower-support-test", [16]byte{}, Vec3{X: 8.5, Y: 64, Z: 8.5})
	base := world.BlockPos{X: 0, Y: 64, Z: 0}
	flower := inventory.FirstTallFlower
	upper, _ := inventory.TallFlowerBlockState(flower, true)
	engine.SetBlock(base.X, base.Y-1, base.Z, inventory.ItemGrass)
	engine.SetBlock(base.X, base.Y, base.Z, flower)
	engine.SetBlock(base.X, base.Y+1, base.Z, upper)

	engine.BreakBlock(player.ID, world.BlockPos{X: base.X, Y: base.Y - 1, Z: base.Z})
	if engine.GetBlock(base.X, base.Y, base.Z) != 0 || engine.GetBlock(base.X, base.Y+1, base.Z) != 0 {
		t.Fatal("breaking support left a tall flower half in the world")
	}
	count := 0
	for _, stack := range player.Inventory.Slots {
		if stack.ItemID == flower {
			count += stack.Count
		}
	}
	if count != 1 {
		t.Fatalf("support break yielded %d tall flower items, want 1", count)
	}
}
