package sim

import (
	"testing"

	"github.com/HUA503/minecraft-go/pkg/inventory"
	"github.com/HUA503/minecraft-go/pkg/world"
)

func TestCactusFlowerPlacementAndSupportBreak(t *testing.T) {
	engine := NewEngine()
	player := engine.AddPlayer("cactus-flower-test", [16]byte{}, Vec3{X: 8.5, Y: 64, Z: 8.5})
	pos := world.BlockPos{X: 0, Y: 65, Z: 0}
	engine.SetBlock(0, 64, 0, 81) // Cactus support is explicitly valid.
	if !engine.PlaceBlock(player.ID, pos, inventory.ItemCactusFlower) {
		t.Fatal("cactus flower placement on cactus failed")
	}
	if got := engine.BreakBlock(player.ID, world.BlockPos{X: 0, Y: 64, Z: 0}); got != 81 {
		t.Fatalf("breaking cactus support dropped %d, want cactus block 81", got)
	}
	if got := engine.GetBlock(0, 65, 0); got != 0 {
		t.Fatalf("unsupported cactus flower remained as block %d", got)
	}
	count := 0
	for _, stack := range player.Inventory.Slots {
		if stack.ItemID == inventory.ItemCactusFlower {
			count += stack.Count
		}
	}
	if count != 1 {
		t.Fatalf("support break yielded %d cactus flowers, want 1", count)
	}
}

func TestBushDropRequiresShears(t *testing.T) {
	for _, tc := range []struct {
		name   string
		toolID int32
		want   int32
	}{
		{"hand", inventory.ItemAir, inventory.ItemAir},
		{"shears", inventory.ItemShears, inventory.ItemBush},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := NewEngine()
			pos := world.BlockPos{X: 0, Y: 64, Z: 0}
			engine.SetBlock(0, 63, 0, inventory.ItemGrass)
			engine.SetBlock(pos.X, pos.Y, pos.Z, inventory.ItemBush)
			got, _ := engine.BreakBlockWithTool(0, pos, tc.toolID)
			if got != tc.want {
				t.Fatalf("bush drop with %s = %d, want %d", tc.name, got, tc.want)
			}
		})
	}
}
